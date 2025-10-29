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

func main() {
    assert := assert.CreateAssertWithContext("Main")
    slog.Info("-------- Starting Fantasy FRC --------")
    skipScoring := flag.Bool("skipScoring", false, "When true is entered, the scorer will not be started")
    populateTeams := flag.Bool("populateTeams", false, "When true is entered, we will take the list of events and add all of those teams to the database")
    flag.Parse()

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
            slog.Info("Creating teams for event", "Event", event)
            for _, team := range tbaHandler.MakeTeamsAtEventRequest(event) {
                slog.Info("Checking if team is needed", "Team", team.Key, "Event", event)
                if model.GetTeam(database, team.Key) == nil {
                    slog.Info("Creating team", "Team", team.Key, "Event", event)
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
        slog.Warn("Failed to start draft daemon", "Error", err)
        panic("failed to start draft manager")
    }

    slog.Info("Checking for drafts that need to be added to daemon")
    drafts := model.GetDraftsInStatus(database, model.PICKING)
    for _, draftId := range drafts {
        err = draftDaemon.AddDraft(draftId)
        if err != nil {
            slog.Warn("Failed to add draft to manager in init", "Error", err)
        }
    }

    scorer := scorer.NewScorer(tbaHandler, database)
    if !*skipScoring {
        slog.Info("Started Scorer")
        scorer.RunScorer()
    }

    handler := handler.Handler {
        Database: database,
        TbaHandler: *tbaHandler,
        DraftManager: draftManager,
        Scorer: scorer,
        TbaWekhookSecret: tbaWebhookSecret,
    }

    CreateServer(serverPort, handler)
}
