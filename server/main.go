package main

import (
	"flag"
	"log/slog"
	"os"
	"server/assert"
	"server/background"
	"server/database"
	"server/draft"
	"server/handler"
	"server/model"
	"server/scorer"
	"server/tbaHandler"
	"server/utils"

	"github.com/joho/godotenv"
)

func initLogger(logLevel string, logFormat string, logAddSource bool) {
	var level slog.Level
	switch logLevel {
	case "debug", "DEBUG", "Debug":
		level = slog.LevelDebug
	case "warn", "WARN", "Warn":
		level = slog.LevelWarn
	case "error", "ERROR", "Error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: logAddSource,
	}

	var handler slog.Handler
	if logFormat == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	slog.SetDefault(slog.New(handler))
}

func main() {
	skipScoring := flag.Bool("skipScoring", false, "When true is entered, the scorer will not be started")
	populateTeams := flag.Bool("populateTeams", false, "When true is entered, we will take the list of events and add all of those teams to the database")
	logLevel := flag.String("logLevel", "info", "Set the log level (debug, info, warn, error)")
	logFormat := flag.String("logFormat", "text", "Set the log format (text, json)")
	logAddSource := flag.Bool("logAddSource", false, "Add source file and line number to logs")
	flag.Parse()

	initLogger(*logLevel, *logFormat, *logAddSource)

	assert := assert.CreateAssertWithContext("Main")
	slog.Info("-------- Starting Fantasy FRC --------")

	err := godotenv.Load()
	assert.NoError(err, "Failed to load env vars")
	tbaTok := os.Getenv("TBA_TOKEN")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbUsername := os.Getenv("DB_USERNAME")
	dbIp := os.Getenv("DB_IP")
	dbName := os.Getenv("DB_NAME")
	serverPort := os.Getenv("SERVER_PORT")
	//TODO this should probably not be stored in the config and instead be
	//populated when we register the web hook and then propogated to other
	//servers if needed
	tbaWebhookSecret := os.Getenv("TBA_WEBHOOK_SECRET")
	slog.Info("Extracted Env Vars")
	database := database.RegisterDatabaseConnection(dbUsername, dbPassword, dbIp, dbName)
	slog.Info("Registered Database Connection")

	tbaHandler := tbaHandler.NewHandler(tbaTok, database)

	if *populateTeams {
		slog.Info("Populating Teams")
		for _, event := range utils.Events() {
			slog.Info("Creating teams for event", "event", event)
			for _, team := range tbaHandler.MakeTeamsAtEventRequest(event) {
				slog.Info("Checking if team is needed", "team", team.Key, "event", event)
				if model.GetTeam(database, team.Key) == nil {
					slog.Info("Creating team", "team", team.Key, "event", event)
					model.CreateTeam(database, team.Key, "")
				}
			}
		}
	}

	draftManager := draft.NewDraftManager(tbaHandler, database)
	//Start the draft daemon and add all running drafts to it
	draftDaemon := background.NewDraftDaemon(database, draftManager)
	err = draftDaemon.Start()
	if err != nil {
		slog.Warn("Failed to start draft daemon", "error", err)
		panic("failed to start draft manager")
	}

	slog.Info("Checking for drafts that need to be added to daemon")
	drafts := model.GetDraftsInStatus(database, model.PICKING)
	for _, draftId := range drafts {
		err = draftDaemon.AddDraft(draftId)
		if err != nil {
			slog.Warn("Failed to add draft to manager in init", "error", err)
		}
	}

	scorer := scorer.NewScorer(tbaHandler, database)
	if !*skipScoring {
		slog.Info("Started Scorer")
		scorer.RunScorer()
	}

	handler := handler.Handler{
		Database:         database,
		TbaHandler:       *tbaHandler,
		DraftManager:     draftManager,
		Scorer:           scorer,
		TbaWekhookSecret: tbaWebhookSecret,
	}

	CreateServer(serverPort, handler)
}
