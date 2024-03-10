package handler

import (
	"log"
	"server/database"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type SessionHandler struct {
    DbHandler *database.DatabaseDriver
}

//validDuration is in seconds
func (s *SessionHandler) registerSession(playerId int, validDuration int) string {
    //Create a session token for the player
    //Save the session token to the database
    //Compute the expiration time
    //Return the registered token

    sessionToken := uuid.New().String()
    encryptTok, err := bcrypt.GenerateFromPassword([]byte(sessionToken), 4)
    if err != nil {
        log.Fatalf("Could not encrypt session token for %d", playerId)
    }
    expirationDate := time.Now().Add(time.Second * time.Duration(validDuration))
    stmt := `UPDATE Players Set authTok = $1, tokExpirationDate = $2 WHERE Id = $3`
    _, err = s.DbHandler.Connection.Exec(stmt, encryptTok, expirationDate, playerId)
    if err != nil {
        log.Fatal("Could not store auth token in database")
    }

    return sessionToken
}

func (s *SessionHandler) validateSession(playerId int, sessionTok string) bool {
    query := `SELECT authTok From Players WHERE Id = $1`
    stmt, err := s.DbHandler.Connection.Prepare(query)
    defer stmt.Close()

    var encryptTok string
    err = stmt.QueryRow(playerId).Scan(&encryptTok)
    if err != nil {
        log.Fatalf("Could not retrieve auth token for player with id %d", playerId)
    }

    valid := bcrypt.CompareHashAndPassword([]byte(encryptTok), []byte(sessionTok))
    if valid != nil {
        return false
    }

    return true
}
