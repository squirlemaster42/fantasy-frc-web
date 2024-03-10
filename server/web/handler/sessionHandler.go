package handler

import (
	//"log"
	"server/database"

	//"github.com/google/uuid"
	//"golang.org/x/crypto/bcrypt"
)

type SessionHandler struct {
    DbHandler *database.DatabaseDriver
}

func (s *SessionHandler) registerSession(playerId int, validDuration int) string {
    //Create a session token for the player
    //Save the session token to the database
    //Compute the expiration time
    //Return the registered token
    /*
    sessionToken := uuid.New().String()
    encryptTok, err := bcrypt.GenerateFromPassword([]byte(sessionToken), 4)
    if err != nil {
        log.Fatal("Could not encrypt session token for " + string(playerId))
    }
    */

    return ""
}

func (s *SessionHandler) validateSession(playerId int, sessionTok string) bool {
    return false
}
