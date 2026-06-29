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
		log.Error(c.Request().Context(), "Failed to get username", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	teamIndex := team.TeamScoreIndex(h.csrfToken(c))
	teamView := team.TeamPick(" | Team Score", true, username, teamIndex)
	if err := Render(c, teamView); err != nil {
		log.Error(c.Request().Context(), "Failed to render team score page", "error", err)
		return err
	}
	return nil
}

func (h *Handler) HandleGetTeamScore(c echo.Context) error {
	teamNumber := c.FormValue("teamNumber")
	log.Debug(c.Request().Context(), "Getting score for team", "teamNumber", teamNumber)

	//Get team score
	scores, err := h.TeamStore.GetScore(c.Request().Context(), "frc"+teamNumber)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get team score", "teamNumber", teamNumber, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	// Get qualification matches
	qualificationMatches, err := h.TeamStore.GetMatchScores(c.Request().Context(), "frc"+teamNumber)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get match scores", "teamNumber", teamNumber, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	teamView := team.TeamScoreReport(teamNumber, scores, qualificationMatches)
	if err := Render(c, teamView); err != nil {
		log.Error(c.Request().Context(), "Failed to render team score report", "teamNumber", teamNumber, "error", err)
		return err
	}
	return nil
}
