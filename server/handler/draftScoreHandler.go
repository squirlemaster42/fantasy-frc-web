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
	assert.NoError(err, "Failed to get user token")

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userUuid)

	draftId, err := strconv.Atoi(c.Param("id"))
	assert.NoError(err, "Failed to convert draft id to int")

	draftModel, err := model.GetDraft(h.Database, draftId)
	if err != nil {
		assert.NoError(err, "Failed to get draft")
	}

	isOwner := draftModel.Owner.UserUuid == userUuid

	userDraftScore := model.GetDraftScore(h.Database, draftId)

	slices.SortFunc(userDraftScore, func(a, b model.DraftPlayer) int {
		return b.Score - a.Score
	})

	for _, draftPlayer := range userDraftScore {
		slices.SortFunc(draftPlayer.Picks, func(a, b model.Pick) int {
			return b.Score - a.Score
		})
	}

	draftIndex := draft.DraftScoreIndex(userDraftScore, draftId)
	draftView := draft.DraftScore(" | Draft Score", true, username, draftIndex, draftId, isOwner)
	return Render(c, draftView)
}

func (h *Handler) HandleDraftTeamScore(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Draft Team Score")
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userUuid)

	draftId, err := strconv.Atoi(c.Param("id"))
	assert.NoError(err, "Failed to convert draft id to int")

	draftModel, err := model.GetDraft(h.Database, draftId)
	if err != nil {
		assert.NoError(err, "Failed to get draft")
	}

	isOwner := draftModel.Owner.UserUuid == userUuid

	teamNumber := c.Param("teamNumber")
	assert.AddContext("Team Number", teamNumber)
	assert.AddContext("Draft ID", draftId)

	scores := model.GetScore(h.Database, "frc"+teamNumber)

	teamScoreReport := team.TeamScoreReport(teamNumber, scores)
	draftTeamScore := draft.DraftTeamScore(" | Score Breakdown", true, username, teamScoreReport, draftId, isOwner)
	return Render(c, draftTeamScore)
}
