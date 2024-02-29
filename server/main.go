package main

import (
    "os"

	"github.com/joho/godotenv"

	"server/database"
    "server/scoring"
    server "server/web"
)

func main() {
    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    dbPassword := os.Getenv("DB_PASSWORD")
    dbUsername := os.Getenv("DB_USERNAME")
    dbIp := os.Getenv("DB_IP")
    dbName := os.Getenv("DB_NAME")
    tbaHandler := scoring.NewHandler(tbaTok)
    dbDriver := database.CreateDatabaseDriver(dbUsername, dbPassword, dbIp, dbName)
    scorer := scoring.NewScorer(tbaHandler, dbDriver)
    scorer.RunScorer()
    server.CreateServer()
}
