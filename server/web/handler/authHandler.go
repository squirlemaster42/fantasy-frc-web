package handler

import (
	"fmt"
	"log"
	"server/database"
	"server/web/view"
	"server/web/view/login"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)


type HomeHandler struct {

}

func (h *HomeHandler) HandleViewHome (c echo.Context) error {
    homeIndex := view.HomeIndex(false) //TODO Figure out how to set this?
    home := view.Home("", false, homeIndex)

    return render(c, home)
}

type LoginHandler struct {
    DbHandler *database.DatabaseDriver
}

func (l *LoginHandler) HandleViewLogin (c echo.Context) error {
    loginIndex := login.LoginIndex(false)
    login := login.Login(" | Login", false, loginIndex)

    var err error
    if c.Request().Method == "POST" {
        err = c.Request().ParseForm()
        if err != nil {
            log.Print(err)
            return err
        }
        _ = c.FormValue("email")
        _ = c.FormValue("password")

        ses, err := session.Get("session", c)
        fmt.Println(ses.Values["foo"])
        if err != nil {
            return err
        }

        ses.Options = &sessions.Options{
            MaxAge: 86400 * 7,
        }
        ses.Values["foo"] = "bar"
        ses.Save(c.Request(), c.Response())
    }

    err = render(c, login)
    if err != nil {
        return err
    }

    return nil
}

type RegistrationHandler struct {
    DbHandler *database.DatabaseDriver
}

func (r *RegistrationHandler) HandleViewRegister (c echo.Context) error {
    registerIndex := login.RegisterIndex(false)
    register := login.Register(" | Register", false, registerIndex)


    var err error
    if c.Request().Method == "POST" {
        err = c.Request().ParseForm()
        if err != nil {
            log.Print(err)
            return err
        }
        _ = c.FormValue("email")
        _ = c.FormValue("password")
    }

    err = render(c, register)
    if err != nil {
        return err
    }

    return nil
}
