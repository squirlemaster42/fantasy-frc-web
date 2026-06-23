package apihandler

import (
	"database/sql"
	"net/http"
	"server/api"
	apimodel "server/api/model"
	"server/draft"
	"server/log"
	"server/model"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// AdminSkipPick skips the current pick. Owner only.
func (h *Handler) AdminSkipPick(c echo.Context) error {
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

	draftActor, err := h.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "DraftId", draftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	skipped := draft.SkipCurrentPick(ctx, draftActor, draftId, draftActor.GetDraftState().CurrentPick.Id)
	if !skipped {
		api.BadRequest(c.Response(), "Failed to skip pick")
		return nil
	}

	return respondJSON(c, http.StatusOK, apimodel.AdminActionResponse{Success: true, Message: "Pick skipped"})
}

// AdminExtendTime extends the current pick's expiration time. Owner only.
func (h *Handler) AdminExtendTime(c echo.Context) error {
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

	var req apimodel.AdminExtendTimeRequest
	if err := c.Bind(&req); err != nil {
		api.BadRequest(c.Response(), "Invalid request body")
		return nil
	}

	if req.Duration == 0 {
		api.BadRequest(c.Response(), "duration is required")
		return nil
	}

	draftActor, err := h.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "DraftId", draftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	if err := draft.ModifyCurrentPickExpirationTime(ctx, draftActor, req.Duration); err != nil {
		log.Warn(ctx, "Failed to extend pick time", "DraftId", draftId, "Duration", req.Duration, "Error", err)
		api.BadRequest(c.Response(), err.Error())
		return nil
	}

	return respondJSON(c, http.StatusOK, apimodel.AdminActionResponse{
		Success: true,
		Message: "Pick time extended by " + req.Duration.String(),
	})
}

// AdminMakePick makes a pick on behalf of the current player. Owner only.
func (h *Handler) AdminMakePick(c echo.Context) error {
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

	var req apimodel.AdminMakePickRequest
	if err := c.Bind(&req); err != nil {
		api.BadRequest(c.Response(), "Invalid request body")
		return nil
	}

	if req.TeamNumber <= 0 {
		api.BadRequest(c.Response(), "team_number must be greater than 0")
		return nil
	}

	draftActor, err := h.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "DraftId", draftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	currentPick := draft.GetCurrentPick(draftActor)
	if currentPick.Id == 0 {
		api.BadRequest(c.Response(), "No current pick found for this draft")
		return nil
	}

	tbaId := "frc" + strconv.Itoa(req.TeamNumber)
	pick := model.Pick{
		Id:       currentPick.Id,
		Player:   currentPick.Player,
		Pick:     sql.NullString{String: tbaId, Valid: true},
		PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
	}

	if err := draft.MakePick(ctx, draftActor, pick); err != nil {
		log.Warn(ctx, "Failed to make admin pick", "DraftId", draftId, "Team", tbaId, "Error", err)
		api.BadRequest(c.Response(), err.Error())
		return nil
	}

	return respondJSON(c, http.StatusOK, apimodel.AdminActionResponse{
		Success: true,
		Message: "Picked team " + strconv.Itoa(req.TeamNumber),
	})
}

// AdminUndoPick undoes the last pick. Owner only.
func (h *Handler) AdminUndoPick(c echo.Context) error {
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

	draftActor, err := h.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "DraftId", draftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	if err := draft.UndoLastPick(ctx, draftActor); err != nil {
		log.Warn(ctx, "Failed to undo pick", "DraftId", draftId, "Error", err)
		api.BadRequest(c.Response(), err.Error())
		return nil
	}

	return respondJSON(c, http.StatusOK, apimodel.AdminActionResponse{Success: true, Message: "Pick undone"})
}
