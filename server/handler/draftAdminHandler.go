package handler

import (
	"database/sql"
	"fmt"
	"server/draft"
	"server/log"
	"server/model"
	"server/types"
	draftView "server/view/draft"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"
)

func renderAdminMessage(c *echo.Context, message string, success bool) error {
	return RenderOrLog(c, draftView.AdminMessage(message, success), "Failed to render admin message")
}

// HandleDraftAdminGet renders the draft admin page for the draft owner.
func (h *Handler) HandleDraftAdminGet(c *echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to serve draft admin page")

	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	draftId, err := h.parseDraftId(c)
	if err != nil {
		return BadRequest(c, "Failed to parse draft id", "Invalid draft ID", "error", err)
	}

	draftModel, err := h.Stores.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "User attempted to visit admin for invalid draft", "userUuid", userUuid, "draftId", draftId, "error", err)
		return Redirect(c, "/u/home")
	}

	if draftModel.Owner.UserUuid != userUuid {
		log.Warn(c.Request().Context(), "Non-owner attempted to access draft admin", "userUuid", userUuid, "draftId", draftId, "ownerUuid", draftModel.Owner.UserUuid)
		return Forbidden(c, "Non-owner accessed draft admin", "You do not have permission to access this page")
	}

	adminIndex := draftView.DraftAdminIndex(draftModel, h.csrfToken(c))
	draftAdmin := draftView.DraftAdmin("Draft Admin", true, username, adminIndex, types.NewPageData(draftId, draftModel.DisplayName, true))
	return RenderOrLog(c, draftAdmin, "Failed to render draft admin page", "draftId", draftId)
}

// HandleAdminEndDraft ends the draft by skipping all remaining picks (owner only).
func (h *Handler) HandleAdminEndDraft(c *echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to end draft")

	_, draftModel, ok := h.requireDraftOwnerAdmin(c)
	if !ok {
		return nil
	}
	draftId := draftModel.Id

	draftActor, err := h.Services.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "draftId", draftId, "error", err)
		return renderAdminMessage(c, "Draft not found", false)
	}

	ended := draft.EndDraft(c.Request().Context(), draftActor, draftId)
	if !ended {
		log.Warn(c.Request().Context(), "Failed to end draft", "draftId", draftId)
		return renderAdminMessage(c, "Failed to end draft", false)
	}

	if h.Services.DraftDaemon != nil {
		if err := h.Services.DraftDaemon.RemoveDraft(c.Request().Context(), draftId); err != nil {
			log.Warn(c.Request().Context(), "Failed to remove draft from daemon", "draftId", draftId, "error", err)
		}
	}

	return renderAdminMessage(c, "Draft ended and remaining picks skipped", true)
}

// HandleAdminSkipPick skips the current pick in the draft (owner only).
func (h *Handler) HandleAdminSkipPick(c *echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to skip pick")

	_, draftModel, ok := h.requireDraftOwnerAdmin(c)
	if !ok {
		return nil
	}
	draftId := draftModel.Id

	draftActor, err := h.Services.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "draftId", draftId, "error", err)
		return renderAdminMessage(c, "Draft not found", false)
	}

	skipped := draft.SkipCurrentPick(c.Request().Context(), draftActor, draftId, draftActor.GetDraftState().CurrentPick.Id)
	if !skipped {
		log.Warn(c.Request().Context(), "Failed to skip pick", "draftId", draftId)
		return renderAdminMessage(c, "Failed to skip pick", false)
	}

	return renderAdminMessage(c, "Pick skipped successfully", true)
}

// HandleAdminExtendTime extends the expiration time for the current pick.
func (h *Handler) HandleAdminExtendTime(c *echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to extend pick time")

	userUuid, draftModel, ok := h.requireDraftOwnerAdmin(c)
	if !ok {
		return nil
	}
	draftId := draftModel.Id

	durationStr := c.QueryParam("duration")
	if durationStr == "" {
		durationStr = c.FormValue("duration")
	}

	if durationStr == "" {
		log.Warn(c.Request().Context(), "Extend pick time missing duration", "userUuid", userUuid, "draftId", draftId)
		return renderAdminMessage(c, "Duration is required", false)
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Warn(c.Request().Context(), "Invalid duration format for extend pick time", "draftId", draftId, "duration", durationStr, "error", err)
		return renderAdminMessage(c, "Invalid duration format. Use format like: 30m, 1h, 2h30m", false)
	}
	if duration <= 0 {
		return renderAdminMessage(c, "Duration must be positive", false)
	}

	log.Info(c.Request().Context(), "Extending pick", "extensionTime", duration)
	draftActor, err := h.Services.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "draftId", draftId, "error", err)
		return renderAdminMessage(c, "Draft not found", false)
	}

	err = draft.ModifyCurrentPickExpirationTime(c.Request().Context(), draftActor, duration)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to extend pick time", "draftId", draftId, "duration", duration, "error", err)
		return renderAdminMessage(c, fmt.Sprintf("Failed to extend time: %s", err.Error()), false)
	}

	return renderAdminMessage(c, fmt.Sprintf("Pick time extended by %s", durationStr), true)
}

// HandleAdminMakePick allows the draft owner to make a pick on behalf of the current player.
func (h *Handler) HandleAdminMakePick(c *echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to make admin pick")

	userUuid, draftModel, ok := h.requireDraftOwnerAdmin(c)
	if !ok {
		return nil
	}
	draftId := draftModel.Id

	teamStr := c.FormValue("teamNumber")
	if teamStr == "" {
		log.Warn(c.Request().Context(), "Admin make pick missing team number", "userUuid", userUuid, "draftId", draftId)
		return renderAdminMessage(c, "Team number is required", false)
	}
	if _, err := strconv.Atoi(teamStr); err != nil {
		return renderAdminMessage(c, "Team number must be numeric", false)
	}

	tbaId := teamPrefix + teamStr

	draftActor, err := h.Services.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "draftId", draftId, "error", err)
		return renderAdminMessage(c, "Draft not found", false)
	}

	currentPick := draft.GetCurrentPick(draftActor)
	if currentPick.Id == 0 {
		log.Warn(c.Request().Context(), "Admin make pick has no current pick", "draftId", draftId)
		return renderAdminMessage(c, "No current pick found for this draft", false)
	}

	pick := model.Pick{
		Id:       currentPick.Id,
		Player:   currentPick.Player,
		Pick:     sql.NullString{String: tbaId, Valid: true},
		PickTime: sql.NullTime{Time: time.Now().UTC(), Valid: true},
	}

	err = draft.MakePick(c.Request().Context(), draftActor, pick)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to make admin pick", "draftId", draftId, "team", teamStr, "error", err)
		return renderAdminMessage(c, fmt.Sprintf("Failed to make pick: %s", err.Error()), false)
	}

	return renderAdminMessage(c, fmt.Sprintf("Successfully picked team %s", teamStr), true)
}

// HandleAdminUndoPick undoes the last pick made in the draft.
func (h *Handler) HandleAdminUndoPick(c *echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to undo pick")

	_, draftModel, ok := h.requireDraftOwnerAdmin(c)
	if !ok {
		return nil
	}
	draftId := draftModel.Id

	draftActor, err := h.Services.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "draftId", draftId, "error", err)
		return renderAdminMessage(c, "Draft not found", false)
	}

	err = draft.UndoLastPick(c.Request().Context(), draftActor)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to undo pick", "draftId", draftId, "error", err)
		return renderAdminMessage(c, fmt.Sprintf("Failed to undo pick: %s", err.Error()), false)
	}

	return renderAdminMessage(c, "Pick undone successfully", true)
}
