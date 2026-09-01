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
		mockDraftStore.On("SearchDrafts", c.Request().Context(), model.DraftSearchQuery{
			UserUuid:        userUuid,
			DraftNameSearch: "",
			PageSize:        20,
			PageNum:         0,
		}).Return([]model.DraftModel{}, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleViewHome(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("success with search", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/home?search=test+draft", "", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("SearchDrafts", c.Request().Context(), model.DraftSearchQuery{
			UserUuid:        userUuid,
			DraftNameSearch: "test draft",
			PageSize:        20,
			PageNum:         0,
		}).Return([]model.DraftModel{}, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleViewHome(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestHandleViewDraftList(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draftList?page=1", "", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("SearchDrafts", c.Request().Context(), model.DraftSearchQuery{
			UserUuid:        userUuid,
			DraftNameSearch: "",
			PageSize:        20,
			PageNum:         1,
		}).Return([]model.DraftModel{}, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleViewDraftList(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("defaults page to zero when absent", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draftList?search=test", "", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("SearchDrafts", c.Request().Context(), model.DraftSearchQuery{
			UserUuid:        userUuid,
			DraftNameSearch: "test",
			PageSize:        20,
			PageNum:         0,
		}).Return([]model.DraftModel{}, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleViewDraftList(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("passes search term to store", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/u/draftList?page=2&search=championship", "", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("SearchDrafts", c.Request().Context(), model.DraftSearchQuery{
			UserUuid:        userUuid,
			DraftNameSearch: "championship",
			PageSize:        20,
			PageNum:         2,
		}).Return([]model.DraftModel{}, nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleViewDraftList(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("rejects negative page", func(t *testing.T) {
		e, c, rec := setupTestContext(t, http.MethodGet, "/u/draftList?page=-1", "", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)

		h := &Handler{
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleViewDraftList(c)
		if assert.Error(t, err) {
			e.HTTPErrorHandler(err, c)
		}
		assert.Equal(t, http.StatusBadRequest, rec.Code)
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
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
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
			Stores: StorageGroup{
				DraftStore: mockDraftStore,
				UserStore:  mockUserStore,
			},
		}

		err := h.HandleCreateDraftPost(c)
		assert.NoError(t, err)
		assert.Equal(t, "/u/draft/42/profile", rec.Header().Get("HX-Redirect"))
	})
}
