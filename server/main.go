package main

import (
	"flag"
	"log"
	"log/slog"
	"os"

	"server/background"
	"server/database"
	draftInit "server/draftInit"
	"server/model"
	"server/scorer"
	"server/tbaHandler"

	"github.com/joho/godotenv"
)

func main() {
    slog.Info("-------- Starting Fantasy FRC --------")
    initDir := flag.String("initDir", "", "The directory containing drafts to initialize the scorer. This should only be done one each time the drafts change. Drafts with the same names as the files will be overriden")
    skipScoring := flag.String("skipScoring", "", "When true is entered, the scorer will not be started")
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
    tbaHandler := tbaHandler.NewHandler(tbaTok)
    database := database.RegisterDatabaseConnection(dbUsername, dbPassword, dbIp, dbName)
    slog.Info("Registered Database Connection")

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

    //Start the draft daemon and add all running drafts to it
    draftDaemon := background.NewDraftDaemon(database)
    draftDaemon.Start()
    drafts := model.GetDraftsInStatus(database, model.PICKING)
    for _, draftId := range drafts {
        draftDaemon.AddDraft(draftId)
    }

    scorer := scorer.NewScorer(tbaHandler, database)
    if !(*skipScoring == "true") {
        slog.Info("Started Scorer")
        scorer.RunScorer()
    }
    CreateServer(database, tbaHandler, serverPort)
}
