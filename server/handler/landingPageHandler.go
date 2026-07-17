package handler

import (
	"net/http"
	"server/log"
	"server/view"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) getAuthenticatedUser(c echo.Context) (uuid.UUID, string, bool) {
	userTok, err := c.Cookie("sessionToken")
	if err != nil {
		return uuid.UUID{}, "", false
	}

	isValid, err := h.UserStore.ValidateSessionToken(c.Request().Context(), userTok.Value)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to validate session token for landing page", "ip", c.RealIP(), "error", err)
		return uuid.UUID{}, "", false
	}
	if !isValid {
		return uuid.UUID{}, "", false
	}

	userUuid, err := h.UserStore.GetUserBySessionToken(c.Request().Context(), userTok.Value)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get user by session token for landing page", "ip", c.RealIP(), "error", err)
		return uuid.UUID{}, "", false
	}

	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username for landing page", "userUuid", userUuid, "error", err)
		return userUuid, "", true
	}

	return userUuid, username, true
}

func (h *Handler) HandleViewLanding(c echo.Context) error {
	_, username, fromProtected := h.getAuthenticatedUser(c)
	landing := view.Landing(fromProtected, username)
	err := Render(c, landing)
	if err != nil {
		log.Error(c.Request().Context(), "Handle View Landing Failed To Render", "error", err)
		return c.String(http.StatusInternalServerError, "Unable to render page")
	}
	return nil
}
