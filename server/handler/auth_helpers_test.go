package handler

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"server/model/mocks"
)

func TestRequireUser_Success(t *testing.T) {
	_, c, _ := setupTestContext(t, http.MethodGet, "/home", "", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockUserStore := mocks.NewMockUserStore(t)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)

	h := &Handler{Stores: StorageGroup{UserStore: mockUserStore}}

	returnedUuid, username, err := h.requireUser(c)
	assert.NoError(t, err)
	assert.Equal(t, userUuid, returnedUuid)
	assert.Equal(t, "testuser", username)
}

func TestRequireUser_MissingContextRedirectsToLogin(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/home", "", "test-session")

	h := &Handler{}

	_, _, err := h.requireUser(c)
	assert.Error(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestRequireUser_WrongTypeRedirectsToLogin(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/home", "", "test-session")
	c.Set("userUuid", "not-a-uuid")

	h := &Handler{}

	_, _, err := h.requireUser(c)
	assert.Error(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}

func TestRequireUser_GetUsernameError(t *testing.T) {
	_, c, _ := setupTestContext(t, http.MethodGet, "/home", "", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	mockUserStore := mocks.NewMockUserStore(t)
	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("", errors.New("db error"))

	h := &Handler{Stores: StorageGroup{UserStore: mockUserStore}}

	_, _, err := h.requireUser(c)
	assert.Error(t, err)
}

func TestRequireUserUuid_Success(t *testing.T) {
	_, c, _ := setupTestContext(t, http.MethodGet, "/home", "", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)

	h := &Handler{}

	returnedUuid, err := h.requireUserUuid(c)
	assert.NoError(t, err)
	assert.Equal(t, userUuid, returnedUuid)
}

func TestRequireUserUuid_MissingContextRedirectsToLogin(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/home", "", "test-session")

	h := &Handler{}

	_, err := h.requireUserUuid(c)
	assert.Error(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Location"))
}
