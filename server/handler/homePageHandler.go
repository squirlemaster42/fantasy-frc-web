package handler

import (
	"fmt"
	"net/http"
	"server/assert"
	"server/model"
	"server/view"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewHome(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Home View")

    isValid := true

    //Grab the cookie from the session
    userTok, err := c.Cookie("sessionToken")
    if err != nil {
        h.Logger.Log(fmt.Sprintf("Failed login from ip %s", c.RealIP()))
        c.Redirect(http.StatusSeeOther, "/login")
        return echo.ErrUnauthorized
    }
    //Check if the cookie is valid
    isValid = model.ValidateSessionToken(h.Database, userTok.Value)

    if isValid {
        //If the cookie is valid we let the request through
        //We should probaly log a message
        userId := model.GetUserBySessionToken(h.Database, userTok.Value)
        h.Logger.Log(fmt.Sprintf("User %d has successfully logged in from ip %s", userId, c.RealIP()))
    } else {
        //If the cookie is not valid then we redirect to the login page
        h.Logger.Log(fmt.Sprintf("Failed login from ip %s", c.RealIP()))
        c.Redirect(http.StatusSeeOther, "/login")
        return echo.ErrUnauthorized
    }

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userId)

    h.Logger.Log(fmt.Sprintf("Loading drafts for user: %s", username))
	drafts := model.GetDraftsForUser(h.Database, userId)
    h.Logger.Log(fmt.Sprintf("Loaded drafts for user: %s", username))

	homeIndex := view.HomeIndex(drafts)
	home := view.Home(" | Draft Overview", true, username, homeIndex)
	err = Render(c, home)
	assert.NoError(err, "Handle View Home Failed To Render")
    h.Logger.Log(fmt.Sprintf("Rendered home page for user: %s", username))
	return nil
}
