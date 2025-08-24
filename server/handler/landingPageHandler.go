package handler

import (
	"server/assert"
	"server/view"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewLanding(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Landing View")

	landing := view.Landing()
	err := Render(c, landing)
	assert.NoError(err, "Handle View Landing Failed To Render")
	return nil
}
