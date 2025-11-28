package handler

import (
	"log/slog"
	"server/assert"
	"server/model"
	"server/view/team"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleTeamScore(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Handle Team Score")
    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")

    userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userUuid)

    teamIndex := team.TeamScoreIndex(c)
    team := team.TeamPick(c, " | Team Score", true, username, teamIndex)
    return Render(c, team)
}

func (h *Handler) HandleGetTeamScore(c echo.Context) error {
    teamNumber := c.FormValue("teamNumber")
    slog.Info("Getting score for team", "Team Number", teamNumber)

    //Get team score
    scores := model.GetScore(h.Database, "frc" + teamNumber)

    team := team.TeamScoreReport(teamNumber, scores)
    return Render(c, team)
}
