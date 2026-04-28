package handler

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/labstack/echo/v4"

	"server/assert"
	"server/log"
	"server/model"
	draftView "server/view/draft"
)

func (h *Handler) HandleViewInvites(c echo.Context) error {
	return renderInviteTable(h, c, false, "", true)
}

func renderInviteTable(h *Handler, c echo.Context, hasError bool, errorMessage string, includeWrapper bool) error {
	assert := assert.CreateAssertWithContext("Render Invite Table")

	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userUuid)

	invites := model.GetInvites(h.Database, userUuid)

	inviteIndex := draftView.DraftInviteIndex(invites, hasError, errorMessage)
	if includeWrapper {
		inviteView := draftView.DraftInvite(" | Draft Invites", true, username, inviteIndex)
		err = Render(c, inviteView)
	} else {
		err = Render(c, inviteIndex)
	}

	return err
}

func (h *Handler) HandleAcceptInvite(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Accept Invite")

	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	inviteIdStr := c.FormValue("inviteId")
	log.Info(c.Request().Context(), "Got request to accept invite", "User", userUuid, "Invite Id", inviteIdStr)
	inviteId, err := strconv.Atoi(inviteIdStr)
	assert.RunAssert(inviteId != 0, "Invite Id Should Never Be 0")
	assert.NoError(err, "Failed to parse invite id")
	invite, err := model.GetInvite(h.Database, inviteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return renderInviteTable(h, c, true, "Invite not found. It may have been cancelled or expired.", false)
		}
		log.Error(c.Request().Context(), "Failed to get invite", "error", err, "inviteId", inviteId)
		return renderInviteTable(h, c, true, "An error occurred. Please try again.", false)
	}

	//Make sure that other players cannot accept someones draft
	if invite.InvitedUserUuid != userUuid {
		log.Info(c.Request().Context(), "Invited player to draft", "Invited User Uuid", invite.InvitedUserUuid, "Inviting User Uuid", userUuid)
		return renderInviteTable(h, c, true, "You are not allowed to accept drafts for other players.", false)
	}

	log.Info(c.Request().Context(), "Accepting invite from player", "Invite Id", inviteId, "User Id", userUuid)

	// If more than 8 players are invites then we cancel the other outstanding invites
	// Maybe we need an active bool
	// Check that accepting this invite will not lead to more than eight players being in the draft
	numPlayers := model.GetNumPlayersInInvitedDraft(h.Database, inviteId)
	if numPlayers >= 8 {
		if err := model.CancelOutstandingInvites(h.Database, invite.DraftId); err != nil {
			log.Error(c.Request().Context(), "Failed to cancel outstanding invites", "error", err, "draftId", invite.DraftId)
		}
		return renderInviteTable(h, c, true, "Too many players are already in the draft. Please contect the draft owner if you think this is an error.", false)
	}

	draftId, playerId := model.AcceptInvite(h.Database, inviteId)
	model.AddPlayerToDraft(h.Database, draftId, playerId)

	if numPlayers >= 7 {
		if err := model.CancelOutstandingInvites(h.Database, invite.DraftId); err != nil {
			log.Error(c.Request().Context(), "Failed to cancel outstanding invites", "error", err, "draftId", invite.DraftId)
		}
	}

	return renderInviteTable(h, c, false, "", false)
}

func (h *Handler) HandleDeclineInvite(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Decline Invite")

	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	inviteIdStr := c.FormValue("inviteId")
	log.Info(c.Request().Context(), "Got request to decline invite", "User", userUuid, "Invite Id", inviteIdStr)
	inviteId, err := strconv.Atoi(inviteIdStr)
	assert.RunAssert(inviteId != 0, "Invite Id Should Never Be 0")
	assert.NoError(err, "Failed to parse invite id")
	invite, err := model.GetInvite(h.Database, inviteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return renderInviteTable(h, c, true, "Invite not found. It may have been cancelled or expired.", false)
		}
		log.Error(c.Request().Context(), "Failed to get invite", "error", err, "inviteId", inviteId)
		return renderInviteTable(h, c, true, "An error occurred. Please try again.", false)
	}

	if invite.InvitedUserUuid != userUuid {
		log.Info(c.Request().Context(), "User attempted to decline invite for another player", "Invited User Uuid", invite.InvitedUserUuid, "Requesting User Uuid", userUuid)
		return renderInviteTable(h, c, true, "You are not allowed to decline invites for other players.", false)
	}

	err = model.CancelInvite(h.Database, inviteId)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to cancel invite", "error", err, "inviteId", inviteId)
		return renderInviteTable(h, c, true, "An error occurred while declining the invite. Please try again.", false)
	}

	return renderInviteTable(h, c, false, "", false)
}
