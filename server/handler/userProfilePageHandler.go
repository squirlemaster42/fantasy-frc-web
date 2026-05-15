package handler

import (
	"fmt"
	"net/http"
	"server/log"
	"server/view/userProfile"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewUserProfile(c echo.Context) error {
	userUuidVal := c.Get("userUuid")
	if userUuidVal == nil {
		log.Warn(c.Request().Context(), "Failed to get user uuid from context")
		return c.Redirect(http.StatusSeeOther, "/login")
	}
	userUuid := userUuidVal.(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "UserUuid", userUuid, "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}
	discordId, err := h.UserStore.GetDiscordId(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get discord id", "UserUuid", userUuid, "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	userProfileIndex := userprofile.UserProfileIndex(username, discordId, "", "", h.csrfToken(c), h.MinPasswordLength)
	userProfile := userprofile.UserProfile(" | User Profile", true, username, userProfileIndex)
	err = Render(c, userProfile)
	if err != nil {
		log.Error(c.Request().Context(), "Handle View User Profile Failed To Render", "Error", err)
	}
	return nil
}

func (h *Handler) HandleUpdateUserProfile(c echo.Context) error {
	userUuidVal := c.Get("userUuid")
	if userUuidVal == nil {
		log.Warn(c.Request().Context(), "Failed to get user uuid from context")
		return c.Redirect(http.StatusSeeOther, "/login")
	}
	userUuid := userUuidVal.(uuid.UUID)
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "UserUuid", userUuid, "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	discordId := c.FormValue("discordId")
	currentPassword := c.FormValue("currentPassword")
	newPassword := c.FormValue("newPassword")
	confirmNewPassword := c.FormValue("confirmNewPassword")

	// Update discord ID
	if err := h.UserStore.UpdateDiscordId(c.Request().Context(), userUuid, discordId); err != nil {
		log.Error(c.Request().Context(), "Failed to update discord id", "UserUuid", userUuid, "Error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	// Handle password change if any password field is filled
	if currentPassword != "" || newPassword != "" || confirmNewPassword != "" {
		if currentPassword == "" {
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, "Current password is required to change your password", "", h.csrfToken(c), h.MinPasswordLength)
			err := Render(c, userProfileIndex)
			if err != nil {
				log.Error(c.Request().Context(), "Handle Update User Profile Failed To Render", "Error", err)
			}
			return nil
		}

		if newPassword == "" {
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, "New password is required", "", h.csrfToken(c), h.MinPasswordLength)
			err := Render(c, userProfileIndex)
			if err != nil {
				log.Error(c.Request().Context(), "Handle Update User Profile Failed To Render", "Error", err)
			}
			return nil
		}

		if newPassword != confirmNewPassword {
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, "New passwords do not match", "", h.csrfToken(c), h.MinPasswordLength)
			err := Render(c, userProfileIndex)
			if err != nil {
				log.Error(c.Request().Context(), "Handle Update User Profile Failed To Render", "Error", err)
			}
			return nil
		}

		if len(newPassword) < h.MinPasswordLength {
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, fmt.Sprintf("New password must be at least %d characters", h.MinPasswordLength), "", h.csrfToken(c), h.MinPasswordLength)
			err := Render(c, userProfileIndex)
			if err != nil {
				log.Error(c.Request().Context(), "Handle Update User Profile Failed To Render", "Error", err)
			}
			return nil
		}

		valid, err := h.UserStore.ValidateLogin(c.Request().Context(), username, currentPassword)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to validate current password", "Username", username, "Error", err)
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, "An error occurred. Please try again.", "", h.csrfToken(c), h.MinPasswordLength)
			err = Render(c, userProfileIndex)
			if err != nil {
				log.Error(c.Request().Context(), "Handle Update User Profile Failed To Render", "Error", err)
			}
			return nil
		}
		if !valid {
			log.Info(c.Request().Context(), "Invalid current password attempt for user", "Username", username)
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, "Current password is incorrect", "", h.csrfToken(c), h.MinPasswordLength)
			err = Render(c, userProfileIndex)
			if err != nil {
				log.Error(c.Request().Context(), "Handle Update User Profile Failed To Render", "Error", err)
			}
			return nil
		}

		log.Info(c.Request().Context(), "Updating password for user", "Username", username)
		if err := h.UserStore.UpdatePassword(c.Request().Context(), username, newPassword); err != nil {
			log.Error(c.Request().Context(), "Failed to update password", "Username", username, "Error", err)
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, "An error occurred. Please try again.", "", h.csrfToken(c), h.MinPasswordLength)
			err = Render(c, userProfileIndex)
			if err != nil {
				log.Error(c.Request().Context(), "Handle Update User Profile Failed To Render", "Error", err)
			}
			return nil
		}
		// Invalidate all other sessions on password change
		userTok, _ := c.Cookie("sessionToken")
		if userTok != nil && userTok.Value != "" {
			if err := h.UserStore.InvalidateAllUserSessionsExcept(c.Request().Context(), userUuid, userTok.Value); err != nil {
				log.Warn(c.Request().Context(), "Failed to invalidate other sessions", "Username", username, "Error", err)
			}
		}
	}

	log.Info(c.Request().Context(), "Updated profile for user", "Username", username)
	userProfileIndex := userprofile.UserProfileIndex(username, discordId, "", "Profile updated successfully", h.csrfToken(c), h.MinPasswordLength)
	err = Render(c, userProfileIndex)
	if err != nil {
		log.Error(c.Request().Context(), "Handle Update User Profile Failed To Render", "Error", err)
	}
	return nil
}
