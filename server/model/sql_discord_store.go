package model

import (
	"context"
	"database/sql"
)

type SQLDiscordStore struct {
	db *sql.DB
}

func NewSQLDiscordStore(db *sql.DB) *SQLDiscordStore {
	return &SQLDiscordStore{db: db}
}

func (s *SQLDiscordStore) GetPlayerDiscordId(ctx context.Context, draftPlayerId int) (sql.NullString, error) {
	return getPlayerDiscordId(ctx, s.db, draftPlayerId)
}

func (s *SQLDiscordStore) GetDraftWebhook(ctx context.Context, draftId int) (string, error) {
	return getDraftWebhook(ctx, s.db, draftId)
}
