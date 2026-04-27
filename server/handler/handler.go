package handler

import (
	"database/sql"
	"server/cache"
	"server/discord"
	"server/draft"
	"server/scorer"
	"server/tbaHandler"
)

type Handler struct {
	Database             *sql.DB
	TbaHandler           tbaHandler.TbaHandler
	DraftManager         *draft.DraftManager
	Scorer               *scorer.Scorer
	AvatarStore          *cache.AvatarStore
	TbaWebhookSecret     string
	TbaVerificationCode  string
	DiscordBus           *discord.DiscordWebhookBus
    SecureHttpCookie     bool
    FaroProxySecret     string
    FaroAlloyInternalURL string
    FaroAlloyBearerToken string
}
