package handler

import (
	"server/assert"
	"server/view"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewLanding(c echo.Context) error {
	landingIndex := view.LandingIndex()
	landing := view.Landing(" | Draft Overview", false, landingIndex)
	//TODO We should probably make tailwind work offline to make the dev experience better
	err := Render(c, landing)
	assert.NoErrorCF(err, "Handle View Landing Failed To Render")
	return nil
}
