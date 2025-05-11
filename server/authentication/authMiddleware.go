package authentication

import (
	"database/sql"
	"log/slog"
	"net/http"
	"server/model"

	"github.com/labstack/echo/v4"
)

type Authenticator struct {
    database *sql.DB
    //Maybe we want a user cache here that we can give a session
    //Token to and get back the user
}

func NewAuth(db *sql.DB) *Authenticator {
    return &Authenticator {
        database: db,
    }
}

func  (a *Authenticator) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
    return func (c echo.Context) error {
        //Grab the cookie from the session
        userTok, err := c.Cookie("sessionToken")
        if err != nil {
            slog.Warn("Failed login", "Ip", c.RealIP())
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
            userId := model.GetUserBySessionToken(a.database, userTok.Value)
            slog.Info("User has successfully logged in", "User Id", userId, "Ip", c.RealIP())
        } else {
            //If the cookie is not valid then we redirect to the login page
            slog.Warn("Failed login", "Ip", c.RealIP())
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

func  (a *Authenticator) CheckAdmin(next echo.HandlerFunc) echo.HandlerFunc {
    return func (c echo.Context) error {
        isAdmin := false

        //Grab the cookie from the session
        userTok, err := c.Cookie("sessionToken")
        if err != nil {
            slog.Warn("Could not get session token from request trying to reach admin page", "Ip", c.RealIP())
            err := c.Redirect(http.StatusSeeOther, "/u/home")
            if err != nil {
                return err
            }
            return echo.ErrUnauthorized
        }

        //Check if the cookie is valid
        userId := model.GetUserBySessionToken(a.database, userTok.Value)
        isAdmin = model.UserIsAdmin(a.database, userId)
        slog.Info("User is admin?", "User Id", userId, "Is Admin", isAdmin)

        if isAdmin {
            //If the cookie is valid we let the request through
            //We should probaly log a message
            slog.Info("User from ip has accessed the admin page", "User Id", userId, "Ip",  c.RealIP())
        } else {
            //If the cookie is not valid then we redirect to the login page
            slog.Info("User from ip did not have access to the admin page", "User Id", userId, "Ip", c.RealIP())
            err := c.Redirect(http.StatusSeeOther, "/u/home")
            if err != nil {
                return err
            }
            return echo.ErrUnauthorized
        }

        return next(c)
    }
}

