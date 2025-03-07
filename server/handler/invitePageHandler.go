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
    return renderInviteTable(h, c, false, "")
}

func renderInviteTable(h *Handler, c echo.Context, hasError bool, errorMessage string) error {
    assert := assert.CreateAssertWithContext("Render Invite Table")

    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userId)

    invites := model.GetInvites(h.Database, userId)

    inviteIndex := draftView.DraftInviteIndex(invites, hasError, errorMessage)
    inviteView := draftView.DraftInvite(" | Draft Invites", true, username, inviteIndex)
    err = Render(c, inviteView)
    return err
}

func (h *Handler) HandleAcceptInvite(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Handle Accept Invite")

    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    inviteId, err := strconv.Atoi(c.FormValue("inviteId"))
    assert.RunAssert(inviteId != 0, "Invite Id Should Never Be 0")
    assert.NoError(err, "Failed to parse invite id")
    invite := model.GetInvite(h.Database, inviteId)

    //Make sure that other players cannot accept someones draft
    if invite.InvitedPlayer != userId {
        h.Logger.Log(fmt.Sprintf("Invited Player: %d User ID: %d", invite.InvitedPlayer, userId))
        return renderInviteTable(h, c, true, "You are not allowed to accept drafts for other players.")
    }

    h.Logger.Log(fmt.Sprintf("Accepting invite %d from player %d", inviteId, userId))
    // if more than 8 players are invites then we cancel the other outstanding invites
    // Maybe we need an active bool

    // Check that accepting this invite will not lead to more than eight players being in the draft
    numPlayers := model.GetNumPlayersInInvitedDraft(h.Database, inviteId)
    if numPlayers >= 8 {
        model.CancelOutstandingInvites(h.Database, invite.DraftId)
        return renderInviteTable(h, c, true, "Too many players are already in the draft. Please contect the draft owner if you think this is an error.")
    }

    draftId, playerId := model.AcceptInvite(h.Database, inviteId)
    model.AddPlayerToDraft(h.Database, draftId, playerId)

    if numPlayers >= 7 {
        model.CancelOutstandingInvites(h.Database, invite.DraftId)
    }

    return renderInviteTable(h, c, false, "")
}
