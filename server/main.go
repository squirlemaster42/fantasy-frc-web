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

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
	tbaTok := requireEnv(ctx, "TBA_TOKEN")
	dbPassword := requireEnv(ctx, "DB_PASSWORD")
	dbUsername := requireEnv(ctx, "DB_USERNAME")
	dbIp := requireEnv(ctx, "DB_IP")
	dbName := requireEnv(ctx, "DB_NAME")
	tbaWebhookSecret := requireEnv(ctx, "TBA_WEBHOOK_SECRET")
	metricSecret := requireEnv(ctx, "METRIC_SECRET")
	csrfSecret := requireEnv(ctx, "CSRF_SECRET")

	allowedOrigin := utils.MustGetEnvString("ALLOWED_ORIGIN", "")
	redisAddr := utils.MustGetEnvString("REDIS_ADDR", "")
	redisPassword := utils.MustGetEnvString("REDIS_PASSWORD", "")
	usernameAllowedSpecialChars := utils.MustGetEnvString("USERNAME_ALLOWED_SPECIAL_CHARS", "_-")

	serverPortNum := mustGetEnvInt(ctx, "SERVER_PORT", 0)
	if serverPortNum < 1 || serverPortNum > 65535 {
		log.Fatal(ctx, "SERVER_PORT must be between 1 and 65535", "value", serverPortNum)
	}

	minPasswordLength := mustGetEnvInt(ctx, "MIN_PASSWORD_LENGTH", 12)
	minUsernameLength := mustGetEnvInt(ctx, "MIN_USERNAME_LENGTH", 3)
	maxUsernameLength := mustGetEnvInt(ctx, "MAX_USERNAME_LENGTH", 32)

	bcryptCost := mustGetEnvInt(ctx, "BCRYPT_COST", 14)
	if bcryptCost < bcrypt.MinCost || bcryptCost > bcrypt.MaxCost {
		log.Warn(ctx, "BCRYPT_COST out of range, using default", "value", bcryptCost, "default", 14)
		bcryptCost = 14
	}

	redisRateLimitDB := mustGetEnvInt(ctx, "REDIS_RATE_LIMIT_DB", 1)
	redisAvatarDB := mustGetEnvInt(ctx, "REDIS_AVATAR_DB", 2)
	postsPerMinute := mustGetEnvInt64(ctx, "RATE_LIMIT_POSTS_PER_MINUTE", 100)

	rateLimitEnabled := mustGetEnvBool(ctx, "RATE_LIMIT_ENABLED", true)
	secureHttpCookie := mustGetEnvBool(ctx, "SECURE_HTTP_COOKIE", true)
	trustProxy := mustGetEnvBool(ctx, "TRUST_PROXY", false)
	log.Info(ctx, "Trust proxy setting", "TRUST_PROXY", trustProxy)

	if trustProxy && allowedOrigin == "" {
		log.Fatal(ctx, "ALLOWED_ORIGIN environment variable is required when TRUST_PROXY is true")
	}

	draftActorCacheSize := mustGetEnvInt(ctx, "DRAFT_ACTOR_CACHE_SIZE", 128)
	if draftActorCacheSize < 1 {
		log.Fatal(ctx, "DRAFT_ACTOR_CACHE_SIZE must be at least 1", "value", draftActorCacheSize)
	}

	log.Info(ctx, "Extracted Env Vars")
	db, err := database.RegisterDatabaseConnection(ctx, dbUsername, dbPassword, dbIp, dbName)
	if err != nil {
		log.Error(ctx, "Failed to register database connection", "error", err)
		os.Exit(1)
	}
	log.Info(ctx, "Registered Database Connection")

	tbaHandler := tbaHandler.NewHandler(tbaTok, db)
	if allowedOrigin != "" {
		log.Info(ctx, "WebSocket origin validation configured", "ALLOWED_ORIGIN", allowedOrigin)
	} else {
		log.Info(ctx, "WebSocket origin validation using development fallback (localhost/same-origin)")
	}

	discordWebhookBus := discord.NewBus()
	draftStore := model.NewSQLDraftStore(db)
	userStore := model.NewSQLUserStore(db)

	passwordHasher, err := authentication.NewBcryptPasswordHasher(bcryptCost)
	if err != nil {
		assert.NoError(ctx, err, "Failed to create password hasher")
	}

	authService := authentication.NewAuthService(userStore, passwordHasher, authentication.AuthConfig{
		MinPasswordLength:           minPasswordLength,
		MinUsernameLength:           minUsernameLength,
		MaxUsernameLength:           maxUsernameLength,
		UsernameAllowedSpecialChars: usernameAllowedSpecialChars,
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

	draftActorMap := draft.NewDraftActorMap(draftStore, tbaHandler, discordStore, discordWebhookBus, pickNotifier, pickConfig, draftActorCacheSize)
	//Start the draft daemon and add all running drafts to it
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

	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	avatarStore, err := cache.NewAvatarStore(ctx, tbaHandler, redisAddr, redisPassword, redisAvatarDB)
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
			DiscordWebhookBus: discordWebhookBus,
		},
		Config: handler.ConfigGroup{
			SecureHttpCookie:            secureHttpCookie,
			MinPasswordLength:           minPasswordLength,
			MinUsernameLength:           minUsernameLength,
			MaxUsernameLength:           maxUsernameLength,
			UsernameAllowedSpecialChars: usernameAllowedSpecialChars,
			AllowedOrigin:               allowedOrigin,
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

	serverPort := strconv.Itoa(serverPortNum)

	app, otelShutdown := CreateServer(ctx, ServerConfig{
		ServerPort:       serverPort,
		Handler:          handler,
		Database:         db,
		MetricSecret:     metricSecret,
		CsrfSecret:       csrfSecret,
		RedisAddr:        redisAddr,
		RedisPassword:    redisPassword,
		RedisRateLimitDB: redisRateLimitDB,
		PostsPerMinute:   postsPerMinute,
		RateLimitEnabled: rateLimitEnabled,
		TrustProxy:       trustProxy,
		AllowedOrigin:    allowedOrigin,
	})

	go func() {
		err := app.Start(":" + serverPort)
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

	if err := app.Shutdown(shutdownCtx); err != nil {
		log.Warn(ctx, "Failed to shutdown server gracefully", "error", err)
	}
	if err := otelShutdown(shutdownCtx); err != nil {
		log.Warn(ctx, "Failed to shutdown OpenTelemetry tracer", "error", err)
	}
	metrics.ShutdownMetrics()
	if err := cleanupService.Stop(ctx); err != nil {
		log.Warn(ctx, "Failed to stop cleanup service", "error", err)
	}
	if err := db.Close(); err != nil {
		log.Error(ctx, "Failed to close database connection", "error", err)
	}
}

func requireEnv(ctx context.Context, key string) string {
	val, err := utils.RequireEnv(key)
	if err != nil {
		log.Fatal(ctx, "missing required environment variable", "key", key)
	}
	return val
}

func mustGetEnvInt(ctx context.Context, key string, defaultVal int) int {
	val, err := utils.GetEnvIntStrict(key, defaultVal)
	if err != nil {
		log.Fatal(ctx, "invalid environment variable", "error", err)
	}
	return val
}

func mustGetEnvInt64(ctx context.Context, key string, defaultVal int64) int64 {
	val, err := utils.GetEnvInt64Strict(key, defaultVal)
	if err != nil {
		log.Fatal(ctx, "invalid environment variable", "error", err)
	}
	return val
}

func mustGetEnvBool(ctx context.Context, key string, defaultVal bool) bool {
	val, err := utils.GetEnvBoolStrict(key, defaultVal)
	if err != nil {
		log.Fatal(ctx, "invalid environment variable", "error", err)
	}
	return val
}
