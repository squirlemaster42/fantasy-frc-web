package main

import (
	//"os"
	"flag"
	"fmt"
	"log"
	"os"

	"server/database"
	draftInit "server/draftInit"
	"server/logging"
	"server/scoring"

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
    //sessionSecret := os.Getenv("SESSION_SECRET")
    tbaHandler := scoring.NewHandler(tbaTok)
    database := database.RegisterDatabaseConnection(dbUsername, dbPassword, dbIp, dbName)

    //If we have an init dir, then parse all of the drafts in that folder
    if len(*initDir) > 0 {
        logger.Log(*initDir)
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
    }

    scorer := scoring.NewScorer(tbaHandler, database)
    if !(*skipScoring == "true") {
        scorer.RunScorer()
    }
    CreateServer(database, &logger)
}
