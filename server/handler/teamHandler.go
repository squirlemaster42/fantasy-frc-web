package handler

import (
	"context"
	"net/http"
	"server/cache"
	"server/log"
	"server/view/team"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleTeamScore(c echo.Context) error {
	_, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	teamIndex := team.TeamScoreIndex(h.csrfToken(c))
	teamView := team.TeamPick("Team Score", true, username, teamIndex)
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
	scores, err := h.Stores.TeamStore.GetScore(c.Request().Context(), teamPrefix+teamNumber)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get team score", "teamNumber", teamNumber, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	// Get qualification matches
	qualificationMatches, err := h.Stores.TeamStore.GetMatchScores(c.Request().Context(), teamPrefix+teamNumber)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get match scores", "teamNumber", teamNumber, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	avatarColor := avatarColorForTeam(c.Request().Context(), h.Services.AvatarStore, teamNumber)

	teamView := team.TeamScoreReport(teamNumber, scores, qualificationMatches, avatarColor)
	if err := Render(c, teamView); err != nil {
		log.Error(c.Request().Context(), "Failed to render team score report", "teamNumber", teamNumber, "error", err)
		return err
	}
	return nil
}

func avatarColorForTeam(ctx context.Context, store cache.AvatarStoreInterface, teamNumber string) string {
	if store == nil {
		return cache.DefaultAvatarColor
	}
	teamNum, err := strconv.Atoi(teamNumber)
	if err != nil {
		return cache.DefaultAvatarColor
	}
	return store.GetAvatarColor(ctx, teamNum)
}
