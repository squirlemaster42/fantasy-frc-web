package apihandler

import (
	"database/sql"
	"errors"
	"net/http"
	"server/api"
	apimodel "server/api/model"
	"server/draft"
	"server/log"
	"server/model"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ListInvites returns all invites for the authenticated user.
func (h *Handler) ListInvites(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	invites, err := h.DraftStore.GetInvites(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to get invites", "UserUuid", userUuid, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	response := make([]apimodel.InviteResponse, 0, len(invites))
	for _, i := range invites {
		response = append(response, mapInvite(i))
	}

	return respondJSON(c, http.StatusOK, response)
}

// CreateInvite invites a user to a draft. Only the owner can invite.
func (h *Handler) CreateInvite(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	draftId, ok := parseDraftId(c)
	if !ok {
		return nil
	}

	draftModel, ok := loadDraft(ctx, h, c, draftId)
	if !ok {
		return nil
	}

	if !requireDraftOwner(c, draftModel, userUuid) {
		return nil
	}

	if draftModel.Status != model.FILLING {
		api.BadRequest(c.Response(), "Draft must be in FILLING state to invite players")
		return nil
	}

	var req apimodel.CreateInviteRequest
	if err := c.Bind(&req); err != nil {
		api.BadRequest(c.Response(), "Invalid request body")
		return nil
	}

	invitedUserUuid, err := uuid.Parse(req.UserUuid)
	if err != nil {
		api.BadRequest(c.Response(), "Invalid user_uuid")
		return nil
	}

	draftActor, err := h.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "DraftId", draftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	if err := draft.InvitePlayer(ctx, draftActor, model.DraftInvite{
		InvitingUserUuid: userUuid,
		InvitedUserUuid:  invitedUserUuid,
	}); err != nil {
		log.Error(ctx, "Failed to invite player", "DraftId", draftId, "InvitedUserUuid", invitedUserUuid, "Error", err)
		api.BadRequest(c.Response(), err.Error())
		return nil
	}

	return c.NoContent(http.StatusNoContent)
}

// AcceptInvite accepts an invite for the authenticated user.
func (h *Handler) AcceptInvite(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	inviteIdStr := c.Param("id")
	inviteId, err := strconv.Atoi(inviteIdStr)
	if err != nil {
		api.BadRequest(c.Response(), "Invalid invite ID")
		return nil
	}

	invite, err := h.DraftStore.GetInvite(ctx, inviteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			api.NotFound(c.Response(), "Invite not found")
			return nil
		}
		log.Error(ctx, "Failed to get invite", "InviteId", inviteId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	if invite.InvitedUserUuid != userUuid {
		api.Forbidden(c.Response(), "You cannot accept invites for other users")
		return nil
	}

	draftActor, err := h.DraftActorMap.GetActor(ctx, invite.DraftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "DraftId", invite.DraftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	if err := draft.AcceptInvite(ctx, draftActor, inviteId, userUuid); err != nil {
		log.Error(ctx, "Failed to accept invite", "InviteId", inviteId, "Error", err)
		api.BadRequest(c.Response(), err.Error())
		return nil
	}

	return c.NoContent(http.StatusNoContent)
}

// DeclineInvite cancels/declines an invite for the authenticated user.
func (h *Handler) DeclineInvite(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	inviteIdStr := c.Param("id")
	inviteId, err := strconv.Atoi(inviteIdStr)
	if err != nil {
		api.BadRequest(c.Response(), "Invalid invite ID")
		return nil
	}

	invite, err := h.DraftStore.GetInvite(ctx, inviteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			api.NotFound(c.Response(), "Invite not found")
			return nil
		}
		log.Error(ctx, "Failed to get invite", "InviteId", inviteId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	if invite.InvitedUserUuid != userUuid {
		api.Forbidden(c.Response(), "You cannot decline invites for other users")
		return nil
	}

	if err := h.DraftStore.DeclineInvite(ctx, inviteId); err != nil {
		log.Error(ctx, "Failed to decline invite", "InviteId", inviteId, "Error", err)
		api.BadRequest(c.Response(), err.Error())
		return nil
	}

	return c.NoContent(http.StatusNoContent)
}
