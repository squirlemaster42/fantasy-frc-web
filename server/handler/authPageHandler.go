package handler

import (
	"fmt"
	"server/assert"
	"server/view/login"

	"github.com/labstack/echo/v4"
)

func HandleViewLogin(c echo.Context) error {
    fmt.Println("Handle Login View Get")
    loginIndex := login.LoginIndex(false, "")
    login := login.Login(" | Login", false, loginIndex)
    err := Render(c, login)
    assert.NoErrorCF(err, "Handle View Login Failed To Render")
    return nil
}
