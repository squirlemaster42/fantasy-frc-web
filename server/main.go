package main

import (
	"flag"
	"io"
	"os"
	"server/assert"
	"server/background"
	"server/cache"
	"server/database"
	"server/discord"
	"server/draft"
	"server/handler"
	"server/log"
	"server/model"
	"server/scorer"
	"server/tbaHandler"
	"server/utils"
	"strconv"

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
	sentryDNS := os.Getenv("SENTRY_DNS")
	dbIp := os.Getenv("DB_IP")
	dbName := os.Getenv("DB_NAME")
	serverPort := os.Getenv("SERVER_PORT")
	tbaWebhookSecret := os.Getenv("TBA_WEBHOOK_SECRET")
	metricSecret := os.Getenv("METRIC_SECRET")
    secureHttpCookieVar := os.Getenv("SECURE_HTTP_COOKIE")
	log.InfoNoContext("Extracted Env Vars")
	database := database.RegisterDatabaseConnection(dbUsername, dbPassword, dbIp, dbName)
	log.InfoNoContext("Registered Database Connection")

	tbaHandler := tbaHandler.NewHandler(tbaTok, database)

    secureHttpCookie, err := strconv.ParseBool(secureHttpCookieVar)
    if err != nil {
        log.WarnNoContext("failed to parse secure http cookie env var", "Error", err)
        secureHttpCookie = true
    }

	discordBus := discord.NewBus()
	draftManager := draft.NewDraftManager(tbaHandler, database, discordBus)
	//Start the draft daemon and add all running drafts to it
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

	scorer := scorer.NewScorer(tbaHandler, database)
	if !*skipScoring {
		log.InfoNoContext("Started Scorer")
		scorer.RunScorer()
	}

	avatarStore, err := cache.NewAvatarStore(*tbaHandler)
	assert.NoError(err, "Failed to create avatar store")

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
		log.WarnNoContext("Unable to open tba webhook secret file", "Error", err)
	} else {
		body, err := io.ReadAll(file)
		if err != nil {
			log.WarnNoContext("Failed to read tba webhook file body", "Error", err)
		} else {
			handler.TbaVerificationCode = string(body)
		}
	}
	handler.TbaWebhookSecret = tbaWebhookSecret

	CreateServer(serverPort, handler, sentryDNS, metricSecret)
}
