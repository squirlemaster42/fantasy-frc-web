package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"server/authentication"
	"server/discord"
	"server/log"
	"server/model"
	"server/view/userProfile"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewUserProfile(c echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()

	discordId, err := h.Stores.UserStore.GetDiscordId(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to get discord id", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	drafts, preferences, err := h.loadUserNotificationSettings(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to load user notification settings", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	userProfileIndex := userprofile.UserProfileIndex(
		userprofile.ProfileData{
			Username:          username,
			DiscordId:         discordId,
			Drafts:            drafts,
			Preferences:       preferences,
			MinPasswordLength: h.Config.MinPasswordLength,
			CsrfToken:         h.csrfToken(c),
		},
	)
	userProfile := userprofile.UserProfile("User Profile", true, username, userProfileIndex)
	if err := Render(c, userProfile); err != nil {
		log.Error(ctx, "Handle View User Profile Failed To Render", "error", err)
		return err
	}
	return nil
}

func (h *Handler) HandleUpdateUserProfile(c echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	discordId := strings.TrimSpace(c.FormValue("discordId"))

	drafts, preferences, err := h.loadUserNotificationSettings(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to load user notification settings", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	// Validate Discord ID if one was provided.
	if discordId != "" && !discord.IsValidId(discordId) {
		return h.renderAccountCard(c, ctx, username, discordId, drafts, preferences,
			fmt.Sprintf("Discord ID must be a numeric snowflake with at least %d characters", discord.DiscordMinIdLength()),
			"error")
	}

	if err := h.Stores.UserStore.UpdateDiscordId(ctx, userUuid, discordId); err != nil {
		log.Error(ctx, "Failed to update discord id", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	log.Debug(ctx, "Updated discord id for user", "username", username)
	return h.renderAccountCard(c, ctx, username, discordId, drafts, preferences, "Discord ID updated successfully", "success")
}

func (h *Handler) HandleUpdateUserPassword(c echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()
	currentPassword := c.FormValue("currentPassword")
	newPassword := c.FormValue("newPassword")
	confirmNewPassword := c.FormValue("confirmNewPassword")

	discordId, err := h.Stores.UserStore.GetDiscordId(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to get discord id", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	drafts, preferences, err := h.loadUserNotificationSettings(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to load user notification settings", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	renderPasswordCard := func(message, messageType string) error {
		return h.renderPasswordCard(c, ctx, username, discordId, drafts, preferences, message, messageType)
	}

	if currentPassword == "" {
		return renderPasswordCard("Current password is required to change your password", "error")
	}

	if newPassword == "" {
		return renderPasswordCard("New password is required", "error")
	}

	if newPassword != confirmNewPassword {
		return renderPasswordCard("Passwords do not match", "error")
	}

	if err := h.Services.AuthService.ChangePassword(ctx, userUuid, username, currentPassword, newPassword); err != nil {
		switch {
		case errors.Is(err, authentication.ErrInvalidCredentials):
			log.Warn(ctx, "Invalid current password attempt for user", "username", username)
			return renderPasswordCard("Current password is incorrect", "error")
		case authentication.IsValidationError(err):
			return renderPasswordCard(err.Error(), "error")
		default:
			log.Error(ctx, "Failed to change password", "username", username, "error", err)
			return renderPasswordCard("An error occurred. Please try again.", "error")
		}
	}

	log.Debug(ctx, "Updated password for user", "username", username)
	return renderPasswordCard("Password updated successfully", "success")
}

func (h *Handler) HandleUpdateUserNotificationPreferences(c echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	ctx := c.Request().Context()

	drafts, preferences, err := h.loadUserNotificationSettings(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to load user notification settings", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	discordId, err := h.Stores.UserStore.GetDiscordId(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to get discord id", "userUuid", userUuid, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	renderNotificationsCard := func(message, messageType string) error {
		return h.renderNotificationsCard(c, ctx, username, discordId, drafts, preferences, message, messageType)
	}

	for _, draft := range drafts {
		upcomingMatch := c.FormValue(fmt.Sprintf("draft_%d_upcomingMatch", draft.Id)) != ""
		pickTurn := c.FormValue(fmt.Sprintf("draft_%d_pickTurn", draft.Id)) != ""

		if err := h.Stores.DraftStore.UpdateUserDraftNotificationPreference(ctx, userUuid, draft.Id, upcomingMatch, pickTurn); err != nil {
			log.Error(ctx, "Failed to update notification preference", "userUuid", userUuid, "draftId", draft.Id, "error", err)
			return renderNotificationsCard("An error occurred while saving preferences", "error")
		}

		// Update local map so the re-rendered UI reflects the saved state.
		preferences[draft.Id] = model.DraftNotificationPreference{
			UserUuid:      userUuid,
			DraftId:       draft.Id,
			UpcomingMatch: upcomingMatch,
			PickTurn:      pickTurn,
		}
	}

	log.Debug(ctx, "Updated notification preferences for user", "username", username)
	return renderNotificationsCard("Notification preferences updated", "success")
}

// loadUserNotificationSettings returns the drafts a user can receive notifications for
// and their current preferences. It returns a map keyed by DraftId.
func (h *Handler) loadUserNotificationSettings(ctx context.Context, userUuid uuid.UUID) ([]model.DraftModel, map[int]model.DraftNotificationPreference, error) {
	drafts, err := h.Stores.DraftStore.GetDraftsForUser(ctx, userUuid)
	if err != nil {
		return nil, nil, err
	}

	preferences, err := h.Stores.DraftStore.GetUserDraftNotificationPreferences(ctx, userUuid)
	if err != nil {
		return nil, nil, err
	}

	if preferences == nil {
		preferences = make(map[int]model.DraftNotificationPreference)
	}

	return drafts, preferences, nil
}

func (h *Handler) renderAccountCard(c echo.Context, ctx context.Context, username string, discordId string, drafts []model.DraftModel, preferences map[int]model.DraftNotificationPreference, message string, messageType string) error {
	card := userprofile.AccountCard(
		userprofile.ProfileData{
			Username:           username,
			DiscordId:          discordId,
			Drafts:             drafts,
			Preferences:        preferences,
			MinPasswordLength:  h.Config.MinPasswordLength,
			CsrfToken:          h.csrfToken(c),
			AccountMessage:     message,
			AccountMessageType: messageType,
		},
	)
	if err := Render(c, card); err != nil {
		log.Error(ctx, "Render account card failed", "error", err)
		return err
	}
	return nil
}

func (h *Handler) renderPasswordCard(c echo.Context, ctx context.Context, username string, discordId string, drafts []model.DraftModel, preferences map[int]model.DraftNotificationPreference, message string, messageType string) error {
	card := userprofile.PasswordCard(
		userprofile.ProfileData{
			Username:           username,
			DiscordId:          discordId,
			Drafts:             drafts,
			Preferences:        preferences,
			MinPasswordLength:  h.Config.MinPasswordLength,
			CsrfToken:          h.csrfToken(c),
			PasswordMessage:    message,
			PasswordMessageType: messageType,
		},
	)
	if err := Render(c, card); err != nil {
		log.Error(ctx, "Render password card failed", "error", err)
		return err
	}
	return nil
}

func (h *Handler) renderNotificationsCard(c echo.Context, ctx context.Context, username string, discordId string, drafts []model.DraftModel, preferences map[int]model.DraftNotificationPreference, message string, messageType string) error {
	card := userprofile.NotificationsCard(
		userprofile.ProfileData{
			Username:                username,
			DiscordId:               discordId,
			Drafts:                  drafts,
			Preferences:             preferences,
			MinPasswordLength:       h.Config.MinPasswordLength,
			CsrfToken:               h.csrfToken(c),
			NotificationMessage:     message,
			NotificationMessageType: messageType,
		},
	)
	if err := Render(c, card); err != nil {
		log.Error(ctx, "Render notifications card failed", "error", err)
		return err
	}
	return nil
}
