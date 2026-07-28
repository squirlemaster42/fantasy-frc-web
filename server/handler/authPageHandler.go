package handler

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net/http"
	"server/log"
	"server/view/login"
	"unicode"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// We can probably do this in the middleware
func (h *Handler) HandleViewLogin(c echo.Context) error {
	return h.renderLoginWithError(c, "")
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

func validatePassword(password string, confirmPassword string, minLength int) string {
	if password != confirmPassword {
		return "Passwords Do Not Match"
	}
	if len(password) < minLength {
		return fmt.Sprintf("Password must be at least %d characters", minLength)
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
		return "Password must contain at least one uppercase letter, one lowercase letter, and one digit"
	}
	return ""
}

func (h *Handler) setSessionAndRedirect(c echo.Context, userUuid uuid.UUID) error {
	sessionTok, err := generateSessionToken()
	if err != nil {
		log.Error(c.Request().Context(), "Failed to generate session token", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create session")
	}
	if err := h.Stores.UserStore.RegisterSession(c.Request().Context(), userUuid, sessionTok); err != nil {
		log.Error(c.Request().Context(), "Failed to register session", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create session")
	}
	cookie := new(http.Cookie)
	cookie.Name = "sessionToken"
	cookie.Value = sessionTok
	cookie.HttpOnly = true
	cookie.Secure = h.Config.SecureHttpCookie
	cookie.SameSite = http.SameSiteLaxMode
	cookie.Path = "/"
	c.SetCookie(cookie)
	c.Response().Header().Set("HX-Redirect", "/u/home")
	return nil
}

func (h *Handler) HandleLoginPost(c echo.Context) error {
	if !validateCSRFCookie(c) {
		log.Warn(c.Request().Context(), "CSRF validation failed on login", "ip", c.RealIP())
		return h.renderLoginWithError(c, "Invalid request. Please try again.")
	}

	username := c.FormValue("username")
	password := c.FormValue("password")

	valid, err := h.Stores.UserStore.ValidateLogin(c.Request().Context(), username, password)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to validate login", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to validate login")
	}

	if valid {
		log.Info(c.Request().Context(), "Valid login attempt for user", "username", username)
		userUuid, err := h.Stores.UserStore.GetUserUuidByUsername(c.Request().Context(), username)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to get user uuid", "username", username, "error", err)
			return c.String(http.StatusInternalServerError, "Failed to validate login")
		}

		// Session fixation prevention: invalidate any pre-existing session token
		oldTok, err := c.Cookie("sessionToken")
		if err == nil && oldTok.Value != "" {
			if unregisterErr := h.Stores.UserStore.UnRegisterSession(c.Request().Context(), oldTok.Value); unregisterErr != nil {
				log.Warn(c.Request().Context(), "Failed to unregister old session during login", "error", unregisterErr)
			}
		}

		return h.setSessionAndRedirect(c, userUuid)
	}

	log.Warn(c.Request().Context(), "Invalid login attempt for user", "username", username)
	return h.renderLoginWithError(c, "You have entered an invalid username or password")
}

func (h *Handler) HandleLogoutPost(c echo.Context) error {
	var userUuidStr string
	if userUuid, ok := c.Get("userUuid").(uuid.UUID); ok {
		userUuidStr = userUuid.String()
	}
	log.Info(c.Request().Context(), "User logged out", "userUuid", userUuidStr, "ip", c.RealIP())
	userTok, err := c.Cookie("sessionToken")
	if err == nil && userTok.Value != "" {
		if unregisterErr := h.Stores.UserStore.UnRegisterSession(c.Request().Context(), userTok.Value); unregisterErr != nil {
			log.Warn(c.Request().Context(), "Failed to unregister session", "error", unregisterErr)
		}
	}
	cookie := new(http.Cookie)
	cookie.Name = "sessionToken"
	cookie.Value = ""
	cookie.HttpOnly = true
	cookie.Secure = h.Config.SecureHttpCookie
	cookie.SameSite = http.SameSiteLaxMode
	cookie.Path = "/"
	cookie.MaxAge = -1
	c.SetCookie(cookie)
	c.Response().Header().Set("HX-Redirect", "/login")
	return nil
}

func (h *Handler) renderRegisterWithError(c echo.Context, message string) error {
	csrfToken, err := generateCSRFCookie(c)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to generate CSRF cookie", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}
	register := login.RegisterIndex(false, message, h.Config.MinPasswordLength, csrfToken)
	return Render(c, register)
}

func (h *Handler) renderLoginWithError(c echo.Context, message string) error {
	csrfToken, err := generateCSRFCookie(c)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to generate CSRF cookie", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}
	loginPage := login.LoginIndex(false, message, h.Config.MinPasswordLength, csrfToken)
	return Render(c, loginPage)
}

func (h *Handler) HandleViewRegister(c echo.Context) error {
	return h.renderRegisterWithError(c, "")
}

func (h *Handler) HandlerRegisterPost(c echo.Context) error {
	if !validateCSRFCookie(c) {
		log.Warn(c.Request().Context(), "CSRF validation failed on register", "ip", c.RealIP())
		return h.renderRegisterWithError(c, "Invalid request. Please try again.")
	}

	username := c.FormValue("username")
	password := c.FormValue("password")
	confirmPassword := c.FormValue("confirmPassword")

	taken, err := h.Stores.UserStore.UsernameTaken(c.Request().Context(), username)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to check if username is taken", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to check username availability")
	}
	if taken {
		log.Warn(c.Request().Context(), "Account creation attempt for existing user but username was taken", "username", username)
		return h.renderRegisterWithError(c, "Username Taken")
	}

	if errMsg := validatePassword(password, confirmPassword, h.Config.MinPasswordLength); errMsg != "" {
		log.Warn(c.Request().Context(), "Registration password validation failed", "username", username)
		return h.renderRegisterWithError(c, errMsg)
	}

	log.Info(c.Request().Context(), "Valid registration for user", "username", username)
	userUuid, err := h.Stores.UserStore.RegisterUser(c.Request().Context(), username, password)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to register user", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to create account")
	}
	return h.setSessionAndRedirect(c, userUuid)
}
