package main

import (
	"flag"
	"log/slog"
	"os"
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
    slog.Info("-------- Starting Fantasy FRC --------")
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

    scorer := scorer.NewScorer(tbaHandler, database)
    if !*skipScoring {
        slog.Info("Started Scorer")
        scorer.RunScorer()
    }

    handler := handler.Handler {
        Database: database,
        TbaHandler: *tbaHandler,
        DraftManager: draftManager,
    }

    CreateServer(serverPort, handler)
}
