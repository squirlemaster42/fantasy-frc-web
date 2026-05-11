package handler

import (
	"net/http"
	"server/assert"
	"server/log"
	"server/view"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewHome(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Home View")

	//Grab the cookie from the session
	userTok, err := c.Cookie("sessionToken")
	if err != nil {
		log.Warn(c.Request().Context(), "Failed login", "Ip", c.RealIP())
		err = c.Redirect(http.StatusSeeOther, "/login")
		if err != nil {
			return err
		}
		return echo.ErrUnauthorized
	}

	//Check if the cookie is valid
	isValid := h.UserStore.ValidateSessionToken(c.Request().Context(), userTok.Value)

	if isValid {
		//If the cookie is valid we let the request through
		//We should probaly log a message
		h.UserStore.GetUserBySessionToken(c.Request().Context(), userTok.Value)
	} else {
		//If the cookie is not valid then we redirect to the login page
		log.Warn(c.Request().Context(), "Failed login", "Ip", c.RealIP())
		err = c.Redirect(http.StatusSeeOther, "/login")
		if err != nil {
			return err
		}
		return echo.ErrUnauthorized
	}

	userUuid := h.UserStore.GetUserBySessionToken(c.Request().Context(), userTok.Value)
	username := h.UserStore.GetUsername(c.Request().Context(), userUuid)

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
	// TODO should we crash here
	assert.NoError(c.Request().Context(), err, "Handle View Home Failed To Render")
	log.Info(c.Request().Context(), "Rendered home page for user", "Username", username)
	return nil
}
