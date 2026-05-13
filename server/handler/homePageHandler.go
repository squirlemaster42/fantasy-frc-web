package handler

import (
	"net/http"
	"server/log"
	"server/model"
	"server/view"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewHome(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)

	username := model.GetUsername(c.Request().Context(), h.Database, userUuid)

	log.Info(c.Request().Context(), "Loading drafts for user", "Username", username)
	drafts, err := model.GetDraftsForUser(c.Request().Context(), h.Database, userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to load drafts for user", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load drafts")
	}
	log.Info(c.Request().Context(), "Loaded drafts for user", "Username", username)

	homeIndex := view.HomeIndex(&drafts, userUuid)
	home := view.Home(" | Draft Overview", true, username, homeIndex)
	err = Render(c, home)
	if err != nil {
		log.Error(c.Request().Context(), "Handle View Home Failed To Render", "Error", err)
	}
	log.Info(c.Request().Context(), "Rendered home page for user", "Username", username)
	return nil
}
