package handler

import (
	"fmt"
	"server/assert"
	"server/model"
	"server/view"

	"github.com/labstack/echo/v4"
)


func (h *Handler) HandleViewHome(c echo.Context) error {
    userTok, err := c.Cookie("sessionToken")
    //TODO We should have already checked that the user has a token
    //here since they should not be able to access the page otherwise
    //There might be some sort of weird thing here where the middleware
    //validates the session token is good and then it expires a second later
    assert.NoErrorCF(err, "Failed to get user token")

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)

    h.Logger.Log(fmt.Sprintf("Loading drafts for user: %s", model.GetUsername(h.Database, userId)))

    drafts := model.GetDraftsForUser(h.Database, userId)

    homeIndex := view.HomeIndex(drafts)
    home := view.Home(" | Draft Overview", false, homeIndex)
    //TODO We should probably make tailwind work offline to make the dev experience better
    err = Render(c, home)
    assert.NoErrorCF(err, "Handle View Home Failed To Render")
    return nil
}
