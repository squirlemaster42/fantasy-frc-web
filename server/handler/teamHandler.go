package handler

import (
	"fmt"
	"server/model"
	"server/view/team"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleTeamScore(c echo.Context) error {
    teamIndex := team.TeamScoreIndex(model.Team{})
    team := team.TeamPick(" | Team Score", teamIndex)
    Render(c, team)
    return nil
}

func (h *Handler) HandleGetTeamScore(c echo.Context) error {
    teamNumber := c.FormValue("teamNumber")
    h.Logger.Log(fmt.Sprintf("Getting score for %s\n", teamNumber))

    //Get team score
    //teamModel := model.GetTeam(h.Database, fmt.Sprintf("tba%s", teamNumber))

    teamIndex := team.TeamScoreIndex(model.Team{})
    team := team.TeamPick(" | Team Score", teamIndex)
    Render(c, team)
    return nil
}
