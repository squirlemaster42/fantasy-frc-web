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
		log.Error(c.Request().Context(), "Failed to get username", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}
	discordId, err := h.UserStore.GetDiscordId(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get discord id", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	userProfileIndex := userprofile.UserProfileIndex(username, discordId, "", "", h.csrfToken(c), h.MinPasswordLength)
	userProfile := userprofile.UserProfile(" | User Profile", true, username, userProfileIndex)
	if err := Render(c, userProfile); err != nil {
		log.Error(c.Request().Context(), "Handle View User Profile Failed To Render", "error", err)
		return err
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
		log.Error(c.Request().Context(), "Failed to get username", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	discordId := c.FormValue("discordId")
	currentPassword := c.FormValue("currentPassword")
	newPassword := c.FormValue("newPassword")
	confirmNewPassword := c.FormValue("confirmNewPassword")

	renderProfile := func(message string) error {
		userProfileIndex := userprofile.UserProfileIndex(username, discordId, message, "", h.csrfToken(c), h.MinPasswordLength)
		if err := Render(c, userProfileIndex); err != nil {
			log.Error(c.Request().Context(), "Handle Update User Profile Failed To Render", "error", err)
			return err
		}
		return nil
	}

	// Update discord ID
	if err := h.UserStore.UpdateDiscordId(c.Request().Context(), userUuid, discordId); err != nil {
		log.Error(c.Request().Context(), "Failed to update discord id", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	// Handle password change if any password field is filled
	if currentPassword != "" || newPassword != "" || confirmNewPassword != "" {
		if currentPassword == "" {
			return renderProfile("Current password is required to change your password")
		}

		if newPassword == "" {
			return renderProfile("New password is required")
		}

		if newPassword != confirmNewPassword {
			return renderProfile("New passwords do not match")
		}

		if len(newPassword) < h.MinPasswordLength {
			return renderProfile(fmt.Sprintf("New password must be at least %d characters", h.MinPasswordLength))
		}

		valid, err := h.UserStore.ValidateLogin(c.Request().Context(), username, currentPassword)
		if err != nil {
			log.Error(c.Request().Context(), "Failed to validate current password", "username", username, "error", err)
			return renderProfile("An error occurred. Please try again.")
		}
		if !valid {
			log.Warn(c.Request().Context(), "Invalid current password attempt for user", "username", username)
			return renderProfile("Current password is incorrect")
		}

		log.Debug(c.Request().Context(), "Updating password for user", "username", username)
		if err := h.UserStore.UpdatePassword(c.Request().Context(), username, newPassword); err != nil {
			log.Error(c.Request().Context(), "Failed to update password", "username", username, "error", err)
			return renderProfile("An error occurred. Please try again.")
		}
		// Invalidate all other sessions on password change
		userTok, _ := c.Cookie("sessionToken")
		if userTok != nil && userTok.Value != "" {
			if err := h.UserStore.InvalidateAllUserSessionsExcept(c.Request().Context(), userUuid, userTok.Value); err != nil {
				log.Error(c.Request().Context(), "Failed to invalidate other sessions", "username", username, "error", err)
			}
		}
	}

	log.Debug(c.Request().Context(), "Updated profile for user", "username", username)
	userProfileIndex := userprofile.UserProfileIndex(username, discordId, "", "Profile updated successfully", h.csrfToken(c), h.MinPasswordLength)
	if err := Render(c, userProfileIndex); err != nil {
		log.Error(c.Request().Context(), "Handle Update User Profile Failed To Render", "error", err)
		return err
	}
	return nil
}
