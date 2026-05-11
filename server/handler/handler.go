package handler

import (
	"server/background"
	"server/cache"
	"server/discord"
	"server/draft"
	"server/model"
	"server/scorer"
	"server/tbaHandler"
)

type Handler struct {
	DraftStore          model.DraftStore
	UserStore           model.UserStore
	TeamStore           model.TeamStore
	TbaHandler          tbaHandler.TbaHandler
	DraftManager        *draft.DraftManager
	DraftDaemon         *background.DraftDaemon
	Scorer              *scorer.Scorer
	AvatarStore         *cache.AvatarStore
	TbaWebhookSecret    string
	TbaVerificationCode string
	DiscordBus          *discord.DiscordWebhookBus
	SecureHttpCookie    bool
}
