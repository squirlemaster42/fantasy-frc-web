package handler

import (
	"net/http"
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
