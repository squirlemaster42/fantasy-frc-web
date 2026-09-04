package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"server/model"
	"server/model/mocks"
)

func TestHandleOverallLeaderboard(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	t.Run("success with sorted picks", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/leaderboard", "", "test-session")
		c.Set("userUuid", userUuid)

		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)

		leaderboardPage := model.LeaderboardPage{
			Entries: []model.LeaderboardEntry{
				{
					User:      model.User{Username: "player1"},
					Score:     42,
					DraftId:   1,
					DraftName: "Test Draft",
					Picks: []model.Pick{
						{Pick: sql.NullString{String: "frc245", Valid: true}, Score: 5},
						{Pick: sql.NullString{String: "frc254", Valid: true}, Score: 10},
					},
				},
			},
			CurrentPage: 1,
			TotalPages:  1,
			PerPage:     25,
			Total:       1,
		}

		mockDraftStore.On("GetOverallLeaderboard", c.Request().Context(), 1, 25).Return(leaderboardPage, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
			Services: ServiceGroup{
				AvatarStore: &mockAvatarStore{},
			},
		}

		err := h.HandleOverallLeaderboard(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)

		html := rec.Body.String()
		assert.Contains(t, html, "player1")
		assert.Contains(t, html, "Test Draft")
		assert.Contains(t, html, "254")
		assert.Contains(t, html, "245")

		// Picks should be sorted by score descending.
		idx254 := strings.Index(html, ">254<")
		idx245 := strings.Index(html, ">245<")
		assert.Less(t, idx254, idx245)
	})

	t.Run("defaults invalid page to 1", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/leaderboard?page=abc", "", "test-session")
		c.Set("userUuid", userUuid)

		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)

		leaderboardPage := model.LeaderboardPage{
			Entries:     []model.LeaderboardEntry{},
			CurrentPage: 1,
			TotalPages:  1,
			PerPage:     25,
			Total:       0,
		}

		mockDraftStore.On("GetOverallLeaderboard", c.Request().Context(), 1, 25).Return(leaderboardPage, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
			Services: ServiceGroup{
				AvatarStore: &mockAvatarStore{},
			},
		}

		err := h.HandleOverallLeaderboard(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("defaults negative page to 1", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/leaderboard?page=-5", "", "test-session")
		c.Set("userUuid", userUuid)

		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)

		leaderboardPage := model.LeaderboardPage{
			Entries:     []model.LeaderboardEntry{},
			CurrentPage: 1,
			TotalPages:  1,
			PerPage:     25,
			Total:       0,
		}

		mockDraftStore.On("GetOverallLeaderboard", c.Request().Context(), 1, 25).Return(leaderboardPage, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
			Services: ServiceGroup{
				AvatarStore: &mockAvatarStore{},
			},
		}

		err := h.HandleOverallLeaderboard(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("store error returns 500", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/leaderboard", "", "test-session")
		c.Set("userUuid", userUuid)

		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("GetOverallLeaderboard", c.Request().Context(), 1, 25).Return(model.LeaderboardPage{}, errors.New("database error"))

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
			Services: ServiceGroup{
				AvatarStore: &mockAvatarStore{},
			},
		}

		err := h.HandleOverallLeaderboard(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "An error occurred")
	})

	t.Run("unauthenticated user is redirected", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/leaderboard", "", "")

		h := &Handler{}

		err := h.HandleOverallLeaderboard(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})
}
