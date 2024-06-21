package main

import (
	//"os"
	"flag"
	"fmt"
	"os"
    "log"

	"github.com/joho/godotenv"
	"server/database"
	"server/scoring"
    draftInit "server/draftInit"
)

func main() {
    fmt.Println("-------- Starting Fantasy FRC --------")
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
        fmt.Println(*initDir)
        files, err := os.ReadDir(*initDir)
        if err != nil {
            log.Fatal(err)
        }

        for _, e := range files {
            if !e.Type().IsDir() {
                fmt.Printf("Loading Draft: %s Into The Database\n", e.Name())
                draftInit.LoadCSVIntoDb(database, *initDir + e.Name())
            }
        }
    }

    scorer := scoring.NewScorer(tbaHandler, database)
    if !(*skipScoring == "true") {
        fmt.Println("Starting with Scoring")
        scorer.RunScorer()
    }
    CreateServer(database)
}
