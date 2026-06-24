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

func renderAdminMessage(c echo.Context, message string, success bool) error {
	if err := Render(c, draftView.AdminMessage(message, success)); err != nil {
		log.Error(c.Request().Context(), "Failed to render admin message", "error", err)
		return err
	}
	return nil
}

func (h *Handler) HandleDraftAdminGet(c echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to serve draft admin page")

	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "error", err)
		username = ""
	}

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft id", "draftIdString", c.Param("id"), "error", err)
		return c.String(http.StatusBadRequest, "Invalid draft ID")
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "User attempted to visit admin for invalid draft", "userUuid", userUuid, "draftId", draftId, "error", err)
		return c.Redirect(http.StatusSeeOther, "/u/home")
	}

	if draftModel.Owner.UserUuid != userUuid {
		log.Warn(c.Request().Context(), "Non-owner attempted to access draft admin", "userUuid", userUuid, "draftId", draftId, "ownerUuid", draftModel.Owner.UserUuid)
		return c.String(http.StatusForbidden, "You do not have permission to access this page")
	}

	isOwner := true

	adminIndex := draftView.DraftAdminIndex(draftModel, h.csrfToken(c))
	draftAdmin := draftView.DraftAdmin(" | Draft Admin", true, username, adminIndex, draftId, isOwner)
	if err := Render(c, draftAdmin); err != nil {
		log.Error(c.Request().Context(), "Failed to render draft admin page", "draftId", draftId, "error", err)
		return err
	}
	return nil
}

func (h *Handler) HandleAdminSkipPick(c echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to skip pick")

	userUuid := c.Get("userUuid").(uuid.UUID)

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Invalid draft id", "draftIdString", c.Param("id"), "error", err)
		return renderAdminMessage(c, "Invalid draft ID", false)
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Draft not found for skip pick", "userUuid", userUuid, "draftId", draftId, "error", err)
		return renderAdminMessage(c, "Draft not found", false)
	}

	if draftModel.Owner.UserUuid != userUuid {
		log.Warn(c.Request().Context(), "Non-owner attempted to skip pick", "userUuid", userUuid, "draftId", draftId, "ownerUuid", draftModel.Owner.UserUuid)
		return renderAdminMessage(c, "Permission denied", false)
	}

	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
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

func (h *Handler) HandleAdminExtendTime(c echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to extend pick time")

	userUuid := c.Get("userUuid").(uuid.UUID)

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Invalid draft id", "draftIdString", c.Param("id"), "error", err)
		return renderAdminMessage(c, "Invalid draft ID", false)
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Draft not found for extend time", "userUuid", userUuid, "draftId", draftId, "error", err)
		return renderAdminMessage(c, "Draft not found", false)
	}

	if draftModel.Owner.UserUuid != userUuid {
		log.Warn(c.Request().Context(), "Non-owner attempted to extend pick time", "userUuid", userUuid, "draftId", draftId, "ownerUuid", draftModel.Owner.UserUuid)
		return renderAdminMessage(c, "Permission denied", false)
	}

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

	log.Info(c.Request().Context(), "Extending pick", "extensionTime", duration)
	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
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

func (h *Handler) HandleAdminMakePick(c echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to make admin pick")

	userUuid := c.Get("userUuid").(uuid.UUID)

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Invalid draft id", "draftIdString", c.Param("id"), "error", err)
		return renderAdminMessage(c, "Invalid draft ID", false)
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Draft not found for admin make pick", "userUuid", userUuid, "draftId", draftId, "error", err)
		return renderAdminMessage(c, "Draft not found", false)
	}

	if draftModel.Owner.UserUuid != userUuid {
		log.Warn(c.Request().Context(), "Non-owner attempted admin make pick", "userUuid", userUuid, "draftId", draftId, "ownerUuid", draftModel.Owner.UserUuid)
		return renderAdminMessage(c, "Permission denied", false)
	}

	teamStr := c.FormValue("teamNumber")
	if teamStr == "" {
		log.Warn(c.Request().Context(), "Admin make pick missing team number", "userUuid", userUuid, "draftId", draftId)
		return renderAdminMessage(c, "Team number is required", false)
	}

	tbaId := "frc" + teamStr

	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
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

func (h *Handler) HandleAdminUndoPick(c echo.Context) error {
	log.Debug(c.Request().Context(), "Got request to undo pick")

	userUuid := c.Get("userUuid").(uuid.UUID)

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Invalid draft id", "draftIdString", c.Param("id"), "error", err)
		return renderAdminMessage(c, "Invalid draft ID", false)
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Draft not found for undo pick", "userUuid", userUuid, "draftId", draftId, "error", err)
		return renderAdminMessage(c, "Draft not found", false)
	}

	if draftModel.Owner.UserUuid != userUuid {
		log.Warn(c.Request().Context(), "Non-owner attempted to undo pick", "userUuid", userUuid, "draftId", draftId, "ownerUuid", draftModel.Owner.UserUuid)
		return renderAdminMessage(c, "Permission denied", false)
	}

	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
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
