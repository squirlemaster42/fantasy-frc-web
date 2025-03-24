package main

import (
	//"os"
	"flag"
	"fmt"
	"log"
	"os"

	"server/background"
	"server/database"
	draftInit "server/draftInit"
	"server/logging"
	"server/model"
	"server/scorer"
	"server/tbaHandler"

	"github.com/joho/godotenv"
)

func main() {
    logger := logging.NewLogger(&logging.TimestampedLogger{})
    logger.Start()
    logger.Log("-------- Starting Fantasy FRC --------")
    initDir := flag.String("initDir", "", "The directory containing drafts to initialize the scorer. This should only be done one each time the drafts change. Drafts with the same names as the files will be overriden")
    skipScoring := flag.String("skipScoring", "", "When true is entered, the scorer will not be started")
    flag.Parse()

    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    dbPassword := os.Getenv("DB_PASSWORD")
    dbUsername := os.Getenv("DB_USERNAME")
    dbIp := os.Getenv("DB_IP")
    dbName := os.Getenv("DB_NAME")
    logger.Log("Extracted Env Vars")
    //sessionSecret := os.Getenv("SESSION_SECRET")
    tbaHandler := tbaHandler.NewHandler(tbaTok, logger)
    database := database.RegisterDatabaseConnection(dbUsername, dbPassword, dbIp, dbName)
    logger.Log("Registered Database Connection")

    //If we have an init dir, then parse all of the drafts in that folder
    if len(*initDir) > 0 {
        logger.Log(fmt.Sprintf("Loading draft from %s", *initDir))
        files, err := os.ReadDir(*initDir)
        if err != nil {
            log.Fatal(err)
        }

        for _, e := range files {
            if !e.Type().IsDir() {
                logger.Log(fmt.Sprintf("Loading Draft: %s Into The Database\n", e.Name()))
                draftInit.LoadCSVIntoDb(database, *initDir + e.Name())
            }
        }
        logger.Log("Finished loading draft")
    }

    //Start the draft daemon and add all running drafts to it
    draftDaemon := background.NewDraftDaemon(logger, database)
    draftDaemon.Start()
    drafts := model.GetDraftsInStatus(database, model.PICKING)
    for _, draftId := range drafts {
        draftDaemon.AddDraft(draftId)
    }

    scorer := scorer.NewScorer(tbaHandler, database, logger)
    if !(*skipScoring == "true") {
        logger.Log("Started Scorer")
        scorer.RunScorer()
    }
    CreateServer(database, tbaHandler, logger)
}
