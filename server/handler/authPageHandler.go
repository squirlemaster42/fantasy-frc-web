package handler

import (
	"errors"
	"net/http"
	"server/authentication"
	"server/log"
	"server/view/login"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// We can probably do this in the middleware
func (h *Handler) HandleViewLogin(c echo.Context) error {
	return h.renderLoginWithError(c, "")
}

func (h *Handler) setSessionCookie(c echo.Context, sessionToken string) {
	cookie := new(http.Cookie)
	cookie.Name = "sessionToken"
	cookie.Value = sessionToken
	cookie.HttpOnly = true
	cookie.Secure = h.Config.SecureHttpCookie
	cookie.SameSite = http.SameSiteLaxMode
	cookie.Path = "/"
	c.SetCookie(cookie)
}

func (h *Handler) setSessionAndRedirect(c echo.Context, userUuid uuid.UUID, sessionToken string) error {
	h.setSessionCookie(c, sessionToken)
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

	// Session fixation prevention: invalidate any pre-existing session token
	oldTok, err := c.Cookie("sessionToken")
	if err == nil && oldTok.Value != "" {
		if logoutErr := h.Services.AuthService.Logout(c.Request().Context(), oldTok.Value); logoutErr != nil {
			log.Warn(c.Request().Context(), "Failed to logout old session during login", "error", logoutErr)
		}
	}

	userUuid, sessionToken, err := h.Services.AuthService.Login(c.Request().Context(), username, password)
	if err != nil {
		if errors.Is(err, authentication.ErrInvalidCredentials) {
			log.Warn(c.Request().Context(), "Invalid login attempt for user", "username", username)
			return h.renderLoginWithError(c, "You have entered an invalid username or password")
		}
		log.Error(c.Request().Context(), "Failed to validate login", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to validate login")
	}

	return h.setSessionAndRedirect(c, userUuid, sessionToken)
}

func (h *Handler) HandleLogoutPost(c echo.Context) error {
	var userUuidStr string
	if userUuid, ok := c.Get("userUuid").(uuid.UUID); ok {
		userUuidStr = userUuid.String()
	}
	log.Info(c.Request().Context(), "User logged out", "userUuid", userUuidStr, "ip", c.RealIP())
	userTok, err := c.Cookie("sessionToken")
	if err == nil && userTok.Value != "" {
		if logoutErr := h.Services.AuthService.Logout(c.Request().Context(), userTok.Value); logoutErr != nil {
			log.Warn(c.Request().Context(), "Failed to logout session", "error", logoutErr)
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

	userUuid, sessionToken, err := h.Services.AuthService.Register(c.Request().Context(), username, password)
	if err != nil {
		switch {
		case errors.Is(err, authentication.ErrUsernameTaken):
			log.Warn(c.Request().Context(), "Account creation attempt for existing user but username was taken", "username", username)
			return h.renderRegisterWithError(c, "Username Taken")
		case authentication.IsValidationError(err):
			log.Warn(c.Request().Context(), "Registration validation failed", "username", username)
			return h.renderRegisterWithError(c, err.Error())
		default:
			log.Error(c.Request().Context(), "Failed to register user", "error", err)
			return c.String(http.StatusInternalServerError, "Failed to create account")
		}
	}

	return h.setSessionAndRedirect(c, userUuid, sessionToken)
}
