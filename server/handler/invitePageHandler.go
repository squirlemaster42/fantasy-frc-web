package handler

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"server/log"
	draftView "server/view/draft"
)

func (h *Handler) HandleViewInvites(c echo.Context) error {
	return renderInviteTable(h, c, false, "", true)
}

func renderInviteTable(h *Handler, c echo.Context, hasError bool, errorMessage string, includeWrapper bool) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "Error", err)
		username = ""
	}

	invites, err := h.DraftStore.GetInvites(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get invites", "Error", err)
		return err
	}

	inviteIndex := draftView.DraftInviteIndex(invites, hasError, errorMessage, h.csrfToken(c))
	if includeWrapper {
		inviteView := draftView.DraftInvite(" | Draft Invites", true, username, inviteIndex)
		err = Render(c, inviteView)
	} else {
		err = Render(c, inviteIndex)
	}

	return err
}

func (h *Handler) HandleAcceptInvite(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	inviteIdStr := c.FormValue("inviteId")
	log.Info(c.Request().Context(), "Got request to accept invite", "User", userUuid, "Invite Id", inviteIdStr)
	inviteId, err := strconv.Atoi(inviteIdStr)
	if err != nil || inviteId == 0 {
		log.Warn(c.Request().Context(), "Failed to parse invite id", "Invite Id", inviteIdStr, "Error", err)
		return renderInviteTable(h, c, true, "Invalid invite ID.", false)
	}

	invite, err := h.DraftStore.GetInvite(c.Request().Context(), inviteId)
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
	numPlayers, err := h.DraftStore.GetNumPlayersInInvitedDraft(c.Request().Context(), inviteId)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get num players in invited draft", "error", err, "inviteId", inviteId)
		return renderInviteTable(h, c, true, "An error occurred. Please try again.", false)
	}
	if numPlayers >= 8 {
		if err := h.DraftStore.CancelOutstandingInvites(c.Request().Context(), invite.DraftId); err != nil {
			log.Error(c.Request().Context(), "Failed to cancel outstanding invites", "error", err, "draftId", invite.DraftId)
		}
		return renderInviteTable(h, c, true, "Too many players are already in the draft. Please contect the draft owner if you think this is an error.", false)
	}

	draftId, playerId, err := h.DraftStore.AcceptInvite(c.Request().Context(), inviteId)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to accept invite", "error", err, "inviteId", inviteId)
		return renderInviteTable(h, c, true, "An error occurred. Please try again.", false)
	}
	if err := h.DraftStore.AddPlayerToDraft(c.Request().Context(), draftId, playerId); err != nil {
		log.Error(c.Request().Context(), "Failed to add player to draft", "error", err, "draftId", draftId, "playerId", playerId)
		return renderInviteTable(h, c, true, "An error occurred. Please try again.", false)
	}

	if numPlayers >= 7 {
		if err := h.DraftStore.CancelOutstandingInvites(c.Request().Context(), invite.DraftId); err != nil {
			log.Error(c.Request().Context(), "Failed to cancel outstanding invites", "error", err, "draftId", invite.DraftId)
		}
	}

	return renderInviteTable(h, c, false, "", false)
}
