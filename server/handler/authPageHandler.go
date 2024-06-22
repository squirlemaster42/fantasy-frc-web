package handler

import (
	"server/assert"
	"server/view/login"

	"github.com/labstack/echo/v4"
)

func HandleViewLogin(c echo.Context) error {
    loginIndex := login.LoginIndex(false, "")
    login := login.Login(" | Login", false, loginIndex)
    err := Render(c, login)
    assert.NoErrorCF(err, "Handle View Login Failed To Render")
    return nil
}

func HandleLoginPost(c echo.Context) error {
    return nil
}
