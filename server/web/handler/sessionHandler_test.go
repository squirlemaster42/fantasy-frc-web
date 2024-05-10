package handler

import (
	"os"
	"server/database"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

//Test registering a session and then checking if it is valid
func TestRegisterSession(t *testing.T) {
    godotenv.Load()
    dbPassword := os.Getenv("DB_PASSWORD")
    dbUsername := os.Getenv("DB_USERNAME")
    dbIp := os.Getenv("DB_IP")
    dbName := os.Getenv("DB_NAME")
    dbDriver := database.CreateDatabaseDriver(dbUsername, dbPassword, dbIp, dbName)

    //Check that the test user is in the databse, if not, create it
    var playerId int
    err := dbDriver.Connection.QueryRow("Select Id From Players Where Name = 'unittestuser'").Scan(&playerId)
    if err != nil {
        dbDriver.Connection.Exec("INSERT INTO Players (Name) VALUES ('unittestuser')")
    }
    err = dbDriver.Connection.QueryRow("Select Id From Players Where Name = 'unittestuser'").Scan(&playerId)
    if err != nil {
        t.Fatal(err)
    }

    //Create an auth token and then check if we can validate the session
    sessionHandler := SessionHandler{DbHandler: dbDriver}
    sessionString := sessionHandler.registerSession(playerId, 10)
    isValid := sessionHandler.validateSession(playerId, sessionString)
    if !isValid {
        t.Fatal("Session should be valid but was not")
    }
    time.Sleep(time.Duration(15) * time.Second)
    isValid = sessionHandler.validateSession(playerId, sessionString)
    if isValid {
        t.Fatal("Session should not be valid but was")
    }
}
