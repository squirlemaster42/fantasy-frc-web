package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"server/model"
	"server/model/mocks"
)

func TestHandleTeamScore(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/team/score", "", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)
	mockUserStore := mocks.NewMockUserStore(t)

	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)

	h := &Handler{
		UserStore: mockUserStore,
	}

	err := h.HandleTeamScore(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleGetTeamScore(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/team/score", "teamNumber=254", "")

	mockTeamStore := mocks.NewMockTeamStore(t)

	mockTeamStore.On("GetScore", c.Request().Context(), "frc254").Return(map[string]int{"total": 42}, nil)
	mockTeamStore.On("GetMatchScores", c.Request().Context(), "frc254").Return([]model.MatchTeamScore{}, nil)

	h := &Handler{
		TeamStore: mockTeamStore,
	}

	err := h.HandleGetTeamScore(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}
