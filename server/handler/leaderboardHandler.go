package handler

import (
	"net/http"
	"server/log"
	"server/model"
	"server/types"
	"server/view/leaderboard"
	"slices"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleOverallLeaderboard(c echo.Context) error {
	_, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage := LeaderboardPerPage()

	leaderboardPage, err := h.Stores.DraftStore.GetOverallLeaderboard(c.Request().Context(), page, perPage)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get overall leaderboard", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	for i := range leaderboardPage.Entries {
		slices.SortFunc(leaderboardPage.Entries[i].Picks, func(a, b model.Pick) int {
			return b.Score - a.Score
		})
	}

	leaderboardIndex := leaderboard.LeaderboardIndex(leaderboardPage)
	leaderboardView := leaderboard.Leaderboard("Leaderboard", true, username, leaderboardIndex, types.NewPageData(0, "", false))
	if err := Render(c, leaderboardView); err != nil {
		log.Error(c.Request().Context(), "Failed to render leaderboard page", "error", err)
		return err
	}
	return nil
}
