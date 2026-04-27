package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"server/assert"
	"server/log"
	"server/model"
	"server/view/login"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

// We can probably do this in the middleware
func (h *Handler) HandleViewLogin(c echo.Context) error {
	loginIndex := login.LoginIndex(false, "")
	login := login.Login(" | Login", false, loginIndex)
	err := h.Render(c, login)
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

type faroTokenPayload struct {
	SessionToken string `json:"session_token"`
	Expiry       int64  `json:"expiry"`
}

func (h *Handler) generateFaroToken(sessionToken string) string {
	payload := faroTokenPayload{
		SessionToken: sessionToken,
		Expiry:       time.Now().Add(15 * time.Minute).Unix(),
	}
	payloadBytes, _ := json.Marshal(payload)
	payloadStr := hex.EncodeToString(payloadBytes)

	mac := hmac.New(sha256.New, []byte(h.FaroProxySecret))
	mac.Write([]byte(payloadStr))
	signature := hex.EncodeToString(mac.Sum(nil))

	return payloadStr + "." + signature
}

func (h *Handler) validateFaroToken(token string) (*faroTokenPayload, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, false
	}

	payloadStr := parts[0]
	signature := parts[1]

	mac := hmac.New(sha256.New, []byte(h.FaroProxySecret))
	mac.Write([]byte(payloadStr))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return nil, false
	}

	payloadBytes, err := hex.DecodeString(payloadStr)
	if err != nil {
		return nil, false
	}

	var payload faroTokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, false
	}

	if time.Now().Unix() > payload.Expiry {
		return nil, false
	}

	return &payload, true
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
		log.Error(c.Request().Context(), "Failed to check if username is taken", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to validate login")
	}

	valid := taken && model.ValidateLogin(h.Database, username, password)
	if valid {
		log.Info(c.Request().Context(), "Valid login attempt for user", "Username", username)
		userUuid := model.GetUserUuidByUsername(h.Database, username)
		sessionTok := generateSessionToken()
		model.RegisterSession(h.Database, userUuid, sessionTok)

		cookie := new(http.Cookie)
		cookie.Name = "sessionToken"
		cookie.Value = sessionTok
		cookie.HttpOnly = true
		cookie.Secure = h.SecureHttpCookie
		c.SetCookie(cookie)
		c.Response().Header().Set("HX-Redirect", "/u/home")
		return nil
	}

	log.Warn(c.Request().Context(), "Invalid login attempt for user", "Username", username)
	loginIndex := login.LoginIndex(false, "You have entered an invalid username or password")
	login := login.Login(" | Login", false, loginIndex)
	err = h.Render(c, login)
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
	err := h.Render(c, register)
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
		log.Error(c.Request().Context(), "Failed to check if username is taken", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to check username availability")
	}
	if taken {
		log.Info(c.Request().Context(), "Account creation attempt for existing user but username was taken", "Username", username)

		registerIndex := login.RegisterIndex(false, "Username Taken")
		register := login.Register(" | Register", false, registerIndex)
		err = h.Render(c, register)
		assert.NoErrorCF(err, "Handle View Register Page Failed To Render")

		return nil
	}

	if password != confirmPassword {
		log.Info(c.Request().Context(), "Password and Confirm Password do not match for user attempting to register", "Username", username)

		registerIndex := login.RegisterIndex(false, "Passwords Do Not Match")
		register := login.Register(" | Register", false, registerIndex)
		err = h.Render(c, register)
		assert.NoErrorCF(err, "Handle View Register Page Failed To Render")

		return nil
	}

	log.Info(c.Request().Context(), "Valid registration for user", "Username", username)
	userUuid := model.RegisterUser(h.Database, username, password)
	sessionTok := generateSessionToken()
	model.RegisterSession(h.Database, userUuid, sessionTok)
	cookie := new(http.Cookie)
	cookie.Name = "sessionToken"
	cookie.Value = sessionTok
	cookie.HttpOnly = true
	cookie.Secure = true
	c.SetCookie(cookie)
	c.Response().Header().Set("HX-Redirect", "/u/home")
	return nil
}
