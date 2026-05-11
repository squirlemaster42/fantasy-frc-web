package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"server/model"
	"server/model/mocks"
)

func TestHandleViewDraftProfile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/profile", "", "test-session")
		c.SetParamNames("id")
		c.SetParamValues("42")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "test-session").Return(userUuid)
		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser")
		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:      42,
			Owner:   model.User{UserUuid: userUuid},
			Status:  model.FILLING,
		}, nil)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleViewDraftProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("invalid draft id", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/abc/profile", "", "test-session")
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

		err := h.HandleViewDraftProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid draft ID")
	})

	t.Run("draft not found redirects home", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/profile", "", "test-session")
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

		err := h.HandleViewDraftProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/u/home", rec.Header().Get("Location"))
	})
}

func TestSearchPlayers(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/u/draft/42/searchPlayers", strings.NewReader("search=john"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		req.Header.Set("Hx-Current-Url", "http://localhost/u/draft/42/profile")
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"})
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:    42,
			Owner: model.User{UserUuid: userUuid},
		}, nil)
		mockUserStore.On("GetUserBySessionToken", c.Request().Context(), "test-session").Return(userUuid)
		mockUserStore.On("SearchUsers", c.Request().Context(), "john", 42).Return([]model.User{
			{UserUuid: uuid.MustParse("660e8400-e29b-41d4-a716-446655440001"), Username: "john_doe"},
		}, nil)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.SearchPlayers(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}


