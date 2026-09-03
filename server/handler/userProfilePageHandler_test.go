package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"server/authentication"
	authmocks "server/authentication/mocks"
	"server/model"
	"server/model/mocks"
)

func TestHandleViewUserProfile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/userProfile", "", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("GetDiscordId", c.Request().Context(), userUuid).Return("12345678901234567", nil)
		mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{}, nil)
		mockDraftStore.On("GetUserDraftNotificationPreferences", c.Request().Context(), userUuid).Return(map[int]model.DraftNotificationPreference{}, nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore:  mockUserStore,
				DraftStore: mockDraftStore,
			},
		}

		err := h.HandleViewUserProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("missing cookie redirects to login", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/userProfile", "", "")

		h := &Handler{}

		err := h.HandleViewUserProfile(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})
}

func TestHandleUpdateUserProfile(t *testing.T) {
	t.Run("update discord id success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile", "discordId=12345678901234567", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("UpdateDiscordId", c.Request().Context(), userUuid, "12345678901234567").Return(nil)
		mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{}, nil)
		mockDraftStore.On("GetUserDraftNotificationPreferences", c.Request().Context(), userUuid).Return(map[int]model.DraftNotificationPreference{}, nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore:  mockUserStore,
				DraftStore: mockDraftStore,
			},
		}

		err := h.HandleUpdateUserProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Discord ID updated successfully")
	})

	t.Run("invalid discord id shows error", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile", "discordId=not-a-number", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{}, nil)
		mockDraftStore.On("GetUserDraftNotificationPreferences", c.Request().Context(), userUuid).Return(map[int]model.DraftNotificationPreference{}, nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore:  mockUserStore,
				DraftStore: mockDraftStore,
			},
		}

		err := h.HandleUpdateUserProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Discord ID must be a numeric snowflake")
	})

	t.Run("missing cookie redirects to login", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile", "", "")

		h := &Handler{}

		err := h.HandleUpdateUserProfile(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})
}

func TestHandleUpdateUserPassword(t *testing.T) {
	t.Run("update password success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile/password", "currentPassword=oldpass&newPassword=newpass123&confirmNewPassword=newpass123", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockAuthService := authmocks.NewMockAuthService(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("GetDiscordId", c.Request().Context(), userUuid).Return("", nil)
		mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{}, nil)
		mockDraftStore.On("GetUserDraftNotificationPreferences", c.Request().Context(), userUuid).Return(map[int]model.DraftNotificationPreference{}, nil)
		mockAuthService.On("ChangePassword", c.Request().Context(), userUuid, "testuser", "oldpass", "newpass123").Return(nil)

		h := &Handler{
			Stores:   StorageGroup{UserStore: mockUserStore, DraftStore: mockDraftStore},
			Services: ServiceGroup{AuthService: mockAuthService},
		}

		err := h.HandleUpdateUserPassword(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Password updated successfully")
	})

	t.Run("passwords do not match", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile/password", "currentPassword=oldpass&newPassword=newpass&confirmNewPassword=different", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("GetDiscordId", c.Request().Context(), userUuid).Return("", nil)
		mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{}, nil)
		mockDraftStore.On("GetUserDraftNotificationPreferences", c.Request().Context(), userUuid).Return(map[int]model.DraftNotificationPreference{}, nil)

		h := &Handler{
			Stores: StorageGroup{UserStore: mockUserStore, DraftStore: mockDraftStore},
		}

		err := h.HandleUpdateUserPassword(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Passwords do not match")
	})

	t.Run("missing current password", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile/password", "currentPassword=&newPassword=newpass&confirmNewPassword=newpass", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("GetDiscordId", c.Request().Context(), userUuid).Return("", nil)
		mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{}, nil)
		mockDraftStore.On("GetUserDraftNotificationPreferences", c.Request().Context(), userUuid).Return(map[int]model.DraftNotificationPreference{}, nil)

		h := &Handler{
			Stores: StorageGroup{UserStore: mockUserStore, DraftStore: mockDraftStore},
		}

		err := h.HandleUpdateUserPassword(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Current password is required")
	})

	t.Run("invalid current password", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile/password", "currentPassword=wrong&newPassword=newpass123&confirmNewPassword=newpass123", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockAuthService := authmocks.NewMockAuthService(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("GetDiscordId", c.Request().Context(), userUuid).Return("", nil)
		mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{}, nil)
		mockDraftStore.On("GetUserDraftNotificationPreferences", c.Request().Context(), userUuid).Return(map[int]model.DraftNotificationPreference{}, nil)
		mockAuthService.On("ChangePassword", c.Request().Context(), userUuid, "testuser", "wrong", "newpass123").Return(authentication.ErrInvalidCredentials)

		h := &Handler{
			Stores:   StorageGroup{UserStore: mockUserStore, DraftStore: mockDraftStore},
			Services: ServiceGroup{AuthService: mockAuthService},
		}

		err := h.HandleUpdateUserPassword(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Current password is incorrect")
	})

	t.Run("missing cookie redirects to login", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile/password", "", "")

		h := &Handler{}

		err := h.HandleUpdateUserPassword(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})
}

func TestHandleUpdateUserNotificationPreferences(t *testing.T) {
	t.Run("update preferences success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile/notifications", "draft_1_upcomingMatch=on&draft_1_pickTurn=on", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		drafts := []model.DraftModel{
			{Id: 1, DisplayName: "Test Draft", Status: model.FILLING},
		}

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("GetDiscordId", c.Request().Context(), userUuid).Return("12345678901234567", nil)
		mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return(drafts, nil)
		mockDraftStore.On("GetUserDraftNotificationPreferences", c.Request().Context(), userUuid).Return(map[int]model.DraftNotificationPreference{}, nil)
		mockDraftStore.On("UpdateUserDraftNotificationPreference", c.Request().Context(), userUuid, 1, true, true).Return(nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore:  mockUserStore,
				DraftStore: mockDraftStore,
			},
		}

		err := h.HandleUpdateUserNotificationPreferences(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Notification preferences updated")
	})

	t.Run("allows saving preferences without discord id", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile/notifications", "", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockDraftStore := mocks.NewMockDraftStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("GetDiscordId", c.Request().Context(), userUuid).Return("", nil)
		mockDraftStore.On("GetDraftsForUser", c.Request().Context(), userUuid).Return([]model.DraftModel{}, nil)
		mockDraftStore.On("GetUserDraftNotificationPreferences", c.Request().Context(), userUuid).Return(map[int]model.DraftNotificationPreference{}, nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore:  mockUserStore,
				DraftStore: mockDraftStore,
			},
		}

		err := h.HandleUpdateUserNotificationPreferences(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Notification preferences updated")
	})

	t.Run("missing cookie redirects to login", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile/notifications", "", "")

		h := &Handler{}

		err := h.HandleUpdateUserNotificationPreferences(c)
		assert.Error(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})
}
