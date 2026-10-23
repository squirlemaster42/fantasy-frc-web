package draft

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"server/cache"
	"server/model"
	"server/model/mocks"
)

type testEventTeamCache struct {
	teams       map[string]struct{}
	lastRefresh time.Time
	refreshErr  error
	isReady     bool
}

func (t *testEventTeamCache) IsTeamAtEvent(ctx context.Context, teamId string) (bool, error) {
	if !t.isReady {
		return false, cache.ErrCacheNotReady
	}
	_, ok := t.teams[teamId]
	return ok, nil
}

func (t *testEventTeamCache) Refresh(ctx context.Context) error {
	return t.refreshErr
}

func (t *testEventTeamCache) InvalidateAndRefresh(ctx context.Context) error {
	return t.refreshErr
}

func (t *testEventTeamCache) LastRefresh() time.Time {
	return t.lastRefresh
}

func TestPickValidator_ValidatePick_NoTeam(t *testing.T) {
	validator := NewPickValidator(nil, nil, 1)

	err := validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: false, String: ""},
	})
	require.Error(t, err)
	assert.Equal(t, "no team entered", err.Error())

	err = validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: true, String: ""},
	})
	require.Error(t, err)
	assert.Equal(t, "no team entered", err.Error())
}

func TestPickValidator_ValidatePick_AlreadyPicked(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("HasBeenPicked", t.Context(), 1, "frc254").Return(true, nil).Once()

	eventCache := &testEventTeamCache{teams: map[string]struct{}{}, isReady: true}
	validator := NewPickValidator(eventCache, mockStore, 1)

	err := validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: true, String: "frc254"},
	})
	require.Error(t, err)
	assert.Equal(t, "team already picked", err.Error())
	mockStore.AssertExpectations(t)
}

func TestPickValidator_ValidatePick_NilCache(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("HasBeenPicked", t.Context(), 1, "frc254").Return(false, nil).Once()

	validator := NewPickValidator(nil, mockStore, 1)

	err := validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: true, String: "frc254"},
	})
	require.Error(t, err)
	assert.Equal(t, "pick validation cache is not available", err.Error())
	mockStore.AssertExpectations(t)
}

func TestPickValidator_ValidatePick_TeamNotAtEvent(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("HasBeenPicked", t.Context(), 1, "frc254").Return(false, nil).Once()

	eventCache := &testEventTeamCache{teams: map[string]struct{}{}, isReady: true}
	validator := NewPickValidator(eventCache, mockStore, 1)

	err := validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: true, String: "frc254"},
	})
	require.Error(t, err)
	assert.Equal(t, "team not at event", err.Error())
	mockStore.AssertExpectations(t)
}

func TestPickValidator_ValidatePick_ValidEvent(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("HasBeenPicked", t.Context(), 1, "frc254").Return(false, nil).Once()

	eventCache := &testEventTeamCache{
		teams:   map[string]struct{}{"frc254": {}},
		isReady: true,
	}
	validator := NewPickValidator(eventCache, mockStore, 1)

	err := validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: true, String: "frc254"},
	})
	require.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestPickValidator_ValidatePick_CacheNotReady(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("HasBeenPicked", t.Context(), 1, "frc254").Return(false, nil).Once()

	eventCache := &testEventTeamCache{teams: map[string]struct{}{"frc254": {}}, isReady: false}
	validator := NewPickValidator(eventCache, mockStore, 1)

	err := validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: true, String: "frc254"},
	})
	require.Error(t, err)
	assert.Equal(t, "unable to validate team eligibility", err.Error())
	mockStore.AssertExpectations(t)
}
