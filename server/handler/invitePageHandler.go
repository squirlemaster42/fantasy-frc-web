package handler

import (
	"github.com/labstack/echo/v4"

	"server/assert"
	"server/model"
	draftView "server/view/draft"
)

func (h *Handler) HandleViewInvites(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Handle View Invites")

    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userId)

    invites := model.GetInvites(h.Database, userId)

    inviteIndex := draftView.DraftInviteIndex(invites)
    inviteView := draftView.DraftInvite(" | Draft Invites", true, username, inviteIndex)
    err = Render(c, inviteView)
    return err
}
