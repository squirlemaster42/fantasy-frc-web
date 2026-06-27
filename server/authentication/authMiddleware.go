package authentication

import (
	"crypto/subtle"
	"net/http"
	"server/log"
	"server/metrics"
	"server/model"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type contextKey string

const UserUuidKey contextKey = "userUuid"
const IsAdminKey contextKey = "isAdmin"

type Authenticator struct {
	userStore model.UserStore
}

func NewAuth(userStore model.UserStore) *Authenticator {
	return &Authenticator{
		userStore: userStore,
	}
}

func (a *Authenticator) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		//Grab the cookie from the session
		userTok, err := c.Cookie("sessionToken")
		if err != nil {
			log.Debug(c.Request().Context(), "No session token for protected route", "ip", c.RealIP(), "path", c.Request().URL.Path, "method", c.Request().Method)
			return c.Redirect(http.StatusSeeOther, "/login")
		}
		//Check if the cookie is valid
		isValid, err := a.userStore.ValidateSessionToken(c.Request().Context(), userTok.Value)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to validate session token", "ip", c.RealIP(), "path", c.Request().URL.Path, "error", err)
			return c.Redirect(http.StatusSeeOther, "/login")
		}

		if isValid {
			userUuid, err := a.userStore.GetUserBySessionToken(c.Request().Context(), userTok.Value)
			if err != nil {
				log.Error(c.Request().Context(), "Failed to get user by session token", "ip", c.RealIP(), "path", c.Request().URL.Path, "error", err)
				return c.Redirect(http.StatusSeeOther, "/login")
			}
			c.Set(string(UserUuidKey), userUuid)
			metrics.RecordUserActivity(userUuid.String())
			metrics.RecordAuthenticatedRequest(c.Request().Method, c.Path())
			log.Debug(c.Request().Context(), "User authenticated for protected route", "userUuid", userUuid, "ip", c.RealIP(), "path", c.Request().URL.Path, "method", c.Request().Method)
		} else {
			log.Warn(c.Request().Context(), "Invalid session token for protected route", "ip", c.RealIP(), "path", c.Request().URL.Path, "method", c.Request().Method)
			return c.Redirect(http.StatusSeeOther, "/login")
		}

		return next(c)
	}
}

func (a *Authenticator) CheckAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userUuidVal := c.Get(string(UserUuidKey))
		if userUuidVal == nil {
			log.Warn(c.Request().Context(), "Could not get user uuid from context trying to reach admin page", "ip", c.RealIP(), "path", c.Request().URL.Path)
			return c.Redirect(http.StatusSeeOther, "/u/home")
		}
		userUuid := userUuidVal.(uuid.UUID)

		isAdmin, err := a.userStore.UserIsAdmin(c.Request().Context(), userUuid)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to check admin status", "userUuid", userUuid, "ip", c.RealIP(), "path", c.Request().URL.Path, "error", err)
			return c.Redirect(http.StatusSeeOther, "/u/home")
		}
		c.Set(string(IsAdminKey), isAdmin)
		metrics.RecordUserActivity(userUuid.String())
		metrics.RecordAuthenticatedRequest(c.Request().Method, c.Path())
		log.Debug(c.Request().Context(), "Admin check completed", "userUuid", userUuid, "isAdmin", isAdmin, "path", c.Request().URL.Path)

		if isAdmin {
			log.Info(c.Request().Context(), "User accessed admin page", "userUuid", userUuid, "ip", c.RealIP(), "path", c.Request().URL.Path)
		} else {
			log.Warn(c.Request().Context(), "User did not have access to admin page", "userUuid", userUuid, "ip", c.RealIP(), "path", c.Request().URL.Path)
			return c.Redirect(http.StatusSeeOther, "/u/home")
		}

		return next(c)
	}
}

type MetricAuth struct {
	secret string
}

func NewMetricAuth(secret string) *MetricAuth {
	if secret == "" {
		panic("usage: METRIC_SECRET environment variable not set")
	}

	return &MetricAuth{
		secret: secret,
	}
}

func (m *MetricAuth) MetricsAuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth := c.Request().Header.Get("Authorization")

			if auth == "" {
				log.Warn(c.Request().Context(), "Metrics auth failed", "ip", c.RealIP(), "reason", "missing_header")
				return c.NoContent(http.StatusUnauthorized)
			}

			// Expect: "Bearer <token>"
			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				log.Warn(c.Request().Context(), "Metrics auth failed", "ip", c.RealIP(), "reason", "malformed_header")
				return c.NoContent(http.StatusForbidden)
			}

			if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(m.secret)) != 1 {
				log.Warn(c.Request().Context(), "Metrics auth failed", "ip", c.RealIP(), "reason", "invalid_token")
				return c.NoContent(http.StatusForbidden)
			}

			return next(c)
		}
	}
}
