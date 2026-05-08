package handler

import (
	"net/http"
	"server/assert"
	"server/log"
	"server/model"
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
	isValid := model.ValidateSessionToken(c.Request().Context(), h.Database, userTok.Value)

	if isValid {
		//If the cookie is valid we let the request through
		//We should probaly log a message
		model.GetUserBySessionToken(c.Request().Context(), h.Database, userTok.Value)
	} else {
		//If the cookie is not valid then we redirect to the login page
		log.Warn(c.Request().Context(), "Failed login", "Ip", c.RealIP())
		err = c.Redirect(http.StatusSeeOther, "/login")
		if err != nil {
			return err
		}
		return echo.ErrUnauthorized
	}

	userUuid := model.GetUserBySessionToken(c.Request().Context(), h.Database, userTok.Value)
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
	// TODO should we crash here
	assert.NoError(c.Request().Context(), err, "Handle View Home Failed To Render")
	log.Info(c.Request().Context(), "Rendered home page for user", "Username", username)
	return nil
}
