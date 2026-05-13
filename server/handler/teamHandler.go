package handler

import (
	"server/log"
	"server/model"
	"server/view/team"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleTeamScore(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	username := model.GetUsername(c.Request().Context(), h.Database, userUuid)

	teamIndex := team.TeamScoreIndex(h.csrfToken(c))
	team := team.TeamPick(" | Team Score", true, username, teamIndex)
	return Render(c, team)
}

func (h *Handler) HandleGetTeamScore(c echo.Context) error {
	teamNumber := c.FormValue("teamNumber")
	log.Info(c.Request().Context(), "Getting score for team", "Team Number", teamNumber)

	//Get team score
	scores := model.GetScore(c.Request().Context(), h.Database, "frc"+teamNumber)

	// Get qualification matches
	qualificationMatches := model.GetMatchScores(c.Request().Context(), h.Database, "frc"+teamNumber)

	team := team.TeamScoreReport(teamNumber, scores, qualificationMatches)
	return Render(c, team)
}
