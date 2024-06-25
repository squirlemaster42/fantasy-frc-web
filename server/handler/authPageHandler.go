package handler

import (
	"crypto/rand"
	"encoding/base32"
	"server/assert"
	"server/view/login"

	"github.com/labstack/echo/v4"
)

func HandleViewLogin(c echo.Context) error {
    loginIndex := login.LoginIndex(false, "")
    login := login.Login(" | Login", false, loginIndex)
    err := Render(c, login)
    assert.NoErrorCF(err, "Handle View Login Failed To Render")
    return nil
}

//We generate a 128 bit session token
//This token then needs to be hashed in the db and send back to the user
//We need to choose an expiration date too
func generateSessionToken() string {
    randomBytes := make([]byte, 16)
    _, err := rand.Read(randomBytes)
    if err != nil {
        panic(err)
    }
    return base32.StdEncoding.EncodeToString(randomBytes)
}

func HandleLoginPost(c echo.Context) error {
    return nil
}
