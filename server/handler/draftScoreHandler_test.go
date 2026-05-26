package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"server/model"
	"server/model/mocks"
)

func TestHandleDraftScore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/draftScore", "", "test-session")
		c.SetParamNames("id")
		c.SetParamValues("42")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:    42,
			Owner: model.User{UserUuid: userUuid},
			Status: model.PICKING,
		}, nil)
		mockDraftStore.On("GetDraftScore", c.Request().Context(), 42).Return([]model.DraftPlayer{}, nil)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleDraftScore(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandleDraftTeamScore(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/team/254", "", "test-session")
		c.SetParamNames("id", "teamNumber")
		c.SetParamValues("42", "254")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockTeamStore := mocks.NewMockTeamStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:    42,
			Owner: model.User{UserUuid: userUuid},
			Status: model.PICKING,
		}, nil)
		mockTeamStore.On("GetScore", c.Request().Context(), "frc254").Return(map[string]int{"total": 42}, nil)
		mockTeamStore.On("GetMatchScores", c.Request().Context(), "frc254").Return([]model.MatchTeamScore{}, nil)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
			TeamStore:  mockTeamStore,
		}

		err := h.HandleDraftTeamScore(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
