package model

import (
	"context"
	"database/sql"
)

type SQLMatchStore struct {
	database *sql.DB
}

func NewSQLMatchStore(database *sql.DB) *SQLMatchStore {
	return &SQLMatchStore{database: database}
}

func (s *SQLMatchStore) AddMatch(ctx context.Context, tbaId string) error {
	return addMatch(ctx, s.database, tbaId)
}

func (s *SQLMatchStore) UpdateScore(ctx context.Context, tbaId string, redScore int, blueScore int) error {
	return updateScore(ctx, s.database, tbaId, redScore, blueScore)
}

func (s *SQLMatchStore) GetMatch(ctx context.Context, tbaId string) (*Match, error) {
	return getMatch(ctx, s.database, tbaId)
}
