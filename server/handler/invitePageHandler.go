package handler

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"server/draft"
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
		log.Error(c.Request().Context(), "Failed to get username", "error", err)
		username = ""
	}

	invites, err := h.DraftStore.GetInvites(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get invites", "error", err)
		return err
	}

	inviteIndex := draftView.DraftInviteIndex(invites, hasError, errorMessage, h.csrfToken(c))
	if includeWrapper {
		inviteView := draftView.DraftInvite("Draft Invites", true, username, inviteIndex)
		if err := Render(c, inviteView); err != nil {
			log.Error(c.Request().Context(), "Failed to render invite page", "error", err)
			return err
		}
	} else {
		if err := Render(c, inviteIndex); err != nil {
			log.Error(c.Request().Context(), "Failed to render invite index", "error", err)
			return err
		}
	}

	return nil
}

func (h *Handler) HandleAcceptInvite(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	inviteIdStr := c.FormValue("inviteId")
	log.Debug(c.Request().Context(), "Got request to accept invite", "userUuid", userUuid, "inviteId", inviteIdStr)
	inviteId, err := strconv.Atoi(inviteIdStr)
	if err != nil || inviteId == 0 {
		log.Warn(c.Request().Context(), "Failed to parse invite id", "inviteId", inviteIdStr, "error", err)
		return renderInviteTable(h, c, true, "Invalid invite ID.", false)
	}

	invite, err := h.DraftStore.GetInvite(c.Request().Context(), inviteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn(c.Request().Context(), "Invite not found", "inviteId", inviteId)
			return renderInviteTable(h, c, true, "Invite not found. It may have been cancelled or expired.", false)
		}
		log.Error(c.Request().Context(), "Failed to get invite", "error", err, "inviteId", inviteId)
		return renderInviteTable(h, c, true, "An error occurred. Please try again.", false)
	}

	//Make sure that other players cannot accept someones draft
	if invite.InvitedUserUuid != userUuid {
		log.Warn(c.Request().Context(), "User attempted to accept invite for another player", "invitedUserUuid", invite.InvitedUserUuid, "userUuid", userUuid)
		return renderInviteTable(h, c, true, "You are not allowed to accept drafts for other players.", false)
	}

	log.Info(c.Request().Context(), "Accepting invite from player", "inviteId", inviteId, "userUuid", userUuid)

	// Route through the draft actor so the cached state stays in sync
	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), invite.DraftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "draftId", invite.DraftId, "error", err)
		return renderInviteTable(h, c, true, "An error occurred. Please try again.", false)
	}

	err = draft.AcceptInvite(c.Request().Context(), draftActor, inviteId, userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to accept invite", "error", err, "inviteId", inviteId)
		return renderInviteTable(h, c, true, err.Error(), false)
	}

	return renderInviteTable(h, c, false, "", false)
}

func (h *Handler) HandleDeclineInvite(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	inviteIdStr := c.FormValue("inviteId")

	log.Info(c.Request().Context(), "Got request to decline invite", "User", userUuid, "Invite Id", inviteIdStr)

	inviteId, err := strconv.Atoi(inviteIdStr)
	if err != nil || inviteId == 0 {
		log.Warn(c.Request().Context(), "Failed to parse invite id", "Invite Id", inviteIdStr, "Error", err)
		return renderInviteTable(h, c, true, "Invalid invite ID.", false)
	}

	// We need the draftId before routing through the actor to get the DraftActor
	inviteForDraftId, err := h.DraftStore.GetInvite(c.Request().Context(), inviteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return renderInviteTable(h, c, true, "Invite not found. It may have been cancelled or expired.", false)
		}
		log.Error(c.Request().Context(), "Failed to get invite", "error", err, "inviteId", inviteId)
		return renderInviteTable(h, c, true, "An error occurred. Please try again.", false)
	}

	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), inviteForDraftId.DraftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "draftId", inviteForDraftId.DraftId, "error", err)
		return renderInviteTable(h, c, true, "An error occurred. Please try again.", false)
	}

	err = draft.DeclineInvite(c.Request().Context(), draftActor, inviteId, userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to decline invite", "error", err, "inviteId", inviteId)
		return renderInviteTable(h, c, true, err.Error(), false)
	}

	invites, err := h.DraftStore.GetInvites(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get invites", "error", err)
		return renderInviteTable(h, c, true, "An error occurred. Please try again.", false)
	}
	return Render(c, draftView.DraftInviteIndex(invites, false, "", h.csrfToken(c)))
}
