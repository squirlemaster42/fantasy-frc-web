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
			log.Warn(c.Request().Context(), "Failed to get session token when trying to login", "Ip", c.RealIP())
			return c.Redirect(http.StatusSeeOther, "/login")
		}
		//Check if the cookie is valid
		isValid, err := a.userStore.ValidateSessionToken(c.Request().Context(), userTok.Value)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to validate session token", "Ip", c.RealIP(), "Error", err)
			return c.Redirect(http.StatusSeeOther, "/login")
		}

		if isValid {
			userUuid, err := a.userStore.GetUserBySessionToken(c.Request().Context(), userTok.Value)
			if err != nil {
				log.Error(c.Request().Context(), "Failed to get user by session token", "Ip", c.RealIP(), "Error", err)
				return c.Redirect(http.StatusSeeOther, "/login")
			}
			c.Set(string(UserUuidKey), userUuid)
			metrics.RecordUserActivity(userUuid.String())
			metrics.RecordAuthenticatedRequest(c.Request().Method, c.Path())
			log.Info(c.Request().Context(), "User has successfully logged in", "User userUuid", userUuid, "Ip", c.RealIP())
		} else {
			log.Warn(c.Request().Context(), "Invalid login request", "Ip", c.RealIP())
			return c.Redirect(http.StatusSeeOther, "/login")
		}

		return next(c)
	}
}

func (a *Authenticator) CheckAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userUuidVal := c.Get(string(UserUuidKey))
		if userUuidVal == nil {
			log.Warn(c.Request().Context(), "Could not get user uuid from context trying to reach admin page", "Ip", c.RealIP())
			return c.Redirect(http.StatusSeeOther, "/u/home")
		}
		userUuid := userUuidVal.(uuid.UUID)

		isAdmin, err := a.userStore.UserIsAdmin(c.Request().Context(), userUuid)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to check admin status", "User Id", userUuid, "Ip", c.RealIP(), "Error", err)
			return c.Redirect(http.StatusSeeOther, "/u/home")
		}
		c.Set(string(IsAdminKey), isAdmin)
		metrics.RecordUserActivity(userUuid.String())
		metrics.RecordAuthenticatedRequest(c.Request().Method, c.Path())
		log.Info(c.Request().Context(), "User is admin?", "User Id", userUuid, "Is Admin", isAdmin)

		if isAdmin {
			log.Info(c.Request().Context(), "User from ip has accessed the admin page", "User Uuid", userUuid, "Ip", c.RealIP())
		} else {
			log.Info(c.Request().Context(), "User from ip did not have access to the admin page", "User Uuid", userUuid, "Ip", c.RealIP())
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
				return c.NoContent(http.StatusUnauthorized)
			}

			// Expect: "Bearer <token>"
			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				return c.NoContent(http.StatusForbidden)
			}

			if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(m.secret)) != 1 {
				return c.NoContent(http.StatusForbidden)
			}

			return next(c)
		}
	}
}
