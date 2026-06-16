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

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	otelecho "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

type ServerConfig struct {
	ServerPort       string
	Handler          handler.Handler
	Database         *sql.DB
	MetricSecret     string
	CsrfSecret       string
	RedisAddr        string
	RedisPassword    string
	RedisRateLimitDB int
	PostsPerMinute   int64
	RateLimitEnabled bool
	TrustProxy       bool
	AllowedOrigin    string
}

func CreateServer(ctx context.Context, cfg ServerConfig) (*echo.Echo, func(context.Context) error) {
	log.Info(ctx, "Starting Server")
	auth := authentication.NewAuth(cfg.Handler.UserStore)
	app := echo.New()
	if cfg.TrustProxy {
		app.IPExtractor = echo.ExtractIPFromXFFHeader(echo.TrustLoopback(true))
		log.Info(ctx, "IP extractor configured to trust proxy (X-Forwarded-For)")
	} else {
		app.IPExtractor = echo.ExtractIPDirect()
		log.Info(ctx, "IP extractor configured for direct access (no proxy)")
	}

	// Initialize OpenTelemetry
	shutdown := otel.InitTracer("fantasy-frc-web")

	if err := metrics.InitMetrics(ctx, cfg.Database); err != nil {
		log.Warn(ctx, "Failed to initialize metrics", "error", err)
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
	app.Use(middleware.SecurityHeaders(cfg.Handler.SecureHttpCookie))

	csrf := middleware.NewCSRF(cfg.CsrfSecret, cfg.Handler.SecureHttpCookie)
	var rateLimiter *middleware.RateLimiter
	if cfg.RateLimitEnabled {
		rateLimiter = middleware.NewRateLimiter(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisRateLimitDB)
	}

	loginPostMiddleware := []echo.MiddlewareFunc{echomiddleware.Gzip()}
	registerPostMiddleware := []echo.MiddlewareFunc{echomiddleware.Gzip()}
	if rateLimiter != nil {
		app.Use(rateLimiter.RateLimitGeneral(cfg.PostsPerMinute))
		loginPostMiddleware = append(loginPostMiddleware, rateLimiter.RateLimitLogin())
		registerPostMiddleware = append(registerPostMiddleware, rateLimiter.RateLimitRegister())
	}

	//Setup Routes
	app.GET("/", cfg.Handler.HandleViewLanding, echomiddleware.Gzip())
	app.GET("/login", cfg.Handler.HandleViewLogin, echomiddleware.Gzip())
	app.POST("/login", cfg.Handler.HandleLoginPost, loginPostMiddleware...)
	app.GET("/register", cfg.Handler.HandleViewRegister, echomiddleware.Gzip())
	app.POST("/register", cfg.Handler.HandlerRegisterPost, registerPostMiddleware...)
	app.POST("/tbaWebhook", cfg.Handler.ConsumeTbaWebhook, echomiddleware.Gzip())

	metricAuth := authentication.NewMetricAuth(cfg.MetricSecret)
	app.GET("/metrics", echo.WrapHandler(promhttp.Handler()), metricAuth.MetricsAuthMiddleware())

	app.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	protected := app.Group("/u", auth.Authenticate, csrf.CSRF())
	protected.Use(echomiddleware.Gzip())
	protected.POST("/logout", cfg.Handler.HandleLogoutPost)
	protected.GET("/home", cfg.Handler.HandleViewHome)
	protected.GET("/createDraft", cfg.Handler.HandleViewCreateDraft)
	protected.POST("/createDraft", cfg.Handler.HandleCreateDraftPost)
	protected.GET("/draft/:id/profile", cfg.Handler.HandleViewDraftProfile)
	protected.POST("/draft/:id/updateDraft", cfg.Handler.HandleUpdateDraftProfile)
	protected.POST("/draft/:id/startDraft", cfg.Handler.HandleStartDraft)
	protected.GET("/draft/:id/pick", cfg.Handler.ServePickPage)
	protected.POST("/draft/:id/makePick", cfg.Handler.HandlerPickRequest)
	protected.GET("/draft/:id/pickNotifier", cfg.Handler.PickNotifier)
	protected.POST("/draft/:id/invitePlayer", cfg.Handler.InviteDraftPlayer)
	protected.GET("/team/score", cfg.Handler.HandleTeamScore)
	protected.POST("/team/score", cfg.Handler.HandleGetTeamScore)
	protected.GET("/draft/:id/draftScore", cfg.Handler.HandleDraftScore)
	protected.GET("/draft/:id/team/:teamNumber", cfg.Handler.HandleDraftTeamScore)
	protected.GET("/draft/:id/admin", cfg.Handler.HandleDraftAdminGet)
	protected.POST("/draft/:id/admin/skipPick", cfg.Handler.HandleAdminSkipPick)
	protected.POST("/draft/:id/admin/extendTime", cfg.Handler.HandleAdminExtendTime)
	protected.POST("/draft/:id/admin/makePick", cfg.Handler.HandleAdminMakePick)
	protected.POST("/draft/:id/admin/undoPick", cfg.Handler.HandleAdminUndoPick)
	protected.POST("/searchPlayers", cfg.Handler.SearchPlayers)
	protected.GET("/viewInvites", cfg.Handler.HandleViewInvites)
	protected.POST("/acceptInvite", cfg.Handler.HandleAcceptInvite)
	protected.POST("/draft/:id/skipPickToggle", cfg.Handler.HandleSkipPickToggle)
	protected.GET("/team/:id/avatar", cfg.Handler.GetTeamAvatar)
	protected.GET("/userProfile", cfg.Handler.HandleViewUserProfile)
	protected.POST("/userProfile", cfg.Handler.HandleUpdateUserProfile)

	admin := protected.Group("/admin", auth.CheckAdmin)
	admin.GET("/console", cfg.Handler.HandleAdminConsoleGet)
	admin.POST("/processCommand", cfg.Handler.HandleRunCommand)

	return app, shutdown
}
