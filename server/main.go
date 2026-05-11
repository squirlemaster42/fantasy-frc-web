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
	draftManager := draft.NewDraftManager(tbaHandler, database, discordBus)
	//Start the draft daemon and add all running drafts to it
	draftDaemon := background.NewDraftDaemon(database, draftManager)
	err = draftDaemon.Start()
	if err != nil {
		log.Warn(context.Background(), "Failed to start draft daemon", "Error", err)
		panic("failed to start draft manager")
	}

	log.DebugNoContext("Checking for drafts that need to be added to daemon")
	drafts := model.GetDraftsInStatus(context.Background(), database, model.PICKING)
	for _, draftId := range drafts {
		err = draftDaemon.AddDraft(draftId)
		if err != nil {
			log.Warn(context.Background(), "Failed to add draft to manager in init", "Error", err)
		}
	}

	scorer := scorer.NewScorer(tbaHandler, database)
	if !*skipScoring {
		log.Info(context.Background(), "Started Scorer")
		scorer.RunScorer()
	}

	cleanupService := background.NewCleanupService(database, 60)
	err = cleanupService.Start()
	if err != nil {
		slog.Error("Failed to start cleanup service", "Error", err)
	}

	avatarStore, err := cache.NewAvatarStore(*tbaHandler)
	assert.NoError(context.Background(), err, "Failed to create avatar store")

	handler := handler.Handler {
		Database:     database,
		TbaHandler:   *tbaHandler,
		DraftManager: draftManager,
		Scorer:       scorer,
		AvatarStore:  &avatarStore,
		DiscordBus:   discordBus,
        SecureHttpCookie: secureHttpCookie,
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

	app, otelShutdown := CreateServer(serverPort, handler, metricSecret)

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
