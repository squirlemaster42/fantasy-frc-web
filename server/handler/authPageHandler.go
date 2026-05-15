package handler

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net/http"
	"server/assert"
	"server/log"
	"server/view/login"
	"unicode"

	"github.com/labstack/echo/v4"
)

// We can probably do this in the middleware
func (h *Handler) HandleViewLogin(c echo.Context) error {
	csrfToken, err := generateCSRFCookie(c)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to generate CSRF cookie", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}
	loginIndex := login.LoginIndex(false, "", h.MinPasswordLength, csrfToken)
	login := login.Login(" | Login", false, loginIndex)
	err = Render(c, login)
	assert.NoErrorCF(c.Request().Context(), err, "Handle View Login Failed To Render")
	return nil
}

// We generate a 128 bit session token
// This token then needs to be hashed in the db and send back to the user
// We need to choose an expiration date too
func generateSessionToken() (string, error) {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base32.StdEncoding.EncodeToString(randomBytes), nil
}

func (h *Handler) HandleLoginPost(c echo.Context) error {
	if !validateCSRFCookie(c) {
		log.Warn(c.Request().Context(), "CSRF validation failed on login", "Ip", c.RealIP())
		csrfToken, err := generateCSRFCookie(c)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to generate CSRF cookie", "error", err)
			return c.String(http.StatusInternalServerError, "An error occurred")
		}
		loginIndex := login.LoginIndex(false, "Invalid request. Please try again.", h.MinPasswordLength, csrfToken)
		return Render(c, loginIndex)
	}

	username := c.FormValue("username")
	password := c.FormValue("password")

	valid, err := h.UserStore.ValidateLogin(c.Request().Context(), username, password)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to validate login", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to validate login")
	}

	if valid {
		log.Info(c.Request().Context(), "Valid login attempt for user", "Username", username)
		userUuid, err := h.UserStore.GetUserUuidByUsername(c.Request().Context(), username)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to get user uuid", "Username", username, "Error", err)
			return c.String(http.StatusInternalServerError, "Failed to validate login")
		}

		// Session fixation prevention: invalidate any pre-existing session token
		oldTok, err := c.Cookie("sessionToken")
		if err == nil && oldTok.Value != "" {
			if unregisterErr := h.UserStore.UnRegisterSession(c.Request().Context(), oldTok.Value); unregisterErr != nil {
				log.Warn(c.Request().Context(), "Failed to unregister old session during login", "Error", unregisterErr)
			}
		}

		sessionTok, err := generateSessionToken()
		if err != nil {
			log.Error(c.Request().Context(), "Failed to generate session token", "Error", err)
			return c.String(http.StatusInternalServerError, "Failed to create session")
		}
		if err := h.UserStore.RegisterSession(c.Request().Context(), userUuid, sessionTok); err != nil {
			log.Error(c.Request().Context(), "Failed to register session", "Error", err)
			return c.String(http.StatusInternalServerError, "Failed to create session")
		}

		cookie := new(http.Cookie)
		cookie.Name = "sessionToken"
		cookie.Value = sessionTok
		cookie.HttpOnly = true
		cookie.Secure = h.SecureHttpCookie
		cookie.SameSite = http.SameSiteLaxMode
		cookie.Path = "/"
		c.SetCookie(cookie)
		c.Response().Header().Set("HX-Redirect", "/u/home")
		return nil
	}

	log.Warn(c.Request().Context(), "Invalid login attempt for user", "Username", username)
	csrfToken, err := generateCSRFCookie(c)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to generate CSRF cookie", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}
	loginIndex := login.LoginIndex(false, "You have entered an invalid username or password", h.MinPasswordLength, csrfToken)
	err = Render(c, loginIndex)
	assert.NoErrorCF(c.Request().Context(), err, "Failed To Render Login Page With Error")

	return err
}

func (h *Handler) HandleLogoutPost(c echo.Context) error {
	userTok, err := c.Cookie("sessionToken")
	if err == nil && userTok.Value != "" {
		if unregisterErr := h.UserStore.UnRegisterSession(c.Request().Context(), userTok.Value); unregisterErr != nil {
			log.Warn(c.Request().Context(), "Failed to unregister session", "Error", unregisterErr)
		}
	}
	cookie := new(http.Cookie)
	cookie.Name = "sessionToken"
	cookie.Value = ""
	cookie.HttpOnly = true
	cookie.Secure = h.SecureHttpCookie
	cookie.SameSite = http.SameSiteLaxMode
	cookie.Path = "/"
	cookie.MaxAge = -1
	c.SetCookie(cookie)
	c.Response().Header().Set("HX-Redirect", "/login")
	return nil
}

func (h *Handler) HandleViewRegister(c echo.Context) error {
	csrfToken, err := generateCSRFCookie(c)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to generate CSRF cookie", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}
	registerIndex := login.RegisterIndex(false, "", h.MinPasswordLength, csrfToken)
	register := login.Register(" | Register", false, registerIndex)
	err = Render(c, register)
	assert.NoErrorCF(c.Request().Context(), err, "Handle View Register Page Failed To Render")
	return nil
}

func (h *Handler) HandlerRegisterPost(c echo.Context) error {
	if !validateCSRFCookie(c) {
		log.Warn(c.Request().Context(), "CSRF validation failed on register", "Ip", c.RealIP())
		csrfToken, err := generateCSRFCookie(c)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to generate CSRF cookie", "error", err)
			return c.String(http.StatusInternalServerError, "An error occurred")
		}
		register := login.RegisterIndex(false, "Invalid request. Please try again.", h.MinPasswordLength, csrfToken)
		return Render(c, register)
	}

	username := c.FormValue("username")
	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirmPassword")

	taken, err := h.UserStore.UsernameTaken(c.Request().Context(), username)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to check if username is taken", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to check username availability")
	}
	if taken {
		log.Info(c.Request().Context(), "Account creation attempt for existing user but username was taken", "Username", username)

		csrfToken, err := generateCSRFCookie(c)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to generate CSRF cookie", "error", err)
			return c.String(http.StatusInternalServerError, "An error occurred")
		}
		register := login.RegisterIndex(false, "Username Taken", h.MinPasswordLength, csrfToken)
		err = Render(c, register)
		assert.NoErrorCF(c.Request().Context(), err, "Handle View Register Page Failed To Render")

		return nil
	}

	if password != confirmPassword {
		log.Info(c.Request().Context(), "Password and Confirm Password do not match for user attempting to register", "Username", username)

		csrfToken, err := generateCSRFCookie(c)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to generate CSRF cookie", "error", err)
			return c.String(http.StatusInternalServerError, "An error occurred")
		}
		register := login.RegisterIndex(false, "Passwords Do Not Match", h.MinPasswordLength, csrfToken)
		err = Render(c, register)
		assert.NoErrorCF(c.Request().Context(), err, "Handle View Register Page Failed To Render")

		return nil
	}

	if len(password) < h.MinPasswordLength {
		log.Info(c.Request().Context(), "Password too short for user attempting to register", "Username", username)

		csrfToken, err := generateCSRFCookie(c)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to generate CSRF cookie", "error", err)
			return c.String(http.StatusInternalServerError, "An error occurred")
		}
		register := login.RegisterIndex(false, fmt.Sprintf("Password must be at least %d characters", h.MinPasswordLength), h.MinPasswordLength, csrfToken)
		err = Render(c, register)
		assert.NoErrorCF(c.Request().Context(), err, "Handle View Register Page Failed To Render")

		return nil
	}

	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		log.Info(c.Request().Context(), "Password does not meet complexity requirements for user attempting to register", "Username", username)

		csrfToken, err := generateCSRFCookie(c)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to generate CSRF cookie", "error", err)
			return c.String(http.StatusInternalServerError, "An error occurred")
		}
		register := login.RegisterIndex(false, "Password must contain at least one uppercase letter, one lowercase letter, and one digit", h.MinPasswordLength, csrfToken)
		err = Render(c, register)
		assert.NoErrorCF(c.Request().Context(), err, "Handle View Register Page Failed To Render")

		return nil
	}

	log.Info(c.Request().Context(), "Valid registration for user", "Username", username)
	userUuid, err := h.UserStore.RegisterUser(c.Request().Context(), username, password)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to register user", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create account")
	}
	sessionTok, err := generateSessionToken()
	if err != nil {
		log.Error(c.Request().Context(), "Failed to generate session token", "Error", err)
		return c.String(http.StatusInternalServerError, "Failed to create session")
	}
	if err := h.UserStore.RegisterSession(c.Request().Context(), userUuid, sessionTok); err != nil {
		log.Error(c.Request().Context(), "Failed to register session", "Error", err)
		return c.String(http.StatusInternalServerError, "Failed to create session")
	}
	cookie := new(http.Cookie)
	cookie.Name = "sessionToken"
	cookie.Value = sessionTok
	cookie.HttpOnly = true
	cookie.Secure = h.SecureHttpCookie
	cookie.SameSite = http.SameSiteLaxMode
	cookie.Path = "/"
	c.SetCookie(cookie)
	c.Response().Header().Set("HX-Redirect", "/u/home")
	return nil
}
