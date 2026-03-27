package main

import (
	"context"
	"flag"
	"io"
	"os"
	"os/signal"
	"server/assert"
	"server/background"
	"server/cache"
	"server/database"
	"server/draft"
	"server/handler"
	"server/log"
	"server/model"
	"server/scorer"
	"server/tbaHandler"
	"server/utils"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	assert := assert.CreateAssertWithContext("Main")

	skipScoring := flag.Bool("skipScoring", false, "When true is entered, the scorer will not be started")
	verbose := flag.Bool("v", false, "Enable debug logging")
	flag.Parse()

	if *verbose {
		log.SetLevel(log.LevelDebug)
	}

	log.InfoNoContext("-------- Starting Fantasy FRC --------")

	err := godotenv.Load()
	assert.NoError(err, "Failed to load env vars")
	tbaTok := os.Getenv("TBA_TOKEN")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbUsername := os.Getenv("DB_USERNAME")
	sentryDSN := os.Getenv("SENTRY_DSN")
	dbIp := os.Getenv("DB_IP")
	dbName := os.Getenv("DB_NAME")
	serverPort := os.Getenv("SERVER_PORT")
	log.InfoNoContext("Extracted Env Vars")
	database := database.RegisterDatabaseConnection(dbUsername, dbPassword, dbIp, dbName)
	log.InfoNoContext("Registered Database Connection")

	tbaHandler := tbaHandler.NewHandler(tbaTok, database)

	draftManager := draft.NewDraftManager(tbaHandler, database)
	draftDaemon := background.NewDraftDaemon(database, draftManager)
	err = draftDaemon.Start()
	if err != nil {
		log.WarnNoContext("Failed to start draft daemon", "Error", err)
		panic("failed to start draft manager")
	}

	log.DebugNoContext("Checking for drafts that need to be added to daemon")
	drafts := model.GetDraftsInStatus(database, model.PICKING)
	for _, draftId := range drafts {
		err = draftDaemon.AddDraft(draftId)
		if err != nil {
			log.WarnNoContext("Failed to add draft to manager in init", "Error", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	scorer := scorer.NewScorer(tbaHandler, database)
	if !*skipScoring {
		log.InfoNoContext("Started Scorer")
		scorer.RunScorer(ctx)
	}

	avatarStore, err := cache.NewAvatarStore(*tbaHandler)
	assert.NoError(err, "Failed to create avatar store")

	handler := handler.Handler{
		Database:     database,
		TbaHandler:   *tbaHandler,
		DraftManager: draftManager,
		Scorer:       scorer,
		AvatarStore:  &avatarStore,
	}

	file, err := os.Open(utils.GetWebhookFilePath())
	if err != nil {
		log.WarnNoContext("Unable to open tba webhook secret file", "Error", err)
	} else {
		body, err := io.ReadAll(file)
		if err != nil {
			log.WarnNoContext("Failed to read tba webhook file body", "Error", err)
		} else {
			handler.TbaWekhookSecret = string(body)
		}
	}

	app := CreateServer(serverPort, handler, sentryDSN)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.InfoNoContext("Received shutdown signal", "Signal", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	draftDaemon.Stop()

	cancel()

	if err := app.Shutdown(shutdownCtx); err != nil {
		log.ErrorNoContext("Server shutdown error", "Error", err)
	}

	if err := avatarStore.Close(); err != nil {
		log.WarnNoContext("Failed to close Redis connection", "Error", err)
	}

	if err := database.Close(); err != nil {
		log.WarnNoContext("Failed to close database connection", "Error", err)
	}

	log.InfoNoContext("-------- Fantasy FRC stopped --------")
}
