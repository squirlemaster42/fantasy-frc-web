package handler

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

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
			UserStore: mockUserStore,
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
			UserStore: mockUserStore,
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

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("UpdateDiscordId", c.Request().Context(), userUuid, "").Return(nil)
		mockUserStore.On("ValidateLogin", c.Request().Context(), "testuser", "oldpass").Return(true, nil)
		mockUserStore.On("UpdatePassword", c.Request().Context(), "testuser", "newpass123").Return(nil)
		mockUserStore.On("InvalidateAllUserSessionsExcept", c.Request().Context(), userUuid, "test-session").Return(nil)

		h := &Handler{
			UserStore: mockUserStore,
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
			UserStore: mockUserStore,
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

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("UpdateDiscordId", c.Request().Context(), userUuid, "").Return(nil)

		h := &Handler{
			UserStore: mockUserStore,
		}

		err := h.HandleUpdateUserProfile(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "New passwords do not match")
	})

	t.Run("invalid current password", func(t *testing.T) {
		_, c, rec := setupTestContext(t, http.MethodPost, "/userProfile", "discordId=&currentPassword=wrong&newPassword=newpass123&confirmNewPassword=newpass123", "test-session")
		userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
		c.Set("userUuid", userUuid)
		mockUserStore := mocks.NewMockUserStore(t)

		mockUserStore.On("GetUsername", c.Request().Context(), userUuid).Return("testuser", nil)
		mockUserStore.On("UpdateDiscordId", c.Request().Context(), userUuid, "").Return(nil)
		mockUserStore.On("ValidateLogin", c.Request().Context(), "testuser", "wrong").Return(false, nil)

		h := &Handler{
			UserStore: mockUserStore,
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
