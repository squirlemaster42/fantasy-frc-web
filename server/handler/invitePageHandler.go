package handler

import (
	"fmt"
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
    assert := assert.CreateAssertWithContext("Handle Accept Invite")

    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")
    // Hi future self, you were working in accepting invites.
    // The current issue is that inviteId is 0

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    inviteId, err := strconv.Atoi(c.FormValue("inviteId"))
    assert.RunAssert(inviteId != 0, "Invite Id Should Never Be 0")
    assert.NoError(err, "Failed to parse invite id")
    //TODO Need to make sure that the player with the invite outstanding is the one who is accepting it
    h.Logger.Log(fmt.Sprintf("Accepting invite %d from player %d", inviteId, userId))
    //TODO We need to make sure that we dont accpet more than 8 players
    // if more than 8 players are invites then we cancel the other outstanding invites
    // Maybe we need an active bool
    model.AcceptInvite(h.Database, inviteId)

    return renderInviteTable(h, c)
}
