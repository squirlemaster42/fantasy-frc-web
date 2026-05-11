package handler

import (
	"server/assert"
	"server/model"
	"server/view/draft"
	"server/view/team"
	"slices"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleDraftScore(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Draft Score")
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(c.Request().Context(), err, "Failed to get user token")

	userUuid := h.UserStore.GetUserBySessionToken(c.Request().Context(), userTok.Value)
	username := h.UserStore.GetUsername(c.Request().Context(), userUuid)

	draftId, err := strconv.Atoi(c.Param("id"))
	// TODO we cannot crash here
	assert.NoError(c.Request().Context(), err, "Failed to convert draft id to int")

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	// TODO we cannot crash here
	assert.NoError(c.Request().Context(), err, "Failed to get draft")

	isOwner := draftModel.Owner.UserUuid == userUuid

	userDraftScore := h.DraftStore.GetDraftScore(c.Request().Context(), draftId)

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
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(c.Request().Context(), err, "Failed to get user token")

	userUuid := h.UserStore.GetUserBySessionToken(c.Request().Context(), userTok.Value)
	username := h.UserStore.GetUsername(c.Request().Context(), userUuid)

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

	scores := h.TeamStore.GetScore(c.Request().Context(), "frc"+teamNumber)

	// Get qualification matches
	qualificationMatches := h.TeamStore.GetMatchScores(c.Request().Context(), "frc"+teamNumber)

	teamScoreReport := team.TeamScoreReport(teamNumber, scores, qualificationMatches)
	draftTeamScore := draft.DraftTeamScore(" | Score Breakdown", true, username, teamScoreReport, draftId, isOwner)
	return Render(c, draftTeamScore)
}
