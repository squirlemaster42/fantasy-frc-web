package web

import (
	"fmt"
	db "server/database"
    "crypto/rand"
    "encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

const SESSON_TOKEN_SIZE = 50

type loginRequest struct {
    dbHandler *db.DatabaseDriver
    username string
    password string
}

type loginRespose struct {
    isValidated bool
    sessionToken string
}

func NewLoginRequest(username string, password string, dbHandler *db.DatabaseDriver) loginRequest {
    lrq := loginRequest{
        dbHandler: dbHandler,
        username: username,
        password: password,
    }

    return lrq
}

func (l *loginRequest) validateLogin() loginRespose {
    //Check password against the database and check that the user exists
    //If it is not correct we set isValidated to false and dont sent a session session token
    //If it is true then we generate a session token, persist it in the database, and set it
    var id int
    var username string
    l.dbHandler.Connection.QueryRow(fmt.Sprintf("Select * From Players Where username = '%s'", l.username)).Scan(&id, &username)

    isValid := false
    if isValid {
        return loginRespose{
            isValidated: true,
            sessionToken: generateSessionToken(SESSON_TOKEN_SIZE),
        }
    } else {
        return loginRespose{
            isValidated: false,
        }
    }
}

func generateSessionToken (length int) string {
    b := make([]byte, length)
    if _, err := rand.Read(b); err != nil {
        return ""
    }
    return hex.EncodeToString(b)
}

func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
