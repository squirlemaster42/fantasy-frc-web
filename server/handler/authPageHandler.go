package handler

import (
	"crypto/rand"
	"encoding/base32"
	"log/slog"
	"net/http"
	"server/assert"
	"server/model"
	"server/view/login"

	"github.com/labstack/echo/v4"
)

// We can probably do this in the middleware
func (h *Handler) HandleViewLogin(c echo.Context) error {
	loginIndex := login.LoginIndex(false, "")
	login := login.Login(" | Login", false, loginIndex)
	err := Render(c, login)
	assert.NoErrorCF(err, "Handle View Login Failed To Render")
	return nil
}

// We generate a 128 bit session token
// This token then needs to be hashed in the db and send back to the user
// We need to choose an expiration date too
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
	taken, err := model.UsernameTaken(h.Database, username)
	if err != nil {
		slog.Error("Failed to check if username is taken", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to validate login")
	}

	valid := taken && model.ValidateLogin(h.Database, username, password)
	if valid {
		slog.Info("Valid login attempt for user", "Username", username)
		userUuid := model.GetUserUuidByUsername(h.Database, username)
		sessionTok := generateSessionToken()
		model.RegisterSession(h.Database, userUuid, sessionTok)

		cookie := new(http.Cookie)
		cookie.Name = "sessionToken"
		cookie.Value = sessionTok
		cookie.HttpOnly = true
		//TODO enable secure again
		//cookie.Secure = true
		c.SetCookie(cookie)
		c.Response().Header().Set("HX-Redirect", "/u/home")
		return nil
	}

	slog.Warn("Invalid login attempt for user", "Username", username)
	login := login.LoginIndex(false, "You have entered an invalid username or password")
	err = RenderError(c, http.StatusUnauthorized, login)
	assert.NoErrorCF(err, "Failed To Render Login Page With Error")

	return err
}

func (h *Handler) HandleLogoutPost(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Logout Post")
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")
	model.UnRegisterSession(h.Database, userTok.Value)
	cookie := new(http.Cookie)
	cookie.Name = "sessionToken"
	cookie.Value = ""
	cookie.HttpOnly = true
	cookie.Secure = true
	c.SetCookie(cookie)
	c.Response().Header().Set("HX-Redirect", "/login")
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

	var err error

	taken, err := model.UsernameTaken(h.Database, username)
	if err != nil {
		slog.Error("Failed to check if username is taken", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to check username availability")
	}
	if taken {
		slog.Info("Account creation attempt for existing user but username was taken", "Username", username)

		register := login.RegisterIndex(false, "Username Taken")
		err = Render(c, register)
		assert.NoErrorCF(err, "Handle View Register Page Failed To Render")

		return nil
	}

	if password != confirmPassword {
		slog.Info("Password and Confirm Password do not match for user attempting to register", "Username", username)

		register := login.RegisterIndex(false, "Passwords Do Not Match")
		err = Render(c, register)
		assert.NoErrorCF(err, "Handle View Register Page Failed To Render")

		return nil
	}

	slog.Info("Valid registration for user", "Username", username)
	userUuid := model.RegisterUser(h.Database, username, password)
	sessionTok := generateSessionToken()
	model.RegisterSession(h.Database, userUuid, sessionTok)
	cookie := new(http.Cookie)
	cookie.Name = "sessionToken"
	cookie.Value = sessionTok
	cookie.HttpOnly = true
	cookie.Secure = true
	c.SetCookie(cookie)
	err = c.Redirect(http.StatusSeeOther, "/u/home")
	assert.NoErrorCF(err, "Failed to redirect on successful registration")

	return nil
}
