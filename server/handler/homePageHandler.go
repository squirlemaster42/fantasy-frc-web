package handler

import (
	"net/http"
	"server/log"
	"server/view"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewHome(c echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	log.Debug(c.Request().Context(), "Loading drafts for user", "username", username)
	drafts, err := h.Stores.DraftStore.GetDraftsForUser(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to load drafts for user", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load drafts")
	}
	log.Debug(c.Request().Context(), "Loaded drafts for user", "username", username)

	homeIndex := view.HomeIndex(&drafts, userUuid)
	home := view.Home("Draft Overview", true, username, homeIndex)
	if err := Render(c, home); err != nil {
		log.Error(c.Request().Context(), "Handle View Home Failed To Render", "error", err)
		return err
	}
	return nil
}
