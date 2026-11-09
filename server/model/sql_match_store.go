package model

import (
	"context"
	"database/sql"

	"server/database"
)

type SQLMatchStore struct {
	database.SQLStore
}

func NewSQLMatchStore(db *sql.DB) *SQLMatchStore {
	return &SQLMatchStore{SQLStore: database.NewSQLStore(db)}
}

func (s *SQLMatchStore) AddMatch(ctx context.Context, tbaId string) error {
	return addMatch(ctx, s.DB, tbaId)
}

func (s *SQLMatchStore) UpdateScore(ctx context.Context, tbaId string, redScore int, blueScore int) error {
	return updateScore(ctx, s.DB, tbaId, redScore, blueScore)
}

func (s *SQLMatchStore) GetMatch(ctx context.Context, tbaId string) (*Match, error) {
	return getMatch(ctx, s.DB, tbaId)
}
