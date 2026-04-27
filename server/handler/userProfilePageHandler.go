package handler

import (
	"net/http"
	"server/assert"
	"server/log"
	"server/model"
	"server/view/userProfile"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewUserProfile(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle View User Profile")

	userTok, err := c.Cookie("sessionToken")
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get session token", "Ip", c.RealIP())
		err = c.Redirect(http.StatusSeeOther, "/login")
		if err != nil {
			return err
		}
		return echo.ErrUnauthorized
	}

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userUuid)
	discordId := model.GetDiscordId(h.Database, userUuid)

	userProfileIndex := userprofile.UserProfileIndex(username, discordId, "", "")
	userProfile := userprofile.UserProfile(" | User Profile", true, username, userProfileIndex)
	err = h.Render(c, userProfile)
	assert.NoError(err, "Handle View User Profile Failed To Render")
	return nil
}

func (h *Handler) HandleUpdateUserProfile(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Update User Profile")

	userTok, err := c.Cookie("sessionToken")
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get session token", "Ip", c.RealIP())
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userUuid)

	discordId := c.FormValue("discordId")
	currentPassword := c.FormValue("currentPassword")
	newPassword := c.FormValue("newPassword")
	confirmNewPassword := c.FormValue("confirmNewPassword")

	// Update discord ID
	model.UpdateDiscordId(h.Database, userUuid, discordId)

	// Handle password change if any password field is filled
	if currentPassword != "" || newPassword != "" || confirmNewPassword != "" {
		if currentPassword == "" {
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, "Current password is required to change your password", "")
			err = h.Render(c, userProfileIndex)
			assert.NoError(err, "Handle Update User Profile Failed To Render")
			return nil
		}

		if newPassword == "" {
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, "New password is required", "")
			err = h.Render(c, userProfileIndex)
			assert.NoError(err, "Handle Update User Profile Failed To Render")
			return nil
		}

		if newPassword != confirmNewPassword {
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, "New passwords do not match", "")
			err = h.Render(c, userProfileIndex)
			assert.NoError(err, "Handle Update User Profile Failed To Render")
			return nil
		}

		if len(newPassword) < 6 {
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, "New password must be at least 6 characters", "")
			err = h.Render(c, userProfileIndex)
			assert.NoError(err, "Handle Update User Profile Failed To Render")
			return nil
		}

		if !model.ValidateLogin(h.Database, username, currentPassword) {
			log.Info(c.Request().Context(), "Invalid current password attempt for user", "Username", username)
			userProfileIndex := userprofile.UserProfileIndex(username, discordId, "Current password is incorrect", "")
			err = h.Render(c, userProfileIndex)
			assert.NoError(err, "Handle Update User Profile Failed To Render")
			return nil
		}

		log.Info(c.Request().Context(), "Updating password for user", "Username", username)
		model.UpdatePassword(h.Database, username, newPassword)
	}

	log.Info(c.Request().Context(), "Updated profile for user", "Username", username)
	userProfileIndex := userprofile.UserProfileIndex(username, discordId, "", "Profile updated successfully")
	err = h.Render(c, userProfileIndex)
	assert.NoError(err, "Handle Update User Profile Failed To Render")
	return nil
}
