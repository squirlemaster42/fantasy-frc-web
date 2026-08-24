package draft

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"server/model"
	"server/model/mocks"
)

func TestPickValidator_ValidatePick_NoTeam(t *testing.T) {
	validator := NewPickValidator(nil, nil, 1)

	err := validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: false, String: ""},
	})
	assert.Error(t, err)
	assert.Equal(t, "no team entered", err.Error())

	err = validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: true, String: ""},
	})
	assert.Error(t, err)
	assert.Equal(t, "no team entered", err.Error())
}

func TestPickValidator_ValidatePick_AlreadyPicked(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("HasBeenPicked", t.Context(), 1, "frc254").Return(true, nil).Once()

	handler := &testTBAHandler{events: map[string][]string{}}
	validator := NewPickValidator(handler, mockStore, 1)

	err := validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: true, String: "frc254"},
	})
	assert.Error(t, err)
	assert.Equal(t, "team already picked", err.Error())
	mockStore.AssertExpectations(t)
}

func TestPickValidator_ValidatePick_TBAError(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("HasBeenPicked", t.Context(), 1, "frc254").Return(false, nil).Once()

	handler := &testTBAHandler{
		events: map[string][]string{},
		err:    errors.New("tba unavailable"),
	}
	validator := NewPickValidator(handler, mockStore, 1)

	err := validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: true, String: "frc254"},
	})
	assert.Error(t, err)
	assert.Equal(t, "tba unavailable", err.Error())
	mockStore.AssertExpectations(t)
}

func TestPickValidator_ValidatePick_TeamNotAtEvent(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("HasBeenPicked", t.Context(), 1, "frc254").Return(false, nil).Once()

	handler := &testTBAHandler{
		events: map[string][]string{
			"frc254": {"2026someother"},
		},
	}
	validator := NewPickValidator(handler, mockStore, 1)

	err := validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: true, String: "frc254"},
	})
	assert.Error(t, err)
	assert.Equal(t, "team not at event", err.Error())
	mockStore.AssertExpectations(t)
}

func TestPickValidator_ValidatePick_ValidEvent(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	mockStore.On("HasBeenPicked", t.Context(), 1, "frc254").Return(false, nil).Once()

	handler := &testTBAHandler{
		events: map[string][]string{
			"frc254": {"2026arc"},
		},
	}
	validator := NewPickValidator(handler, mockStore, 1)

	err := validator.ValidatePick(t.Context(), model.Pick{
		Pick: sql.NullString{Valid: true, String: "frc254"},
	})
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}
