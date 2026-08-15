package main

import (
	"context"
	"database/sql"
	"fmt"
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
	auth := authentication.NewAuth(cfg.Handler.Services.AuthService, cfg.Handler.Stores.UserStore)
	app := echo.New()
	if cfg.TrustProxy {
		app.IPExtractor = echo.ExtractIPFromXFFHeader(echo.TrustLoopback(true))
		log.Info(ctx, "IP extractor configured to trust proxy (X-Forwarded-For)")
	} else {
		app.IPExtractor = echo.ExtractIPDirect()
		log.Info(ctx, "IP extractor configured for direct access (no proxy)")
	}

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
        pageData := types.NewPageData(0, "", false)
		if userUuidVal := c.Get(string(authentication.UserUuidKey)); userUuidVal != nil {
			if userUuid, ok := userUuidVal.(uuid.UUID); ok {
				fromProtected = true
				name, err := cfg.Handler.Stores.UserStore.GetUsername(c.Request().Context(), userUuid)
				if err == nil {
					username = name
				}
			}
		}

		// Select appropriate error page template
		var page templ.Component
		switch code {
		case http.StatusNotFound:
			page = errorpage.NotFound404("Page Not Found", fromProtected, username, pageData)
		case http.StatusForbidden:
			page = errorpage.Forbidden403("Access Denied", fromProtected, username, pageData)
		case http.StatusInternalServerError:
			page = errorpage.ServerError500("Server Error", fromProtected, username, pageData)
		default:
			// For other status codes, fall back to a generic error page
			page = errorpage.ServerError500("Error", fromProtected, username, pageData)
		}

		// Render the templ component; if rendering itself fails, fall back to plain text
		if renderErr := handler.RenderError(c, code, page); renderErr != nil {
			log.Error(c.Request().Context(), "Failed to render error page", "error", renderErr)
			_ = c.String(code, http.StatusText(code))
		}
	}

	// Initialize OpenTelemetry
	shutdown := otel.InitTracer(otelServiceName)

	if err := metrics.InitMetrics(ctx, cfg.Database); err != nil {
		log.Warn(ctx, "Failed to initialize metrics", "error", err)
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
	app.Use(otelecho.Middleware(otelServiceName))
	app.Use(metrics.MetricsMiddleware())
	app.Use(middleware.SecurityHeaders(cfg.Handler.Config.SecureHttpCookie))

	csrf := middleware.NewCSRF(cfg.CsrfSecret, cfg.Handler.Config.SecureHttpCookie)
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
	registerPublicRoutes(app, cfg, auth, loginPostMiddleware, registerPostMiddleware)

	metricAuth := authentication.NewMetricAuth(cfg.MetricSecret)
	registerSystemRoutes(app, cfg, metricAuth)

	protected := app.Group("/u", auth.Authenticate, csrf.CSRF())
	protected.Use(echomiddleware.Gzip())
	registerProtectedRoutes(protected, cfg)

	admin := protected.Group("/admin", auth.CheckAdmin)
	registerAdminRoutes(admin, cfg)

	registerCatchAll(app)

	return app, shutdown
}

func registerPublicRoutes(app *echo.Echo, cfg ServerConfig, auth *authentication.Authenticator, loginMiddleware, registerMiddleware []echo.MiddlewareFunc) {
	app.GET("/", cfg.Handler.HandleViewLanding, echomiddleware.Gzip())
	app.GET("/login", cfg.Handler.HandleViewLogin, auth.RedirectIfAuthenticated, echomiddleware.Gzip())
	app.POST("/login", cfg.Handler.HandleLoginPost, loginMiddleware...)
	app.GET("/register", cfg.Handler.HandleViewRegister, auth.RedirectIfAuthenticated, echomiddleware.Gzip())
	app.POST("/register", cfg.Handler.HandlerRegisterPost, registerMiddleware...)
	app.POST("/tbaWebhook", cfg.Handler.ConsumeTbaWebhook, echomiddleware.Gzip())
}

func registerSystemRoutes(app *echo.Echo, cfg ServerConfig, metricAuth *authentication.MetricAuth) {
	app.GET("/metrics", echo.WrapHandler(promhttp.Handler()), metricAuth.MetricsAuthMiddleware())
	app.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
}

func registerProtectedRoutes(protected *echo.Group, cfg ServerConfig) {
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
	protected.POST("/draft/:id/uninvitePlayer", cfg.Handler.HandleUninvitePlayer)
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
	protected.GET("/leaderboard", cfg.Handler.HandleOverallLeaderboard)
	protected.GET("/viewInvites", cfg.Handler.HandleViewInvites)
	protected.POST("/acceptInvite", cfg.Handler.HandleAcceptInvite)
	protected.POST("/declineInvite", cfg.Handler.HandleDeclineInvite)
	protected.POST("/draft/:id/skipPickToggle", cfg.Handler.HandleSkipPickToggle)
	protected.GET("/team/:id/avatar", cfg.Handler.GetTeamAvatar)
	protected.GET("/userProfile", cfg.Handler.HandleViewUserProfile)
	protected.POST("/userProfile", cfg.Handler.HandleUpdateUserProfile)
}

func registerAdminRoutes(admin *echo.Group, cfg ServerConfig) {
	admin.GET("/console", cfg.Handler.HandleAdminConsoleGet)
	admin.POST("/processCommand", cfg.Handler.HandleRunCommand)
}

func registerCatchAll(app *echo.Echo) {
	app.Any("/*", func(c echo.Context) error {
		return echo.NewHTTPError(http.StatusNotFound)
	})
}

func cacheControlMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", StaticAssetMaxAgeSeconds()))
		return next(c)
	}
}

