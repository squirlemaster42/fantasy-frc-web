package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log/slog"
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
	"server/scorer"
	"server/tbaHandler"
	"server/utils"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	assert := assert.CreateAssertWithContext("Main")

	skipScoring := flag.Bool("skipScoring", false, "When true is entered, the scorer will not be started")
	verbose := flag.Bool("v", false, "Enable debug logging")
	logFormat := flag.String("log-format", "json", "Log format: json or text")
	flag.Parse()

	log.SetupLogger(*logFormat)

	if *verbose {
		log.SetLevel(log.LevelDebug)
	}

	log.Info(context.Background(), "-------- Starting Fantasy FRC --------")

	err := godotenv.Load()
	if err != nil {
		log.Info(context.Background(), "No .env file loaded, using environment variables")
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
	minPasswordLengthVar := os.Getenv("MIN_PASSWORD_LENGTH")
	redisAddr := os.Getenv("REDIS_ADDR")
	redisPassword := os.Getenv("REDIS_PASSWORD")
	redisRateLimitDBVar := os.Getenv("REDIS_RATE_LIMIT_DB")
	redisAvatarDBVar := os.Getenv("REDIS_AVATAR_DB")
	postsPerMinuteVar := os.Getenv("RATE_LIMIT_POSTS_PER_MINUTE")

	if csrfSecret == "" {
		panic("CSRF_SECRET environment variable is required")
	}

	minPasswordLength := 12
	if minPasswordLengthVar != "" {
		parsed, err := strconv.Atoi(minPasswordLengthVar)
		if err == nil && parsed > 0 {
			minPasswordLength = parsed
		}
	}

	redisRateLimitDB := 1
	if redisRateLimitDBVar != "" {
		parsed, err := strconv.Atoi(redisRateLimitDBVar)
		if err == nil {
			redisRateLimitDB = parsed
		}
	}

	redisAvatarDB := 2
	if redisAvatarDBVar != "" {
		parsed, err := strconv.Atoi(redisAvatarDBVar)
		if err == nil {
			redisAvatarDB = parsed
		}
	}

	postsPerMinute := int64(100)
	if postsPerMinuteVar != "" {
		parsed, err := strconv.ParseInt(postsPerMinuteVar, 10, 64)
		if err == nil && parsed > 0 {
			postsPerMinute = parsed
		}
	}

	log.Info(context.Background(), "Extracted Env Vars")
	database, err := database.RegisterDatabaseConnection(context.Background(), dbUsername, dbPassword, dbIp, dbName)
	if err != nil {
		log.Error(context.Background(), "Failed to register database connection", "error", err)
		os.Exit(1)
	}
	log.Info(context.Background(), "Registered Database Connection")

	tbaHandler := tbaHandler.NewHandler(tbaTok, database)

	secureHttpCookie, err := strconv.ParseBool(secureHttpCookieVar)
	if err != nil {
		log.Warn(context.Background(), "failed to parse secure http cookie env var. setting secureHttp to true", "Error", err)
		secureHttpCookie = true
	}

	discordBus := discord.NewBus()
	draftStore := model.NewSQLDraftStore(database)
	userStore := model.NewSQLUserStore(database)
	teamStore := model.NewSQLTeamStore(database)
	discordStore := model.NewSQLDiscordStore(database)
	matchStore := model.NewSQLMatchStore(database)
	matchTeamStore := model.NewSQLMatchTeamStore(database)

	draftManager := draft.NewDraftManager(tbaHandler, draftStore, teamStore, discordStore, discordBus)
	//Start the draft daemon and add all running drafts to it
	draftDaemon := background.NewDraftDaemon(draftStore, draftManager)
	err = draftDaemon.Start()
	if err != nil {
		log.Warn(context.Background(), "Failed to start draft daemon", "Error", err)
		panic("failed to start draft manager")
	}

	log.DebugNoContext("Checking for drafts that need to be added to daemon")
	drafts, err := draftStore.GetDraftsInStatus(context.Background(), model.PICKING)
	if err != nil {
		log.Warn(context.Background(), "Could not get any drafts in picking status", "Error", err)
	} else {
		for _, draftId := range drafts {
			err = draftDaemon.AddDraft(draftId)
			if err != nil {
				log.Warn(context.Background(), "Failed to add draft to manager in init", "Error", err)
			}
		}
	}

	scorer := scorer.NewScorer(tbaHandler, matchStore, matchTeamStore, teamStore)
	if !*skipScoring {
		log.Info(context.Background(), "Started Scorer")
		scorer.RunScorer()
	}

	cleanupService := background.NewCleanupService(database, 60)
	err = cleanupService.Start()
	if err != nil {
		slog.Error("Failed to start cleanup service", "Error", err)
	}

	avatarStore, err := cache.NewAvatarStore(*tbaHandler, redisAddr, redisPassword, redisAvatarDB)
	assert.NoError(context.Background(), err, "Failed to create avatar store")

	handler := handler.Handler{
		DraftStore:        draftStore,
		UserStore:         userStore,
		TeamStore:         teamStore,
		TbaHandler:        *tbaHandler,
		DraftManager:      draftManager,
		Scorer:            scorer,
		AvatarStore:       &avatarStore,
		DiscordBus:        discordBus,
		SecureHttpCookie:  secureHttpCookie,
		MinPasswordLength: minPasswordLength,
		CsrfSecret:        csrfSecret,
	}

	// Load the tba webhook secret
	file, err := os.Open(utils.GetWebhookFilePath())
	if err != nil {
		log.Warn(context.Background(), "Unable to open tba webhook secret file", "Error", err)
	} else {
		body, err := io.ReadAll(file)
		if err != nil {
			log.Warn(context.Background(), "Failed to read tba webhook file body", "Error", err)
		} else {
			handler.TbaVerificationCode = string(body)
		}
	}
	handler.TbaWebhookSecret = tbaWebhookSecret

	app, otelShutdown := CreateServer(serverPort, handler, database, metricSecret, csrfSecret, redisAddr, redisPassword, redisRateLimitDB, postsPerMinute)

	go func() {
		err := app.Start(":" + serverPort)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			assert.NoError(context.Background(), err, "Failed to start server")
		}
	}()

	// Wait for shutdown signal
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	<-shutdownChan

	log.Info(context.Background(), "Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.Shutdown(ctx); err != nil {
		log.Warn(context.Background(), "Failed to shutdown server gracefully", "error", err)
	}
	if err := otelShutdown(ctx); err != nil {
		log.Warn(context.Background(), "Failed to shutdown OpenTelemetry tracer", "error", err)
	}
	metrics.ShutdownMetrics()
	if err := database.Close(); err != nil {
		log.Warn(context.Background(), "Failed to close database connection", "error", err)
	}
}
