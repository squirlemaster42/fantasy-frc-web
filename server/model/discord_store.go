package model

import (
	"context"
	"database/sql"
)

type DiscordStore interface {
	GetPlayerDiscordId(ctx context.Context, draftPlayerId int) (sql.NullString, error)
	GetDraftWebhook(ctx context.Context, draftId int) (string, error)
}
