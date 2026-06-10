package handler

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
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
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)

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
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
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
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
			Id:    42,
			Owner: model.User{UserUuid: userUuid},
		}, nil)
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

// Layer 2 HTML body assertions for draft profile page

func TestHandleViewDraftProfile_HTML(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/draft/42/profile", "", "test-session")
	c.SetParamNames("id")
	c.SetParamValues("42")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)
	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)

	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetDraft", c.Request().Context(), 42).Return(model.DraftModel{
		Id:          42,
		DisplayName: "Test Draft",
		Description: "A test draft",
		Status:      model.FILLING,
		Interval:    60,
		StartTime:   time.Now().Add(1 * time.Hour),
		EndTime:     time.Now().Add(72 * time.Hour),
		Owner:       model.User{UserUuid: userUuid, Username: "testuser"},
		Players: []model.DraftPlayer{
			{User: model.User{Username: "testuser"}, Pending: false},
			{User: model.User{Username: "player2"}, Pending: true},
		},
	}, nil)

	h := &Handler{
		DraftStore: mockDraftStore,
		UserStore:  mockUserStore,
	}

	err := h.HandleViewDraftProfile(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "Test Draft")
	assert.Contains(t, body, "A test draft")
	assert.Contains(t, body, string(model.FILLING))
	assert.Contains(t, body, `name="draftName"`)
	assert.Contains(t, body, `name="description"`)
	assert.Contains(t, body, `name="interval"`)
	assert.Contains(t, body, `name="startTime"`)
	assert.Contains(t, body, `name="endTime"`)
	assert.Contains(t, body, `hx-post="/u/draft/42/updateDraft"`)
	assert.Contains(t, body, "Save Changes")
	assert.Contains(t, body, "Invite Players")
	assert.Contains(t, body, "Start Draft")
	assert.Contains(t, body, "testuser")
	assert.Contains(t, body, "player2")
	assert.Contains(t, body, `<!doctype html>`)
	assert.Contains(t, body, `hx-boost="true"`)
}
