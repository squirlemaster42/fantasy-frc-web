package handler

import (
	"net/http"
	"server/background"
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

type Handler struct {
	DraftStore          model.DraftStore
	UserStore           model.UserStore
	TeamStore           model.TeamStore
	TBAHandler          tbaHandler.TBAHandler
	DraftActorMap 		*draft.DraftActorMap
	DraftDaemon         *background.DraftDaemon
	Scorer              *scorer.Scorer
	AvatarStore         *cache.AvatarStore
	TbaWebhookSecret    string
	TbaVerificationCode string
	DiscordWebhookBus   *discord.DiscordWebhookBus
	SecureHttpCookie    bool
	MinPasswordLength   int
	CsrfSecret          string
	AllowedOrigin       string
}

func (h *Handler) csrfToken(c echo.Context) string {
	tok, _ := c.Get("csrfToken").(string)
	return tok
}

func (h *Handler) getAuthenticatedUsername(c echo.Context, userUuid uuid.UUID) (string, error) {
	username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get username", "error", err)
		return "", echo.NewHTTPError(http.StatusInternalServerError, "An error occurred")
	}
	return username, nil
}
