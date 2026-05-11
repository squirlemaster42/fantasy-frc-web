package handler

import (
	"server/assert"
	"server/log"
	"server/view/team"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleTeamScore(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Team Score")
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(c.Request().Context(), err, "Failed to get user token")

	userUuid := h.UserStore.GetUserBySessionToken(c.Request().Context(), userTok.Value)
	username := h.UserStore.GetUsername(c.Request().Context(), userUuid)

	teamIndex := team.TeamScoreIndex()
	team := team.TeamPick(" | Team Score", true, username, teamIndex)
	return Render(c, team)
}

func (h *Handler) HandleGetTeamScore(c echo.Context) error {
	teamNumber := c.FormValue("teamNumber")
	log.Info(c.Request().Context(), "Getting score for team", "Team Number", teamNumber)

	//Get team score
	scores := h.TeamStore.GetScore(c.Request().Context(), "frc"+teamNumber)

	// Get qualification matches
	qualificationMatches := h.TeamStore.GetMatchScores(c.Request().Context(), "frc"+teamNumber)

	team := team.TeamScoreReport(teamNumber, scores, qualificationMatches)
	return Render(c, team)
}
