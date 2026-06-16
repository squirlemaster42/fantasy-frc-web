package handler

import (
	"server/background"
	"server/cache"
	"server/discord"
	"server/draft"
	"server/model"
	"server/scorer"
	"server/tbaHandler"

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
