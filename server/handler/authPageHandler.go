package handler

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net/http"
	"server/assert"
	"server/model"
	"server/view/login"

	"github.com/labstack/echo/v4"
)

//TODO Do we want to do some sort of redirect here if the user already has a valid session token
//We can probably do this in the middleware
func (h *Handler) HandleViewLogin(c echo.Context) error {
    loginIndex := login.LoginIndex(false, "")
    login := login.Login(" | Login", false, loginIndex)
    //TODO We should probably make tailwind work offline to make the dev experience better
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
    //We wont validate the password if the user does not exist
    valid := model.UsernameTaken(h.Database, username) && model.ValidateLogin(h.Database, username, password)
    if valid {
        h.Logger.Log(fmt.Sprintf("Valid login attempt for user: %s", username))
        userId := model.GetUserIdByUsername(h.Database, username)
        sessionTok := generateSessionToken()
        model.RegisterSession(h.Database, userId, sessionTok)

        cookie := new(http.Cookie)
        cookie.Name = "sessionToken"
        cookie.Value = sessionTok
        cookie.HttpOnly = true
        cookie.Secure = true
        c.SetCookie(cookie)
        err := c.Redirect(http.StatusSeeOther, "/u/home")
        assert.NoErrorCF(err, "Failed to redirect on successful login")

        return nil
    }

    h.Logger.Log(fmt.Sprintf("---- Invalid login attempt for user: %s ----", username))
    login := login.LoginIndex(false, "You have entered an invalid username or password")
    err := Render(c, login)
    assert.NoErrorCF(err, "Failed To Render Login Page With Error")

    return nil
}

func (h *Handler) HandleViewRegister(c echo.Context) error {
    registerIndex := login.RegisterIndex(false, "")
    register := login.Register(" | Register", false, registerIndex)
    err := Render(c, register)
    assert.NoErrorCF(err, "Handle View Register Page Failed To Render")
    return nil
}

func (h *Handler) HandlerRegisterPost(c echo.Context) error {
    username := c.FormValue("username")
    password := c.FormValue("password")
    confirmPassword := c.FormValue("confirmPassword")

    //TODO We need to make sure the user does not already exist
    if model.UsernameTaken(h.Database, username) {
        h.Logger.Log(fmt.Sprintf("---- Account creation attempt for existing user: %s ----", username))

        register := login.RegisterIndex(false, "Username Taken")
        err := Render(c, register)
        assert.NoErrorCF(err, "Handle View Register Page Failed To Render")

        return nil

    }

    if password != confirmPassword {
        h.Logger.Log(fmt.Sprintf("---- Password and Confirm Password do not match for user attempting to register: %s ----", username))

        register := login.RegisterIndex(false, "Passwords Do Not Match")
        err := Render(c, register)
        assert.NoErrorCF(err, "Handle View Register Page Failed To Render")

        return nil
    }

    h.Logger.Log(fmt.Sprintf("Valid registration for user: %s", username))
    model.RegisterUser(h.Database, username, password)
    //TODO We should make register user return the user id so we dont need two queries
    userId := model.GetUserIdByUsername(h.Database, username)
    sessionTok := generateSessionToken()
    model.RegisterSession(h.Database, userId, sessionTok)
    cookie := new(http.Cookie)
    cookie.Name = "sessionToken"
    cookie.Value = sessionTok
    cookie.HttpOnly = true
    cookie.Secure = true
    c.SetCookie(cookie)
    err := c.Redirect(http.StatusSeeOther, "/u/home")
    assert.NoErrorCF(err, "Failed to redirect on successful registration")

    return nil
}
