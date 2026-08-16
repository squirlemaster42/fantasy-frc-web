package handler

import (
	"net/http"
	"server/authentication"
	"server/cache"
	"server/discord"
	"server/draft"
	"server/log"
	"server/model"
	"server/scorer"
	"server/tbaHandler"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type StorageGroup struct {
	DraftStore model.DraftStore
	UserStore  model.UserStore
	TeamStore  model.TeamStore
}

type ServiceGroup struct {
	AuthService       authentication.AuthService
	TBAHandler        tbaHandler.TBAInterface
	DraftActorMap     *draft.DraftActorMap
	Scorer            *scorer.Scorer
	AvatarStore       cache.AvatarStoreInterface
	DiscordWebhookBus discord.DiscordNotifier
}

type ConfigGroup struct {
	TbaWebhookSecret            string
	TbaVerificationCode         string
	SecureHttpCookie            bool
	MinPasswordLength           int
	MinUsernameLength           int
	MaxUsernameLength           int
	UsernameAllowedSpecialChars string
	AllowedOrigin               string
}

type Handler struct {
	Stores   StorageGroup
	Services ServiceGroup
	Config   ConfigGroup
}

func (h *Handler) csrfToken(c echo.Context) string {
	tok, _ := c.Get("csrfToken").(string)
	return tok
}

func (h *Handler) getAuthenticatedUsername(c echo.Context, userUuid uuid.UUID) (string, error) {
	username, err := h.Stores.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "error", err)
		return "", echo.NewHTTPError(http.StatusInternalServerError, "An error occurred")
	}
	return username, nil
}

// requireUserUuid returns the authenticated user's UUID from the Echo context.
// If the UUID is missing or has the wrong type, it redirects to /login.
func (h *Handler) requireUserUuid(c echo.Context) (uuid.UUID, error) {
	userUuidVal := c.Get("userUuid")
	userUuid, ok := userUuidVal.(uuid.UUID)
	if !ok {
		log.Warn(c.Request().Context(), "Missing or invalid user uuid in context", "ip", c.RealIP(), "path", c.Request().URL.Path)
		return uuid.UUID{}, c.Redirect(http.StatusSeeOther, "/login")
	}
	return userUuid, nil
}

// requireUser returns the authenticated user's UUID and username.
// If the UUID is missing or has the wrong type, it redirects to /login.
// If the username lookup fails, it returns an internal server error.
func (h *Handler) requireUser(c echo.Context) (uuid.UUID, string, error) {
	userUuid, err := h.requireUserUuid(c)
	if err != nil {
		return uuid.UUID{}, "", err
	}
	username, err := h.getAuthenticatedUsername(c, userUuid)
	if err != nil {
		return uuid.UUID{}, "", err
	}
	return userUuid, username, nil
}
