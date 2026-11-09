package model

import (
	"context"
	"database/sql"

	"server/database"
)

type SQLDiscordStore struct {
	database.SQLStore
}

func NewSQLDiscordStore(db *sql.DB) *SQLDiscordStore {
	return &SQLDiscordStore{SQLStore: database.NewSQLStore(db)}
}

func (s *SQLDiscordStore) GetPlayerDiscordId(ctx context.Context, draftPlayerId int) (sql.NullString, error) {
	return getPlayerDiscordId(ctx, s.DB, draftPlayerId)
}

func (s *SQLDiscordStore) GetDraftWebhook(ctx context.Context, draftId int) (string, error) {
	return getDraftWebhook(ctx, s.DB, draftId)
}

func (s *SQLDiscordStore) GetPlayerPickNotificationId(ctx context.Context, draftPlayerId int) (sql.NullString, error) {
	return getPlayerPickNotificationId(ctx, s.DB, draftPlayerId)
}
