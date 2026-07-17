package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"server/model/mocks"
)

func TestHandleViewLanding_Unauthenticated(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/", "", "")

	mockUserStore := mocks.NewMockUserStore(t)
	h := &Handler{
		UserStore: mockUserStore,
	}

	err := h.HandleViewLanding(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Log In")
	assert.Contains(t, rec.Body.String(), "Sign Up")
}

func TestHandleViewLanding_Authenticated(t *testing.T) {
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	_, c, rec := setupTestContext(t, http.MethodGet, "/", "", "valid-token")

	mockUserStore := mocks.NewMockUserStore(t)
	mockUserStore.On("ValidateSessionToken", c.Request().Context(), "valid-token").Return(true, nil)
	mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "valid-token").Return(userUuid, nil)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("TestUser", nil)

	h := &Handler{
		UserStore: mockUserStore,
	}

	err := h.HandleViewLanding(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "TestUser")
	assert.Contains(t, rec.Body.String(), "/u/home")
	assert.NotContains(t, rec.Body.String(), "Sign Up")
	mockUserStore.AssertExpectations(t)
}

func TestHandleViewLanding_InvalidSession(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/", "", "invalid-token")

	mockUserStore := mocks.NewMockUserStore(t)
	mockUserStore.On("ValidateSessionToken", c.Request().Context(), "invalid-token").Return(false, nil)

	h := &Handler{
		UserStore: mockUserStore,
	}

	err := h.HandleViewLanding(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Log In")
	mockUserStore.AssertExpectations(t)
}
