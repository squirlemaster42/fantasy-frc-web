package main

import (
	"flag"
	"log"
	"log/slog"
	"os"

	"server/background"
	"server/database"
	draftInit "server/draftInit"
	"server/handler"
	"server/model"
	"server/picking"
	"server/scorer"
	"server/tbaHandler"
	"server/utils"

	"github.com/joho/godotenv"
)

func main() {
    slog.Info("-------- Starting Fantasy FRC --------")
    initDir := flag.String("initDir", "", "The directory containing drafts to initialize the scorer. This should only be done one each time the drafts change. Drafts with the same names as the files will be overriden")
    skipScoring := flag.Bool("skipScoring", false, "When true is entered, the scorer will not be started")
    populateTeams := flag.Bool("populateTeams", false, "When true is entered, we will take the list of events and add all of those teams to the database")
    flag.Parse()

    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    dbPassword := os.Getenv("DB_PASSWORD")
    dbUsername := os.Getenv("DB_USERNAME")
    dbIp := os.Getenv("DB_IP")
    dbName := os.Getenv("DB_NAME")
    serverPort := os.Getenv("SERVER_PORT")
    slog.Info("Extracted Env Vars")
    //sessionSecret := os.Getenv("SESSION_SECRET")
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

    //If we have an init dir, then parse all of the drafts in that folder
    if len(*initDir) > 0 {
        slog.Info("Loading draft from file", "Directory", *initDir)
        files, err := os.ReadDir(*initDir)
        if err != nil {
            log.Fatal(err)
        }

        for _, e := range files {
            if !e.Type().IsDir() {
                slog.Info("Loading Draft Into The Database\n", "Draft", e.Name())
                draftInit.LoadCSVIntoDb(database, *initDir + e.Name())
            }
        }
        slog.Info("Finished loading draft")
    }

    pickNotifier := &picking.PickNotifier {
        Database: database,
        Watchers: make(map[int][]picking.Watcher),
    }

    draftPickManager := picking.NewDraftPickManager(database, tbaHandler)

    //Start the draft daemon and add all running drafts to it
    draftDaemon := background.NewDraftDaemon(database, pickNotifier)
    draftDaemon.Start()
    slog.Info("Checking for drafts that need to be added to daemon")
    drafts := model.GetDraftsInStatus(database, model.PICKING)
    for _, draftId := range drafts {
        draftDaemon.AddDraft(draftId)
        draftPickManager.GetPickManagerForDraft(draftId).AddListener(pickNotifier)
    }

    scorer := scorer.NewScorer(tbaHandler, database)
    if !*skipScoring {
        slog.Info("Started Scorer")
        scorer.RunScorer()
    }

    handler := handler.Handler{
        Database: database,
        TbaHandler: *tbaHandler,
        Notifier: pickNotifier,
        DraftDaemon: draftDaemon,
        DraftPickManager: draftPickManager,
    }

    CreateServer(serverPort, handler)
}
