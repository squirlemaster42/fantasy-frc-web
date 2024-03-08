package handler

import (
    "server/web/view/login"
    "github.com/labstack/echo/v4"
)

type LoginHandler struct {

}

func (l *LoginHandler) HandleLoginShow(c echo.Context) error {
    return render(c, login.Show())
}
