package handler

import (
	"net/http"
	"server/log"
	"server/view"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewHome(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)

	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	log.Info(c.Request().Context(), "Loading drafts for user", "Username", username)
	drafts, err := h.DraftStore.GetDraftsForUser(c.Request().Context(), userUuid)
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
