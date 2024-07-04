package handler

import (
	"server/assert"
	"server/view"

	"github.com/labstack/echo/v4"
)


func (h *Handler) HandleViewHome(c echo.Context) error {
    homeIndex := view.HomeIndex(false)
    home := view.Home(" | Draft Overview", false, homeIndex)
    //TODO We should probably make tailwind work offline to make the dev experience better
    err := Render(c, home)
    assert.NoErrorCF(err, "Handle View Home Failed To Render")
    return nil
}
