package handler

import (
	"database/sql"
	"server/background"
	"server/cache"
	"server/draft"
	"server/scorer"
	"server/tbaHandler"
)

type Handler struct {
	Database         *sql.DB
	TbaHandler       tbaHandler.TbaHandler
	DraftManager     *draft.DraftManager
	DraftDaemon      *background.DraftDaemon
	Scorer           *scorer.Scorer
	AvatarStore      *cache.AvatarStore
	TbaWebhookSecret string
}
