package model

import (
	"context"
	"database/sql"
)

type SQLMatchStore struct {
	db *sql.DB
}

func NewSQLMatchStore(db *sql.DB) *SQLMatchStore {
	return &SQLMatchStore{db: db}
}

func (s *SQLMatchStore) AddMatch(ctx context.Context, tbaId string) {
	addMatch(ctx, s.db, tbaId)
}

func (s *SQLMatchStore) UpdateScore(ctx context.Context, tbaId string, redScore int, blueScore int) {
	updateScore(ctx, s.db, tbaId, redScore, blueScore)
}

func (s *SQLMatchStore) GetMatch(ctx context.Context, tbaId string) *Match {
	return getMatch(ctx, s.db, tbaId)
}
