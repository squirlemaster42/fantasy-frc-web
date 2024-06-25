package handler

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"server/assert"
	"server/model"
	"server/view/login"

	"github.com/labstack/echo/v4"
)

//TODO Do we want to do some sort of redirect here if the user already has a valid session token
func (h *Handler) HandleViewLogin(c echo.Context) error {
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

func (h *Handler) HandleLoginPost(c echo.Context) error {
    username := c.FormValue("username")
    password := c.FormValue("password")

    //Here we need to validate the login
    //We then want to pass a session token as a cookie
    //And redirect the user to the come page (or somewhere else if they were redirected to login from there [idk how to do this])

    //TODO We need a database here. How do we want to get that
    //Can we put it on the context? We could, there is a middlewhere that we could use or we could make a handler struct
    //The handler struct might be nicer because we can put the logger on it too
    valid := model.ValidateLogin(h.Database, username, password)
    if valid {
        h.Logger.Log(fmt.Sprintf("Valid login attempt for user: %s", username))
    } else {
        h.Logger.Log(fmt.Sprintf("---- Invalid login attempt for user: %s ----", username))
    }

    return nil
}
