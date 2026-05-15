package handler

import (
	"net/http"
	"server/log"
	"server/view/team"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleTeamScore(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	teamIndex := team.TeamScoreIndex(h.csrfToken(c))
	team := team.TeamPick(" | Team Score", true, username, teamIndex)
	return Render(c, team)
}

func (h *Handler) HandleGetTeamScore(c echo.Context) error {
	teamNumber := c.FormValue("teamNumber")
	log.Info(c.Request().Context(), "Getting score for team", "Team Number", teamNumber)

	//Get team score
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

	team := team.TeamScoreReport(teamNumber, scores, qualificationMatches)
	return Render(c, team)
}
