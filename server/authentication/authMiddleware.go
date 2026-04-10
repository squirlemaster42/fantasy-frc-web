package authentication

import (
	"database/sql"
	"net/http"
	"server/log"
	"server/model"
	"strings"

	"github.com/labstack/echo/v4"
)

type Authenticator struct {
	database *sql.DB
	//Maybe we want a user cache here that we can give a session
	//Token to and get back the user
}

func NewAuth(db *sql.DB) *Authenticator {
	return &Authenticator{
		database: db,
	}
}

func (a *Authenticator) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		//Grab the cookie from the session
		userTok, err := c.Cookie("sessionToken")
		if err != nil {
			log.WarnNoContext("Failed to get session token when trying to login", "Ip", c.RealIP(), "Error", err)
			err := c.Redirect(http.StatusSeeOther, "/login")
			if err != nil {
				return err
			}
			return echo.ErrUnauthorized
		}
		//Check if the cookie is valid
		isValid := model.ValidateSessionToken(a.database, userTok.Value)

		if isValid {
			//If the cookie is valid we let the request through
			//We should probaly log a message
			userUuid := model.GetUserBySessionToken(a.database, userTok.Value)
			log.InfoNoContext("User has successfully logged in", "User userUuid", userUuid, "Ip", c.RealIP())
		} else {
			//If the cookie is not valid then we redirect to the login page
			log.WarnNoContext("Invalid login request", "Ip", c.RealIP())
			err := c.Redirect(http.StatusSeeOther, "/login")
			if err != nil {
				return err
			}
			return echo.ErrUnauthorized
		}

		return next(c)
	}
}

type AdminAuth struct {
	database *sql.DB
}

func NewAdminAuth(db *sql.DB) *AdminAuth {
	return &AdminAuth{
		database: db,
	}
}

func (a *Authenticator) CheckAdmin(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		isAdmin := false

		//Grab the cookie from the session
		userTok, err := c.Cookie("sessionToken")
		if err != nil {
			log.WarnNoContext("Could not get session token from request trying to reach admin page", "Ip", c.RealIP())
			err := c.Redirect(http.StatusSeeOther, "/u/home")
			if err != nil {
				return err
			}
			return echo.ErrUnauthorized
		}

		//Check if the cookie is valid
		userUuid := model.GetUserBySessionToken(a.database, userTok.Value)
		isAdmin = model.UserIsAdmin(a.database, userUuid)
		log.InfoNoContext("User is admin?", "User Id", userUuid, "Is Admin", isAdmin)

		if isAdmin {
			//If the cookie is valid we let the request through
			//We should probaly log a message
			log.InfoNoContext("User from ip has accessed the admin page", "User Uuid", userUuid, "Ip", c.RealIP())
		} else {
			//If the cookie is not valid then we redirect to the login page
			log.InfoNoContext("User from ip did not have access to the admin page", "User Uuid", userUuid, "Ip", c.RealIP())
			err := c.Redirect(http.StatusSeeOther, "/u/home")
			if err != nil {
				return err
			}
			return echo.ErrUnauthorized
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
			if len(parts) != 2 || parts[0] != "Bearer" || parts[1] != m.secret {
				return c.NoContent(http.StatusForbidden)
			}

			return next(c)
		}
	}
}
