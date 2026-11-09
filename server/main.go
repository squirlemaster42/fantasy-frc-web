package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"strconv"
	"os"
	"os/signal"
	"server/assert"
	"server/authentication"
	"server/background"
	"server/cache"
	"server/config"
	"server/database"
	"server/discord"
	"server/draft"
	"server/handler"
	"server/log"
	"server/metrics"
	"server/model"
	"server/picking"
	"server/scorer"
	"server/tbaHandler"
	"server/utils"
	"syscall"

	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

type mainConfig struct {
	AllowedOrigin               string        `env:"ALLOWED_ORIGIN" default:""`
	RedisAddr                   string        `env:"REDIS_ADDR" default:""`
	RedisPassword               string        `env:"REDIS_PASSWORD" default:""`
	UsernameAllowedSpecialChars string        `env:"USERNAME_ALLOWED_SPECIAL_CHARS" default:"_-"`
	ServerPort                  int           `env:"SERVER_PORT" required:"true"`
	MinPasswordLength           int           `env:"MIN_PASSWORD_LENGTH" default:"12"`
	MinUsernameLength           int           `env:"MIN_USERNAME_LENGTH" default:"3"`
	MaxUsernameLength           int           `env:"MAX_USERNAME_LENGTH" default:"32"`
	BcryptCost                  int           `env:"BCRYPT_COST" default:"14"`
	RedisRateLimitDB            int           `env:"REDIS_RATE_LIMIT_DB" default:"1"`
	RedisAvatarDB               int           `env:"REDIS_AVATAR_DB" default:"2"`
	PostsPerMinute              int64         `env:"RATE_LIMIT_POSTS_PER_MINUTE" default:"100"`
	RateLimitEnabled            bool          `env:"RATE_LIMIT_ENABLED" default:"true"`
	SecureHttpCookie            bool          `env:"SECURE_HTTP_COOKIE" default:"true"`
	TrustProxy                  bool          `env:"TRUST_PROXY" default:"false"`
	DraftActorCacheSize         int           `env:"DRAFT_ACTOR_CACHE_SIZE" default:"128"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	assert := assert.CreateAssertWithContext("Main")

	skipScoring := flag.Bool("skipScoring", false, "When true is entered, the scorer will not be started")
	verbose := flag.Bool("v", false, "Enable debug logging")
	logFormat := flag.String("log-format", "json", "Log format: json or text")
	flag.Parse()

	log.SetupLogger(*logFormat)

	if *verbose {
		log.SetLevel(log.LevelDebug)
	}

	log.Info(ctx, "-------- Starting Fantasy FRC --------")

	err := godotenv.Load()
	if err != nil {
		log.Info(ctx, "No .env file loaded, using environment variables")
	}

	var cfg mainConfig
	if err := config.Load(&cfg); err != nil {
		log.Fatal(ctx, "invalid configuration", "error", err)
	}

	tbaTok := requireEnv(ctx, "TBA_TOKEN")
	dbPassword := requireEnv(ctx, "DB_PASSWORD")
	dbUsername := requireEnv(ctx, "DB_USERNAME")
	dbIp := requireEnv(ctx, "DB_IP")
	dbName := requireEnv(ctx, "DB_NAME")
	tbaWebhookSecret := requireEnv(ctx, "TBA_WEBHOOK_SECRET")
	metricSecret := requireEnv(ctx, "METRIC_SECRET")
	csrfSecret := requireEnv(ctx, "CSRF_SECRET")

	if cfg.ServerPort < 1 || cfg.ServerPort > 65535 {
		log.Fatal(ctx, "SERVER_PORT must be between 1 and 65535", "value", cfg.ServerPort)
	}

	if cfg.BcryptCost < bcrypt.MinCost || cfg.BcryptCost > bcrypt.MaxCost {
		log.Warn(ctx, "BCRYPT_COST out of range, using default", "value", cfg.BcryptCost, "default", 14)
		cfg.BcryptCost = 14
	}

	log.Info(ctx, "Trust proxy setting", "TRUST_PROXY", cfg.TrustProxy)

	if cfg.TrustProxy && cfg.AllowedOrigin == "" {
		log.Fatal(ctx, "ALLOWED_ORIGIN environment variable is required when TRUST_PROXY is true")
	}

	if cfg.DraftActorCacheSize < 1 {
		log.Fatal(ctx, "DRAFT_ACTOR_CACHE_SIZE must be at least 1", "value", cfg.DraftActorCacheSize)
	}

	if cfg.RedisAddr == "" {
		cfg.RedisAddr = "localhost:6379"
	}

	log.Info(ctx, "Extracted Env Vars")
	db, err := database.RegisterDatabaseConnection(ctx, dbUsername, dbPassword, dbIp, dbName)
	if err != nil {
		log.Error(ctx, "Failed to register database connection", "error", err)
		os.Exit(1)
	}
	log.Info(ctx, "Registered Database Connection")

	tbaHandler := tbaHandler.NewHandler(tbaTok, db)
	if cfg.AllowedOrigin != "" {
		log.Info(ctx, "WebSocket origin validation configured", "ALLOWED_ORIGIN", cfg.AllowedOrigin)
	} else {
		log.Info(ctx, "WebSocket origin validation using development fallback (localhost/same-origin)")
	}

	discordWebhookBus := discord.NewBus()
	draftStore := model.NewSQLDraftStore(db)
	userStore := model.NewSQLUserStore(db)

	passwordHasher, err := authentication.NewBcryptPasswordHasher(cfg.BcryptCost)
	if err != nil {
		assert.NoError(ctx, err, "Failed to create password hasher")
	}

	authService := authentication.NewAuthService(userStore, passwordHasher, authentication.AuthConfig{
		MinPasswordLength:           cfg.MinPasswordLength,
		MinUsernameLength:           cfg.MinUsernameLength,
		MaxUsernameLength:           cfg.MaxUsernameLength,
		UsernameAllowedSpecialChars: cfg.UsernameAllowedSpecialChars,
	})

	teamStore := model.NewSQLTeamStore(db)
	discordStore := model.NewSQLDiscordStore(db)
	matchStore := model.NewSQLMatchStore(db)
	matchTeamStore := model.NewSQLMatchTeamStore(db)

	pickNotifier := &picking.PickNotifier{
		Watchers: make(map[int][]picking.Watcher),
	}

	pickConfig, err := utils.LoadPickWindowConfigFromEnv()
	if err != nil {
		assert.NoError(ctx, err, "Failed to load pick window configuration")
	}

	pickCacheConfig := cache.EventTeamCacheConfigFromDefaults()
	eventTeamCache, err := cache.NewEventTeamCache(ctx, tbaHandler, cfg.RedisAddr, cfg.RedisPassword, pickCacheConfig)
	assert.NoError(ctx, err, "Failed to create event team cache")
	if err := eventTeamCache.Refresh(ctx); err != nil {
		log.Warn(ctx, "Initial pick validation cache refresh failed, will retry in background", "error", err)
	}
	cache.StartEventTeamCacheRefreshLoop(ctx, eventTeamCache, pickCacheConfig.RefreshInterval)

	draftActorMap := draft.NewDraftActorMap(draftStore, tbaHandler, discordStore, discordWebhookBus, pickNotifier, pickConfig, cfg.DraftActorCacheSize, eventTeamCache)
	// Start the draft daemon and add all running drafts to it
	draftDaemon := background.NewDraftDaemon(draftStore, draftActorMap)
	err = draftDaemon.Start(ctx)
	if err != nil {
		log.Warn(ctx, "Failed to start draft daemon", "error", err)
		panic("failed to start draft manager")
	}

	log.Debug(ctx, "Checking for drafts that need to be added to daemon")
	drafts, err := draftStore.GetDraftsInStatus(ctx, model.PICKING)
	if err != nil {
		log.Warn(ctx, "Could not get any drafts in picking status", "error", err)
	} else {
		for _, draftId := range drafts {
			err = draftDaemon.AddDraft(ctx, draftId)
			if err != nil {
				log.Warn(ctx, "Failed to add draft to manager in init", "error", err)
			}
		}
	}

	scorer := scorer.NewScorer(tbaHandler, matchStore, matchTeamStore, teamStore)
	var waitScorer <-chan struct{}
	if !*skipScoring {
		log.Info(ctx, "Started Scorer")
		waitScorer = scorer.RunScorer(ctx)
	}

	cleanupService := background.NewCleanupService(db, background.CleanupIntervalMinutes())
	err = cleanupService.Start(ctx)
	if err != nil {
		log.Error(ctx, "Failed to start cleanup service", "error", err)
	}

	avatarStore, err := cache.NewAvatarStore(ctx, tbaHandler, cfg.RedisAddr, cfg.RedisPassword, cfg.RedisAvatarDB)
	assert.NoError(ctx, err, "Failed to create avatar store")

	handler := handler.Handler{
		Stores: handler.StorageGroup{
			DraftStore: draftStore,
			UserStore:  userStore,
			TeamStore:  teamStore,
		},
		Services: handler.ServiceGroup{
			AuthService:       authService,
			TBAHandler:        tbaHandler,
			DraftActorMap:     draftActorMap,
			DraftDaemon:       draftDaemon,
			Scorer:            scorer,
			AvatarStore:       &avatarStore,
			EventTeamCache:    eventTeamCache,
			DiscordWebhookBus: discordWebhookBus,
		},
		Config: handler.ConfigGroup{
			SecureHttpCookie:            cfg.SecureHttpCookie,
			MinPasswordLength:           cfg.MinPasswordLength,
			MinUsernameLength:           cfg.MinUsernameLength,
			MaxUsernameLength:           cfg.MaxUsernameLength,
			UsernameAllowedSpecialChars: cfg.UsernameAllowedSpecialChars,
			AllowedOrigin:               cfg.AllowedOrigin,
		},
	}

	// Load the tba webhook secret
	file, err := os.Open(utils.GetWebhookFilePath())
	if err != nil {
		log.Warn(ctx, "Unable to open tba webhook secret file", "error", err)
	} else {
		defer func() { _ = file.Close() }()
		body, err := io.ReadAll(file)
		if err != nil {
			log.Warn(ctx, "Failed to read tba webhook file body", "error", err)
		} else {
			handler.Config.TbaVerificationCode = string(body)
		}
	}
	handler.Config.TbaWebhookSecret = tbaWebhookSecret

	serverPort := strconv.Itoa(cfg.ServerPort)

	app, otelShutdown := CreateServer(ctx, ServerConfig{
		ServerPort:       serverPort,
		Handler:          handler,
		Database:         db,
		MetricSecret:     metricSecret,
		CsrfSecret:       csrfSecret,
		RedisAddr:        cfg.RedisAddr,
		RedisPassword:    cfg.RedisPassword,
		RedisRateLimitDB: cfg.RedisRateLimitDB,
		PostsPerMinute:   cfg.PostsPerMinute,
		RateLimitEnabled: cfg.RateLimitEnabled,
		TrustProxy:       cfg.TrustProxy,
		AllowedOrigin:    cfg.AllowedOrigin,
	})

	httpServer := &http.Server{
		Addr:              ":" + serverPort,
		Handler:           app,
		ReadHeaderTimeout: ServerReadHeaderTimeout(),
	}

	go func() {
		// Use net/http directly so we can configure server-level timeouts such as ReadHeaderTimeout.
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			assert.NoError(ctx, err, "Failed to start server")
		}
	}()

	// Wait for shutdown signal
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	<-shutdownChan

	log.Info(ctx, "Shutting down gracefully...")
	cancel()

	if waitScorer != nil {
		<-waitScorer
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), ServerShutdownTimeout())
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Warn(ctx, "Failed to shutdown server gracefully", "error", err)
	}
	if err := otelShutdown(shutdownCtx); err != nil {
		log.Warn(ctx, "Failed to shutdown OpenTelemetry tracer", "error", err)
	}
	metrics.ShutdownMetrics()
	if err := cleanupService.Stop(ctx); err != nil {
		log.Warn(ctx, "Failed to stop cleanup service", "error", err)
	}
	if err := cache.CloseEventTeamCache(eventTeamCache); err != nil {
		log.Warn(ctx, "Failed to close event team cache", "error", err)
	}
	database.CloseCachedStatements(shutdownCtx, db)
	if err := db.Close(); err != nil {
		log.Error(ctx, "Failed to close database connection", "error", err)
	}
	cancel()
}

func requireEnv(ctx context.Context, key string) string {
	val, err := utils.RequireEnv(key)
	if err != nil {
		log.Fatal(ctx, "missing required environment variable", "key", key)
	}
	return val
}
