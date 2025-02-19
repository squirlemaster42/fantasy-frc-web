package handler

import (
	"strconv"

	"github.com/labstack/echo/v4"

	"server/assert"
	"server/model"
	draftView "server/view/draft"
)

func (h *Handler) HandleViewInvites(c echo.Context) error {
    return renderInviteTable(h, c)
}

func renderInviteTable(h *Handler, c echo.Context) error {
    assert := assert.CreateAssertWithContext("Render Invite Table")

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

func (h *Handler) HandleAcceptInvite(c echo.Context) error {
    inviteId, err := strconv.Atoi(c.FormValue("inviteId"))
    assert.NoErrorCF(err, "Failed to parse invite id")
    //TODO We need to make sure that we dont accpet more than 8 players
    // if more than 8 players are invites then we cancel the other outstanding invites
    // Maybe we need an active bool
    model.AcceptInvite(h.Database, inviteId)

    return renderInviteTable(h, c)
}
