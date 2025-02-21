package handler

import (
	"github.com/labstack/echo/v4"

	"server/assert"
	"server/model"
	"server/view/admin"
)

func (h *Handler) HandleAdminConsoleGet(c echo.Context) error {
    h.Logger.Log("Got request to render admin console")
    assert := assert.CreateAssertWithContext("Handle Admin Console Get")
    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userId)

    adminConsoleIndex := admin.AdminConsoleIndex()
    adminConsole := admin.AdminConsole(" | Admin Console", true, username, adminConsoleIndex)
    Render(c, adminConsole)

    return nil
}
