package handler

import (
	"server/model"
	"server/types"
	"server/view/leaderboard"
	"slices"
	"strconv"

	"github.com/labstack/echo/v5"
)

func (h *Handler) HandleOverallLeaderboard(c *echo.Context) error {
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
		return InternalServerError(c, "Failed to get overall leaderboard", err)
	}

	for i := range leaderboardPage.Entries {
		slices.SortFunc(leaderboardPage.Entries[i].Picks, func(a, b model.Pick) int {
			return b.Score - a.Score
		})
	}

	avatarColors := collectLeaderboardAvatarColors(c.Request().Context(), h.Services.AvatarStore, leaderboardPage.Entries)

	leaderboardIndex := leaderboard.LeaderboardIndex(leaderboardPage, avatarColors)
	leaderboardView := leaderboard.Leaderboard("Leaderboard", true, username, leaderboardIndex, types.NewPageData(0, "", false))
	return RenderOrLog(c, leaderboardView, "Failed to render leaderboard page")
}
