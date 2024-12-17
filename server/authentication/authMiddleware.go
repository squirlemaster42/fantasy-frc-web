package authentication

import (
	"database/sql"
	"fmt"
	"net/http"
	"server/logging"
	"server/model"

	"github.com/labstack/echo/v4"
)

type Authenticator struct {
    database *sql.DB
    logger *logging.Logger
    //Maybe we want a user cache here that we can give a session
    //Token to and get back the user
}

func NewAuth(db *sql.DB, logger *logging.Logger) *Authenticator {
    return &Authenticator {
        database: db,
        logger: logger,
    }
}

func  (a *Authenticator) Authenticate(next echo.HandlerFunc) echo.HandlerFunc {
    return func (c echo.Context) error {
        isValid := true

        //Grab the cookie from the session
        userTok, err := c.Cookie("sessionToken")
        if err != nil {
            a.logger.Log(fmt.Sprintf("Failed login from ip %s", c.RealIP()))
            c.Redirect(http.StatusSeeOther, "/login")
            return echo.ErrUnauthorized
        }
        //Check if the cookie is valid
        isValid = model.ValidateSessionToken(a.database, userTok.Value)

        if isValid {
            //If the cookie is valid we let the request through
            //We should probaly log a message
            userId := model.GetUserBySessionToken(a.database, userTok.Value)
            a.logger.Log(fmt.Sprintf("User %d has successfully logged in from ip %s", userId, c.RealIP()))
        } else {
            //If the cookie is not valid then we redirect to the login page
            a.logger.Log(fmt.Sprintf("Failed login from ip %s", c.RealIP()))
            c.Redirect(http.StatusSeeOther, "/login")
            return echo.ErrUnauthorized
        }

        return next(c)
    }
}


