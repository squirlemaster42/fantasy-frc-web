package main

import (
	"net/http"
	"server/assert"
	"server/assets"
	"server/authentication"
	"server/handler"
	"server/log"
	"server/metrics"
	"server/middleware"

	"github.com/getsentry/sentry-go"
	sentryecho "github.com/getsentry/sentry-go/echo"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func CreateServer(serverPort string, h handler.Handler, sentryDNS string, metricSecret string) {
	log.InfoNoContext("Starting Server")
	assert := assert.CreateAssertWithContext("Create Server")
	auth := authentication.NewAuth(h.Database)
	app := echo.New()
	app.IPExtractor = echo.ExtractIPDirect()

	// To initialize Sentry's handler, you need to initialize Sentry itself beforehand
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              sentryDNS,
		EnableLogs:       true,
		TracesSampleRate: 1.0,
	}); err != nil {
		log.ErrorNoContext("Sentry initialize failed", "Error", err)
	}

	metrics.InitMetrics(h.Database)

	cacheControlMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Cache-Control", "public, max-age=2592000")
			return next(c)
		}
	}

	app.Add(
		http.MethodGet,
		"/css/*",
		echo.StaticDirectoryHandler(assets.CSS(), false),
		cacheControlMiddleware,
	)

	app.Add(
		http.MethodGet,
		"/img/*",
		echo.StaticDirectoryHandler(assets.Img(), false),
		cacheControlMiddleware,
	)

	//app.Use(echomiddleware.Recover())
	app.Use(middleware.CorrelationID())
	app.Use(sentryecho.New(sentryecho.Options{
		Repanic: true,
	}))
	app.Use(metrics.MetricsMiddleware())

	//Setup Routes
	app.GET("/", h.HandleViewLanding, echomiddleware.Gzip())
	app.GET("/login", h.HandleViewLogin, echomiddleware.Gzip())
	app.POST("/login", h.HandleLoginPost, echomiddleware.Gzip())
	app.GET("/register", h.HandleViewRegister, echomiddleware.Gzip())
	app.POST("/register", h.HandlerRegisterPost, echomiddleware.Gzip())
	app.POST("/logout", h.HandleLogoutPost, echomiddleware.Gzip())
	app.POST("/tbaWebhook", h.ConsumeTbaWebhook, echomiddleware.Gzip())

	metricAuth := authentication.NewMetricAuth(metricSecret)
	app.GET("/metrics", echo.WrapHandler(promhttp.Handler()), metricAuth.MetricsAuthMiddleware())
	app.GET("/logs", h.HandleLogs, metricAuth.MetricsAuthMiddleware())

	protected := app.Group("/u", auth.Authenticate)
	protected.Use(echomiddleware.Gzip())
	protected.GET("/home", h.HandleViewHome)
	protected.GET("/createDraft", h.HandleViewCreateDraft)
	protected.POST("/createDraft", h.HandleCreateDraftPost)
	protected.GET("/draft/:id/profile", h.HandleViewDraftProfile)
	protected.POST("/draft/:id/updateDraft", h.HandleUpdateDraftProfile)
	protected.POST("/draft/:id/startDraft", h.HandleStartDraft)
	protected.GET("/draft/:id/pick", h.ServePickPage)
	protected.POST("/draft/:id/makePick", h.HandlerPickRequest)
	protected.GET("/draft/:id/pickNotifier", h.PickNotifier)
	protected.POST("/draft/:id/invitePlayer", h.InviteDraftPlayer)
	protected.GET("/team/score", h.HandleTeamScore)
	protected.POST("/team/score", h.HandleGetTeamScore)
	protected.GET("/draft/:id/draftScore", h.HandleDraftScore)
	protected.GET("/draft/:id/team/:teamNumber", h.HandleDraftTeamScore)
	protected.GET("/draft/:id/admin", h.HandleDraftAdminGet)
	protected.POST("/draft/:id/admin/skipPick", h.HandleAdminSkipPick)
	protected.POST("/draft/:id/admin/extendTime", h.HandleAdminExtendTime)
	protected.POST("/draft/:id/admin/makePick", h.HandleAdminMakePick)
	protected.POST("/draft/:id/admin/undoPick", h.HandleAdminUndoPick)
	protected.POST("/searchPlayers", h.SearchPlayers)
	protected.GET("/viewInvites", h.HandleViewInvites)
	protected.POST("/acceptInvite", h.HandleAcceptInvite)
	protected.POST("/draft/:id/skipPickToggle", h.HandleSkipPickToggle)
	protected.GET("/team/:id/avatar", h.GetTeamAvatar)
	protected.GET("/userProfile", h.HandleViewUserProfile)
	protected.POST("/userProfile", h.HandleUpdateUserProfile)

	admin := protected.Group("/admin", auth.CheckAdmin)
	admin.GET("/console", h.HandleAdminConsoleGet)
	admin.POST("/processCommand", h.HandleRunCommand)

	err := app.Start(":" + serverPort)
	assert.NoError(err, "Failed to start server")
}
