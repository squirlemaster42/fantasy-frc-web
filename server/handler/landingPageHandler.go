package handler

import (
	"net/http"
	"server/log"
	"server/view"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewLanding(c echo.Context) error {
	landing := view.Landing()
	err := Render(c, landing)
	if err != nil {
		log.Error(c.Request().Context(), "Handle View Landing Failed To Render", "error", err)
		return c.String(http.StatusInternalServerError, DefaultErrorMessage)
	}
	return nil
}
