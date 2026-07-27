package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"net/http"
	"os"
	"os/signal"
	"server/assert"
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
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
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
	tbaTok := os.Getenv("TBA_TOKEN")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbUsername := os.Getenv("DB_USERNAME")
	dbIp := os.Getenv("DB_IP")
	dbName := os.Getenv("DB_NAME")
	serverPort := os.Getenv("SERVER_PORT")
	tbaWebhookSecret := os.Getenv("TBA_WEBHOOK_SECRET")
	metricSecret := os.Getenv("METRIC_SECRET")
	secureHttpCookieVar := os.Getenv("SECURE_HTTP_COOKIE")
	csrfSecret := os.Getenv("CSRF_SECRET")
	trustProxyVar := os.Getenv("TRUST_PROXY")
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")

	if csrfSecret == "" {
		panic("CSRF_SECRET environment variable is required")
	}

	minPasswordLength := utils.GetEnvInt("MIN_PASSWORD_LENGTH", 12)

	redisRateLimitDB := utils.GetEnvInt("REDIS_RATE_LIMIT_DB", 1)

	redisAvatarDB := utils.GetEnvInt("REDIS_AVATAR_DB", 2)

	postsPerMinute := utils.GetEnvInt64("RATE_LIMIT_POSTS_PER_MINUTE", 100)

	rateLimitEnabled := utils.GetEnvBool("RATE_LIMIT_ENABLED", true)

	log.Info(ctx, "Extracted Env Vars")
	database, err := database.RegisterDatabaseConnection(ctx, dbUsername, dbPassword, dbIp, dbName)
	if err != nil {
		log.Error(ctx, "Failed to register database connection", "error", err)
		os.Exit(1)
	}
	log.Info(ctx, "Registered Database Connection")

	tbaHandler := tbaHandler.NewHandler(tbaTok, database)

	secureHttpCookie, err := strconv.ParseBool(secureHttpCookieVar)
	if err != nil {
		log.Warn(ctx, "failed to parse secure http cookie env var. setting secureHttp to true", "error", err)
		secureHttpCookie = true
	}

	trustProxy, err := strconv.ParseBool(trustProxyVar)
	if err != nil {
		trustProxy = false
	}
	log.Info(ctx, "Trust proxy setting", "TRUST_PROXY", trustProxy)

	if trustProxy && allowedOrigin == "" {
		panic("ALLOWED_ORIGIN environment variable is required when TRUST_PROXY is true")
	}
	if allowedOrigin != "" {
		log.Info(ctx, "WebSocket origin validation configured", "ALLOWED_ORIGIN", allowedOrigin)
	} else {
		log.Info(ctx, "WebSocket origin validation using development fallback (localhost/same-origin)")
	}

	discordWebhookBus := discord.NewBus()
	draftStore := model.NewSQLDraftStore(database)
	userStore := model.NewSQLUserStore(database)
	teamStore := model.NewSQLTeamStore(database)
	discordStore := model.NewSQLDiscordStore(database)
	matchStore := model.NewSQLMatchStore(database)
	matchTeamStore := model.NewSQLMatchTeamStore(database)

	pickNotifier := &picking.PickNotifier{
		Watchers: make(map[int][]picking.Watcher),
	}

	draftActorMap := draft.NewDraftActorMap(draftStore, tbaHandler, discordStore, discordWebhookBus, pickNotifier)
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
	if !*skipScoring {
		log.Info(ctx, "Started Scorer")
		scorer.RunScorer(ctx)
	}

	cleanupService := background.NewCleanupService(database, 60)
	err = cleanupService.Start(ctx)
	if err != nil {
		log.Error(ctx, "Failed to start cleanup service", "error", err)
	}

	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	avatarStore, err := cache.NewAvatarStore(ctx, *tbaHandler, redisAddr, redisPassword, redisAvatarDB)
	assert.NoError(ctx, err, "Failed to create avatar store")

	handler := handler.Handler{
		DraftStore:        draftStore,
		UserStore:         userStore,
		TeamStore:         teamStore,
		TBAHandler:        *tbaHandler,
		DraftActorMap: draftActorMap,
		Scorer:            scorer,
		AvatarStore:       &avatarStore,
		DiscordWebhookBus: discordWebhookBus,
		SecureHttpCookie:  secureHttpCookie,
		MinPasswordLength: minPasswordLength,
		CsrfSecret:        csrfSecret,
		AllowedOrigin:     allowedOrigin,
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
			handler.TbaVerificationCode = string(body)
		}
	}
	handler.TbaWebhookSecret = tbaWebhookSecret

	app, otelShutdown := CreateServer(ctx, ServerConfig{
		ServerPort:       serverPort,
		Handler:          handler,
		Database:         database,
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

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
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
	if err := database.Close(); err != nil {
		log.Error(ctx, "Failed to close database connection", "error", err)
	}
}
