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
    assert.NoErrorCF(err, "Failed to get user token")

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userId)

    h.Logger.Log(fmt.Sprintf("Loading drafts for user: %s", username))

    drafts := model.GetDraftsForUser(h.Database, userId)

    homeIndex := view.HomeIndex(drafts)
    home := view.Home(" | Draft Overview", true, username, homeIndex)
    err = Render(c, home)
    assert.NoErrorCF(err, "Handle View Home Failed To Render")
    return nil
}
