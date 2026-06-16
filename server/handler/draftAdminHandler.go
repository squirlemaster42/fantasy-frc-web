package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"server/draft"
	"server/log"
	"server/model"
	draftView "server/view/draft"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// HandleDraftAdminGet renders the draft admin page for the draft owner.
func (h *Handler) HandleDraftAdminGet(c echo.Context) error {
	log.Info(c.Request().Context(), "Got request to serve draft admin page")

	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "Error", err)
		username = ""
	}

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft id", "Draft Id String", c.Param("id"), "Error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "User attempted to visit admin for invalid draft", "User Uuid", userUuid, "Draft Id", draftId, "Error", err)
		return c.Redirect(http.StatusSeeOther, "/u/home")
	}

	if draftModel.Owner.UserUuid != userUuid {
		log.Warn(c.Request().Context(), "Non-owner attempted to access draft admin", "User Uuid", userUuid, "Draft Id", draftId, "Owner", draftModel.Owner.UserUuid)
		return c.String(http.StatusForbidden, "You do not have permission to access this page")
	}

	isOwner := true

	adminIndex := draftView.DraftAdminIndex(draftModel, h.csrfToken(c))
	draftAdmin := draftView.DraftAdmin(" | Draft Admin", true, username, adminIndex, draftId, isOwner)
	return Render(c, draftAdmin)
}

// HandleAdminSkipPick skips the current pick in the draft (owner only).
func (h *Handler) HandleAdminSkipPick(c echo.Context) error {
	log.Info(c.Request().Context(), "Got request to skip pick")

	userUuid := c.Get("userUuid").(uuid.UUID)

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return Render(c, draftView.AdminMessage("Invalid draft ID", false))
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		return Render(c, draftView.AdminMessage("Draft not found", false))
	}

	if draftModel.Owner.UserUuid != userUuid {
		return Render(c, draftView.AdminMessage("Permission denied", false))
	}

	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "Draft Id", draftId, "Error", err)
		return Render(c, draftView.AdminMessage("Draft not found", false))
	}

	skipped := draft.SkipCurrentPick(c.Request().Context(), draftActor, draftId, draftActor.GetDraftState().CurrentPick.Id)
	if !skipped {
		log.Warn(c.Request().Context(), "Failed to skip pick", "Draft Id", draftId)
		return Render(c, draftView.AdminMessage("Failed to skip pick", false))
	}

	return Render(c, draftView.AdminMessage("Pick skipped successfully", true))
}

// HandleAdminExtendTime extends the expiration time for the current pick.
func (h *Handler) HandleAdminExtendTime(c echo.Context) error {
	log.Info(c.Request().Context(), "Got request to extend pick time")

	userUuid := c.Get("userUuid").(uuid.UUID)

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return Render(c, draftView.AdminMessage("Invalid draft ID", false))
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		return Render(c, draftView.AdminMessage("Draft not found", false))
	}

	if draftModel.Owner.UserUuid != userUuid {
		return Render(c, draftView.AdminMessage("Permission denied", false))
	}

	durationStr := c.QueryParam("duration")
	if durationStr == "" {
		durationStr = c.FormValue("duration")
	}

	if durationStr == "" {
		return Render(c, draftView.AdminMessage("Duration is required", false))
	}

	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return Render(c, draftView.AdminMessage("Invalid duration format. Use format like: 30m, 1h, 2h30m", false))
	}
	if duration <= 0 {
		return Render(c, draftView.AdminMessage("Duration must be positive", false))
	}

	log.Info(c.Request().Context(), "Extending pick", "Extension time", duration)
	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "Draft Id", draftId, "Error", err)
		return Render(c, draftView.AdminMessage("Draft not found", false))
	}

	err = draft.ModifyCurrentPickExpirationTime(c.Request().Context(), draftActor, duration)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to extend pick time", "Draft Id", draftId, "Duration", duration, "Error", err)
		return Render(c, draftView.AdminMessage(fmt.Sprintf("Failed to extend time: %s", err.Error()), false))
	}

	return Render(c, draftView.AdminMessage(fmt.Sprintf("Pick time extended by %s", durationStr), true))
}

// HandleAdminMakePick allows the draft owner to make a pick on behalf of the current player.
func (h *Handler) HandleAdminMakePick(c echo.Context) error {
	log.Info(c.Request().Context(), "Got request to make admin pick")

	userUuid := c.Get("userUuid").(uuid.UUID)

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return Render(c, draftView.AdminMessage("Invalid draft ID", false))
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		return Render(c, draftView.AdminMessage("Draft not found", false))
	}

	if draftModel.Owner.UserUuid != userUuid {
		return Render(c, draftView.AdminMessage("Permission denied", false))
	}

	teamStr := c.FormValue("teamNumber")
	if teamStr == "" {
		return Render(c, draftView.AdminMessage("Team number is required", false))
	}
	if _, err := strconv.Atoi(teamStr); err != nil {
		return Render(c, draftView.AdminMessage("Team number must be numeric", false))
	}

	tbaId := "frc" + teamStr

	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "Draft Id", draftId, "Error", err)
		return Render(c, draftView.AdminMessage("Draft not found", false))
	}

	currentPick := draft.GetCurrentPick(draftActor)
	if currentPick.Id == 0 {
		return Render(c, draftView.AdminMessage("No current pick found for this draft", false))
	}

	pick := model.Pick{
		Id:       currentPick.Id,
		Player:   currentPick.Player,
		Pick:     sql.NullString{String: tbaId, Valid: true},
		PickTime: sql.NullTime{Time: time.Now(), Valid: true},
	}

	err = draft.MakePick(c.Request().Context(), draftActor, pick)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to make admin pick", "Draft Id", draftId, "Team", teamStr, "Error", err)
		return Render(c, draftView.AdminMessage(fmt.Sprintf("Failed to make pick: %s", err.Error()), false))
	}

	return Render(c, draftView.AdminMessage(fmt.Sprintf("Successfully picked team %s", teamStr), true))
}

// HandleAdminUndoPick undoes the last pick made in the draft.
func (h *Handler) HandleAdminUndoPick(c echo.Context) error {
	log.Info(c.Request().Context(), "Got request to undo pick")

	userUuid := c.Get("userUuid").(uuid.UUID)

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return Render(c, draftView.AdminMessage("Invalid draft ID", false))
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		return Render(c, draftView.AdminMessage("Draft not found", false))
	}

	if draftModel.Owner.UserUuid != userUuid {
		return Render(c, draftView.AdminMessage("Permission denied", false))
	}

	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "Draft Id", draftId, "Error", err)
		return Render(c, draftView.AdminMessage("Draft not found", false))
	}

	err = draft.UndoLastPick(c.Request().Context(), draftActor)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to undo pick", "Draft Id", draftId, "Error", err)
		return Render(c, draftView.AdminMessage(fmt.Sprintf("Failed to undo pick: %s", err.Error()), false))
	}

	return Render(c, draftView.AdminMessage("Pick undone successfully", true))
}
