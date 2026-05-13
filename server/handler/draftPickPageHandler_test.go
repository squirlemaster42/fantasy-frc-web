package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	"server/model/mocks"
)

func TestServePickPage(t *testing.T) {
	t.Run("invalid draft id", func(t *testing.T) {
		_, c, _ := setupTestContext(t, http.MethodGet, "/u/draft/abc/pick", "", "test-session")
		c.SetParamNames("id")
		c.SetParamValues("abc")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.ServePickPage(c)
		assert.Error(t, err)
	})
}

func TestHandleSkipPickToggle(t *testing.T) {
	t.Run("success - mark skipping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/u/draft/42/skipPickToggle", strings.NewReader("skipping=true"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"})
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("42")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockDraftStore.On("GetDraftPlayerId", c.Request().Context(), 42, userUuid).Return(7, nil)
		mockDraftStore.On("MarkShouldSkipPick", c.Request().Context(), 7, true).Return(nil)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleSkipPickToggle(c)
		assert.NoError(t, err)
	})

	t.Run("invalid draft id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/u/draft/abc/skipPickToggle", strings.NewReader("skipping=true"))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		req.AddCookie(&http.Cookie{Name: "sessionToken", Value: "test-session"})
		rec := httptest.NewRecorder()

		e := echo.New()
		c := e.NewContext(req, rec)
		c.SetParamNames("id")
		c.SetParamValues("abc")

		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleSkipPickToggle(c)
		assert.Error(t, err)
	})
}
