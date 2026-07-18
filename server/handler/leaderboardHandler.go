package handler

import (
	"net/http"
	"server/log"
	"server/model"
	"server/types"
	"server/view/leaderboard"
	"slices"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleOverallLeaderboard(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	page, err := strconv.Atoi(c.QueryParam("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage := 25

	leaderboardPage, err := h.DraftStore.GetOverallLeaderboard(c.Request().Context(), page, perPage)
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
