package handler

import (
	"fmt"
	"server/model"
	"server/view/team"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleTeamScore(c echo.Context) error {
    teamIndex := team.TeamScoreIndex()
    team := team.TeamPick(" | Team Score", teamIndex)
    Render(c, team)
    return nil
}

func (h *Handler) HandleGetTeamScore(c echo.Context) error {
    teamNumber := c.FormValue("teamNumber")
    h.Logger.Log(fmt.Sprintf("Getting score for %s\n", teamNumber))

    //Get team score
    scores := model.GetScore(h.Database, "frc" + teamNumber)

    team := team.TeamScoreReport(teamNumber, scores)
    Render(c, team)
    return nil
}
