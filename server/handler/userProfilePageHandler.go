package handler

import (
	"fmt"
	"server/log"
	"server/model"
	"server/view/userProfile"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewUserProfile(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	username := model.GetUsername(c.Request().Context(), h.Database, userUuid)
	discordId := model.GetDiscordId(c.Request().Context(), h.Database, userUuid)

	userProfileIndex := userprofile.UserProfileIndex(username, discordId, "", "", h.csrfToken(c), h.MinPasswordLength)
	userProfile := userprofile.UserProfile(" | User Profile", true, username, userProfileIndex)
	err := Render(c, userProfile)
	if err != nil {
		log.Error(c.Request().Context(), "Handle View User Profile Failed To Render", "Error", err)
	}
	return nil
}

func (h *Handler) HandleUpdateUserProfile(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	username := model.GetUsername(c.Request().Context(), h.Database, userUuid)

	discordId := c.FormValue("discordId")
	currentPassword := c.FormValue("currentPassword")
	newPassword := c.FormValue("newPassword")
	confirmNewPassword := c.FormValue("confirmNewPassword")

	// Update discord ID
	model.UpdateDiscordId(c.Request().Context(), h.Database, userUuid, discordId)

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

		valid, err := model.ValidateLogin(c.Request().Context(), h.Database, username, currentPassword)
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
		if err := model.UpdatePassword(c.Request().Context(), h.Database, username, newPassword); err != nil {
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
			if err := model.InvalidateAllUserSessionsExcept(c.Request().Context(), h.Database, userUuid, userTok.Value); err != nil {
				log.Warn(c.Request().Context(), "Failed to invalidate other sessions", "Username", username, "Error", err)
			}
		}
	}

	log.Info(c.Request().Context(), "Updated profile for user", "Username", username)
	userProfileIndex := userprofile.UserProfileIndex(username, discordId, "", "Profile updated successfully", h.csrfToken(c), h.MinPasswordLength)
	err := Render(c, userProfileIndex)
	if err != nil {
		log.Error(c.Request().Context(), "Handle Update User Profile Failed To Render", "Error", err)
	}
	return nil
}
