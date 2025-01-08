package handler

import (
	"fmt"
	"server/assert"
	"server/model"
	"server/view/team"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleTeamScore(c echo.Context) error {
    userTok, err := c.Cookie("sessionToken")
    //TODO We should have already checked that the user has a token
    //here since they should not be able to access the page otherwise
    //There might be some sort of weird thing here where the middleware
    //validates the session token is good and then it expires a second later
    assert.NoErrorCF(err, "Failed to get user token")

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userId)

    teamIndex := team.TeamScoreIndex()
    team := team.TeamPick(" | Team Score", true, username, teamIndex)
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
