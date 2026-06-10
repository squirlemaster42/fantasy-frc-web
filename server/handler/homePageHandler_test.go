package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"server/model"
	"server/model/mocks"
)

func TestHandleViewHome(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{}, nil)

		h := &Handler{
			DraftStore: mockDraftStore,
			UserStore:  mockUserStore,
		}

		err := h.HandleViewHome(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandleViewCreateDraft(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/createDraft", "", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)

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
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
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

// Layer 2 HTML body assertions for home page

func TestHandleViewHome_HTML(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)
	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)

	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{}, nil)

	h := &Handler{
		DraftStore: mockDraftStore,
		UserStore:  mockUserStore,
	}

	err := h.HandleViewHome(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "No Drafts Yet")
	assert.Contains(t, body, `href="/u/createDraft"`)
	assert.Contains(t, body, "Create New Draft")
	assert.Contains(t, body, "testuser")
	assert.Contains(t, body, `<!doctype html>`)
	assert.Contains(t, body, `hx-boost="true"`)
}

func TestHandleViewHome_HTML_WithDrafts(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/home", "", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)
	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)

	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
	mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{
		{
			Id:          1,
			DisplayName: "My Draft",
			Status:      model.FILLING,
			Owner:       model.User{UserUuid: userUuid, Username: "testuser"},
			NextPick:    model.DraftPlayer{User: model.User{Username: "nextpicker"}},
			Players: []model.DraftPlayer{
				{User: model.User{Username: "testuser"}, Pending: false},
				{User: model.User{Username: "player2"}, Pending: true},
			},
		},
	}, nil)

	h := &Handler{
		DraftStore: mockDraftStore,
		UserStore:  mockUserStore,
	}

	err := h.HandleViewHome(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "My Draft")
	assert.Contains(t, body, "Filling")
	assert.Contains(t, body, "nextpicker")
	assert.Contains(t, body, "testuser")
	assert.Contains(t, body, "player2")
	assert.Contains(t, body, `href="/u/draft/1/profile"`)
	assert.Contains(t, body, `title="Owner"`)
}

func TestHandleViewCreateDraft_HTML(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodGet, "/u/createDraft", "", "test-session")
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	c.Set("userUuid", userUuid)
	mockUserStore := mocks.NewMockUserStore(t)
	mockDraftStore := mocks.NewMockDraftStore(t)

	mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)

	h := &Handler{
		DraftStore: mockDraftStore,
		UserStore:  mockUserStore,
		MinPasswordLength: 8,
	}

	err := h.HandleViewCreateDraft(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, `hx-post="/u/createDraft"`)
	assert.Contains(t, body, `name="draftName"`)
	assert.Contains(t, body, `name="description"`)
	assert.Contains(t, body, `name="interval"`)
	assert.Contains(t, body, `name="startTime"`)
	assert.Contains(t, body, `name="endTime"`)
	assert.Contains(t, body, `name="csrf_token"`)
}
