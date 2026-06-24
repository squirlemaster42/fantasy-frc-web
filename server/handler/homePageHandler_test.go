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
		_, c, rec := setupTestContext(t, http.MethodPost, "/u/createDraft", "draftName=Test+Draft&description=A+test+draft&interval=60", "test-session")
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
