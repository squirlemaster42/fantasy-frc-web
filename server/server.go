package main

import (
	"context"
	"database/sql"
	"net/http"
	"server/assets"
	"server/authentication"
	"server/handler"
	"server/log"
	"server/metrics"
	"server/middleware"
	"server/otel"
	"server/types"
	"server/view/errorpage"

	"github.com/a-h/templ"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelecho "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

func CreateServer(serverPort string, h handler.Handler, database *sql.DB, metricSecret string, csrfSecret string, redisAddr string, redisPassword string, redisRateLimitDB int, postsPerMinute int64) (*echo.Echo, func(context.Context) error) {
	log.Info(context.Background(), "Starting Server")
	auth := authentication.NewAuth(h.UserStore)
	app := echo.New()
	app.IPExtractor = echo.ExtractIPDirect()

	// Custom HTTP error handler: renders templ error pages for 404/403/500
	app.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		code := http.StatusInternalServerError
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
		}

		// Log appropriately based on severity
		switch {
		case code >= 500:
			log.Error(c.Request().Context(), "HTTP error", "code", code, "error", err)
		case code >= 400:
			log.Warn(c.Request().Context(), "HTTP error", "code", code, "error", err)
		}

		// Determine auth context for consistent navbar/footer rendering
		fromProtected := false
		username := ""
		var pageData *types.PageData
		if userUuidVal := c.Get(string(authentication.UserUuidKey)); userUuidVal != nil {
			if userUuid, ok := userUuidVal.(uuid.UUID); ok {
				fromProtected = true
				name, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
				if err == nil {
					username = name
				}
			}
		}

		// Select appropriate error page template
		var page templ.Component
		switch code {
		case http.StatusNotFound:
			page = errorpage.NotFound404(" | Page Not Found", fromProtected, username, pageData)
		case http.StatusForbidden:
			page = errorpage.Forbidden403(" | Access Denied", fromProtected, username, pageData)
		case http.StatusInternalServerError:
			page = errorpage.ServerError500(" | Server Error", fromProtected, username, pageData)
		default:
			// For other status codes, fall back to a generic error page
			page = errorpage.ServerError500(" | Error", fromProtected, username, pageData)
		}

		// Render the templ component; if rendering itself fails, fall back to plain text
		if renderErr := handler.RenderError(c, code, page); renderErr != nil {
			log.Error(c.Request().Context(), "Failed to render error page", "error", renderErr)
			_ = c.String(code, http.StatusText(code))
		}
	}

	// Initialize OpenTelemetry
	shutdown := otel.InitTracer("fantasy-frc-web")

	if err := metrics.InitMetrics(database); err != nil {
		log.Warn(context.Background(), "Failed to initialize metrics", "error", err)
	}

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

	app.Add(
		http.MethodGet,
		"/js/*",
		echo.StaticDirectoryHandler(assets.JS(), false),
		cacheControlMiddleware,
	)

	//app.Use(echomiddleware.Recover())
	app.Use(middleware.CorrelationID())
	app.Use(otelecho.Middleware("fantasy-frc-web"))
	app.Use(metrics.MetricsMiddleware())
	app.Use(middleware.SecurityHeaders(h.SecureHttpCookie))

	csrf := middleware.NewCSRF(csrfSecret, h.SecureHttpCookie)
	rateLimiter := middleware.NewRateLimiter(redisAddr, redisPassword, redisRateLimitDB)

	//Setup Routes
	app.GET("/", h.HandleViewLanding, echomiddleware.Gzip())
	app.GET("/login", h.HandleViewLogin, echomiddleware.Gzip())
	app.POST("/login", h.HandleLoginPost, echomiddleware.Gzip(), rateLimiter.RateLimitLogin())
	app.GET("/register", h.HandleViewRegister, echomiddleware.Gzip())
	app.POST("/register", h.HandlerRegisterPost, echomiddleware.Gzip(), rateLimiter.RateLimitRegister())
	app.POST("/tbaWebhook", h.ConsumeTbaWebhook, echomiddleware.Gzip())

	metricAuth := authentication.NewMetricAuth(metricSecret)
	app.GET("/metrics", echo.WrapHandler(promhttp.Handler()), metricAuth.MetricsAuthMiddleware())

	app.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	protected := app.Group("/u", auth.Authenticate, csrf.CSRF(), rateLimiter.RateLimitGeneral(postsPerMinute))
	protected.Use(echomiddleware.Gzip())
	protected.POST("/logout", h.HandleLogoutPost)
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

	// Catch-all for unmatched routes on all HTTP methods
	app.Any("/*", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusNotFound)
	})

	return app, shutdown
}
