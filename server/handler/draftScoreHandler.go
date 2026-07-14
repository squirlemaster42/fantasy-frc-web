package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"server/assert"
	"server/log"
	"server/model"
	"server/types"
	"server/view/draft"
	"server/view/team"
	"slices"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleDraftScore(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to convert draft id to int", "draftIdString", c.Param("id"), "error", err)
		return c.String(http.StatusBadRequest, "Draft id was not an int")
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn(c.Request().Context(), "Draft not found", "draftId", draftId)
			return c.String(http.StatusNotFound, fmt.Sprintf("Failed to load draft id %d", draftId))
		}
		log.Error(c.Request().Context(), "Failed to get draft by id", "draftId", draftId, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	isOwner := draftModel.Owner.UserUuid == userUuid

	userDraftScore, err := h.DraftStore.GetDraftScore(c.Request().Context(), draftId)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get draft score", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	slices.SortFunc(userDraftScore, func(a, b model.DraftPlayer) int {
		return b.Score - a.Score
	})

	for _, draftPlayer := range userDraftScore {
		slices.SortFunc(draftPlayer.Picks, func(a, b model.Pick) int {
			return b.Score - a.Score
		})
	}

	draftIndex := draft.DraftScoreIndex(userDraftScore, draftId, draftModel.Status)
	draftView := draft.DraftScore("Draft Scores", true, username, draftIndex, types.NewPageData(draftId, draftModel.DisplayName, isOwner))
	if err := Render(c, draftView); err != nil {
		log.Error(c.Request().Context(), "Failed to render draft score page", "draftId", draftId, "error", err)
		return err
	}
	return nil
}

func (h *Handler) HandleDraftTeamScore(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Draft Team Score")

	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to convert draft id to int", "draftIdString", c.Param("id"), "error", err)
		return c.String(http.StatusBadRequest, "Draft id was not an int")
	}

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn(c.Request().Context(), "Draft not found", "draftId", draftId)
			return c.String(http.StatusNotFound, fmt.Sprintf("Failed to load draft id %d", draftId))
		}
		log.Error(c.Request().Context(), "Failed to get draft by id", "draftId", draftId, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	isOwner := draftModel.Owner.UserUuid == userUuid

	teamNumber := c.Param("teamNumber")
	assert.AddContext("teamNumber", teamNumber)
	assert.AddContext("draftId", draftId)

	scores, err := h.TeamStore.GetScore(c.Request().Context(), "frc"+teamNumber)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get team score", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	// Get qualification matches
	qualificationMatches, err := h.TeamStore.GetMatchScores(c.Request().Context(), "frc"+teamNumber)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get match scores", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	teamScoreReport := team.TeamScoreReport(teamNumber, scores, qualificationMatches)
	draftTeamScore := draft.DraftTeamScore(" | Score Breakdown", true, username, teamScoreReport, types.NewPageData(draftId, draftModel.DisplayName, isOwner))
	if err := Render(c, draftTeamScore); err != nil {
		log.Error(c.Request().Context(), "Failed to render draft team score page", "draftId", draftId, "teamNumber", teamNumber, "error", err)
		return err
	}
	return nil
}
