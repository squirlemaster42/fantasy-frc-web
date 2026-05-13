package handler

import (
	"net/http"
	"server/assert"
	"server/log"
	"server/model"
	"server/view/draft"
	"server/view/team"
	"slices"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleDraftScore(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Draft Score")

	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	draftId, err := strconv.Atoi(c.Param("id"))
	// TODO we cannot crash here
	assert.NoError(c.Request().Context(), err, "Failed to convert draft id to int")

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	// TODO we cannot crash here
	assert.NoError(c.Request().Context(), err, "Failed to get draft")

	isOwner := draftModel.Owner.UserUuid == userUuid

	userDraftScore, err := h.DraftStore.GetDraftScore(c.Request().Context(), draftId)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get draft score", "Error", err)
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
	draftView := draft.DraftScore(" | Draft Score", true, username, draftIndex, draftId, isOwner)
	return Render(c, draftView)
}

func (h *Handler) HandleDraftTeamScore(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Draft Team Score")

	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	draftId, err := strconv.Atoi(c.Param("id"))
	// TODO we shouldn't crash here
	assert.NoError(c.Request().Context(), err, "Failed to convert draft id to int")

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	// TODO we should not crash here
	assert.NoError(c.Request().Context(), err, "Failed to get draft")

	isOwner := draftModel.Owner.UserUuid == userUuid

	teamNumber := c.Param("teamNumber")
	assert.AddContext("Team Number", teamNumber)
	assert.AddContext("Draft ID", draftId)

	scores, err := h.TeamStore.GetScore(c.Request().Context(), "frc"+teamNumber)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get team score", "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	// Get qualification matches
	qualificationMatches, err := h.TeamStore.GetMatchScores(c.Request().Context(), "frc"+teamNumber)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get match scores", "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	teamScoreReport := team.TeamScoreReport(teamNumber, scores, qualificationMatches)
	draftTeamScore := draft.DraftTeamScore(" | Score Breakdown", true, username, teamScoreReport, draftId, isOwner)
	return Render(c, draftTeamScore)
}
