package apihandler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	apimodel "server/api/model"
	"server/model/mocks"
	"testing"
	"uuid"

	"github.com/labstack/echo/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	mocklib "github.com/stretchr/testify/mock"
)

func TestGetMe_Success(t *testing.T) {
	userStore := new(mocks.MockUserStore)
	userUuid := uuid.New()
	username := "testuser"

	userStore.On("GetUsername", mocklib.Anything, userUuid).Return(username, nil)

	handler := NewHandler(nil, userStore, nil, nil, nil)
	e := echo.New()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userUuid", userUuid)

	err := handler.GetMe(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, responseRecorder(c).Code)

	var resp apimodel.UserSummary
	require.NoError(t, json.Unmarshal(responseRecorder(c).Body.Bytes(), &resp))
	assert.Equal(t, userUuid.String(), resp.Uuid)
	assert.Equal(t, username, resp.Username)
}

func TestGetMe_Unauthorized(t *testing.T) {
	handler := NewHandler(nil, nil, nil, nil, nil)
	e := echo.New()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := handler.GetMe(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, responseRecorder(c).Code)
}

func TestGetMe_StoreError(t *testing.T) {
	userStore := new(mocks.MockUserStore)
	userUuid := uuid.New()

	userStore.On("GetUsername", mocklib.Anything, userUuid).Return("", errors.New("db error"))

	handler := NewHandler(nil, userStore, nil, nil, nil)
	e := echo.New()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api/v1/users/me", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.Set("userUuid", userUuid)

	err := handler.GetMe(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, responseRecorder(c).Code)
}
