package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"server/model"
	"server/model/mocks"
)

func TestHandleDraftAdminGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/admin", "", "test-session")
		c.SetParamNames("id")
		c.SetParamValues("42")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "test-session").Return(userUuid)
		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser")
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:    42,
			Owner: model.User{UserUuid: userUuid},
		}, nil)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleDraftAdminGet(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid draft id", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/abc/admin", "", "test-session")
		c.SetParamNames("id")
		c.SetParamValues("abc")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "test-session").Return(userUuid)
		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser")

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleDraftAdminGet(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("draft not found redirects home", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/admin", "", "test-session")
		c.SetParamNames("id")
		c.SetParamValues("42")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "test-session").Return(userUuid)
		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser")
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{}, sql.ErrNoRows)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleDraftAdminGet(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/u/home", rec.Header().Get("Location"))
	})

	t.Run("non-owner forbidden", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/admin", "", "test-session")
		c.SetParamNames("id")
		c.SetParamValues("42")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		ownerUuid := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "test-session").Return(userUuid)
		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser")
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:    42,
			Owner: model.User{UserUuid: ownerUuid},
		}, nil)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleDraftAdminGet(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestHandleAdminSkipPick(t *testing.T) {
	t.Run("invalid draft id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/u/draft/abc/admin/skip", nil)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"})
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "test-session").Return(userUuid)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleAdminSkipPick(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("draft not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/u/draft/42/admin/skip", nil)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"})
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("42")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "test-session").Return(userUuid)
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{}, sql.ErrNoRows)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleAdminSkipPick(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
