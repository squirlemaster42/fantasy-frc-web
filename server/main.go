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
	server "server/web"
    draftInit "server/draftInit"
)

func main() {
    initDir := flag.String("initDir", "", "The directory containing drafts to initialize the scorer. This should only be done one each time the drafts change. Drafts with the same names as the files will be overriden")
    flag.Parse()

    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    dbPassword := os.Getenv("DB_PASSWORD")
    dbUsername := os.Getenv("DB_USERNAME")
    dbIp := os.Getenv("DB_IP")
    dbName := os.Getenv("DB_NAME")
    tbaHandler := scoring.NewHandler(tbaTok)
    dbDriver := database.CreateDatabaseDriver(dbUsername, dbPassword, dbIp, dbName)

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
                draftInit.LoadCSVIntoDb(dbDriver, *initDir + e.Name())
            }
        }
    }

    scorer := scoring.NewScorer(tbaHandler, dbDriver)
    scorer.RunScorer()
    server.CreateServer(scorer)
}
