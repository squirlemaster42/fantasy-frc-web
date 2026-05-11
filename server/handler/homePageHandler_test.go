package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"server/model"
	"server/model/mocks"
)

func TestHandleViewHome(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "test-session")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("ValidateSessionToken", c.Request().Context(), "test-session").Return(true)
		mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "test-session").Return(userUuid)
		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser")
		mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{}, nil)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleViewHome(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid session redirects to login", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/u/home", nil)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "bad-session"})
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)

		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("ValidateSessionToken", c.Request().Context(), "bad-session").Return(false)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleViewHome(c)
		assert.ErrorIs(t, err, echo.ErrUnauthorized)
	})
}

func TestHandleViewCreateDraft(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/createDraft", "", "test-session")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "test-session").Return(userUuid)
		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser")

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleViewCreateDraft(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandleCreateDraftPost(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/u/createDraft", "draftName=Test+Draft&description=A+test+draft&interval=60&startTime=2026-05-11T12:00:00&endTime=2026-05-12T12:00:00", "test-session")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "test-session").Return(userUuid)
		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser")
		mockDraftStore.On("CreateDraft", c.Request().Context(), mock.AnythingOfType("*model.DraftModel")).Return(42, nil)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleCreateDraftPost(c)
		assert.NoError(t, err)
		assert.Equal(t, "/u/draft/42/profile", rec.Header().Get("HX-Redirect"))
	})
}
