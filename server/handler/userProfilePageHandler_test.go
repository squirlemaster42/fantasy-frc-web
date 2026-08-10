package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"server/authentication"
	authmocks "server/authentication/mocks"
	"server/model/mocks"
)

func TestHandleViewUserProfile(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodGet, "/userProfile", "", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("GetDiscordId", c.Request().Context(), userUuid).Return("12345678901234567", nil)

		h := &Handler{
			Stores: StorageGroup{
				UserStore: mockUserStore,
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
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})
}

func TestHandleUpdateUserProfile(t *testing.T) {
	t.Run("update discord only", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile", "discordId=12345678901234567", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("UpdateDiscordId", c.Request().Context(), userUuid, "12345678901234567").Return(nil)

		h := &Handler{
			Stores: StorageGroup{UserStore: mockUserStore},
		}

		err := h.HandleUpdateUserProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Profile updated successfully")
	})

	t.Run("update password success", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile", "discordId=&currentPassword=oldpass&newPassword=newpass123&confirmNewPassword=newpass123", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockAuthService := authmocks.NewMockAuthService(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("UpdateDiscordId", c.Request().Context(), userUuid, "").Return(nil)
		mockAuthService.On("ChangePassword", c.Request().Context(), userUuid, "testuser", "oldpass", "newpass123").Return(nil)

		h := &Handler{
			Stores:   StorageGroup{UserStore: mockUserStore},
			Services: ServiceGroup{AuthService: mockAuthService},
		}

		err := h.HandleUpdateUserProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Profile updated successfully")
	})

	t.Run("missing current password", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile", "discordId=&currentPassword=&newPassword=newpass&confirmNewPassword=newpass", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("UpdateDiscordId", c.Request().Context(), userUuid, "").Return(nil)

		h := &Handler{
			Stores: StorageGroup{UserStore: mockUserStore},
		}

		err := h.HandleUpdateUserProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Current password is required")
	})

	t.Run("passwords do not match", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile", "discordId=&currentPassword=oldpass&newPassword=newpass&confirmNewPassword=different", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockAuthService := authmocks.NewMockAuthService(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("UpdateDiscordId", c.Request().Context(), userUuid, "").Return(nil)
		mockAuthService.On("ChangePassword", c.Request().Context(), userUuid, "testuser", "oldpass", "newpass").Return(&authentication.ValidationError{Message: "Passwords Do Not Match"})

		h := &Handler{
			Stores:   StorageGroup{UserStore: mockUserStore},
			Services: ServiceGroup{AuthService: mockAuthService},
		}

		err := h.HandleUpdateUserProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Passwords Do Not Match")
	})

	t.Run("invalid current password", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile", "discordId=&currentPassword=wrong&newPassword=newpass123&confirmNewPassword=newpass123", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)
		mockAuthService := authmocks.NewMockAuthService(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("UpdateDiscordId", c.Request().Context(), userUuid, "").Return(nil)
		mockAuthService.On("ChangePassword", c.Request().Context(), userUuid, "testuser", "wrong", "newpass123").Return(authentication.ErrInvalidCredentials)

		h := &Handler{
			Stores:   StorageGroup{UserStore: mockUserStore},
			Services: ServiceGroup{AuthService: mockAuthService},
		}

		err := h.HandleUpdateUserProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Current password is incorrect")
	})

	t.Run("missing cookie redirects to login", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile", "", "")

		h := &Handler{}

		err := h.HandleUpdateUserProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/login", rec.Header().Get("Location"))
	})
}
