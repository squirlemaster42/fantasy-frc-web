package handler

import (
	"log/slog"
	"net/http"
	"server/assert"
	"server/model"
	"server/view"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewHome(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Home View")

	//Grab the cookie from the session
	userTok, err := c.Cookie("sessionToken")
	if err != nil {
		slog.Warn("Failed login", "Ip", c.RealIP())
		err = c.Redirect(http.StatusSeeOther, "/login")
		if err != nil {
			return err
		}
		return echo.ErrUnauthorized
	}
	//Check if the cookie is valid
	isValid := model.ValidateSessionToken(h.Database, userTok.Value)

	if isValid {
		//If the cookie is valid we let the request through
		//We should probaly log a message
		model.GetUserBySessionToken(h.Database, userTok.Value)
	} else {
		//If the cookie is not valid then we redirect to the login page
		slog.Warn("Failed login", "Ip", c.RealIP())
		err = c.Redirect(http.StatusSeeOther, "/login")
		if err != nil {
			return err
		}
		return echo.ErrUnauthorized
	}

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userUuid)

	slog.Info("Loading drafts for user", "Username", username)
	drafts, err := model.GetDraftsForUser(h.Database, userUuid)
	if err != nil {
		slog.Error("Failed to load drafts for user", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to load drafts")
	}
	slog.Info("Loaded drafts for user", "Username", username)

	homeIndex := view.HomeIndex(&drafts, userUuid)
	home := view.Home(" | Draft Overview", true, username, homeIndex)
	err = Render(c, home)
	assert.NoError(err, "Handle View Home Failed To Render")
	slog.Info("Rendered home page for user", "Username", username)
	return nil
}
