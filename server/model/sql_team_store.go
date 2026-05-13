package model

import (
	"context"
	"database/sql"
)

type SQLTeamStore struct {
	db *sql.DB
}

func NewSQLTeamStore(db *sql.DB) *SQLTeamStore {
	return &SQLTeamStore{db: db}
}

func (s *SQLTeamStore) GetScore(ctx context.Context, tbaId string) (map[string]int, error) {
	return getScore(ctx, s.db, tbaId)
}

func (s *SQLTeamStore) GetMatchScores(ctx context.Context, tbaId string) ([]MatchTeamScore, error) {
	return getMatchScores(ctx, s.db, tbaId)
}

func (s *SQLTeamStore) GetTeam(ctx context.Context, tbaId string) (*Team, error) {
	return getTeam(ctx, s.db, tbaId)
}

func (s *SQLTeamStore) CreateTeam(ctx context.Context, tbaId string, name string) error {
	return createTeam(ctx, s.db, tbaId, name)
}

func (s *SQLTeamStore) UpdateTeamAllianceScore(ctx context.Context, tbaId string, allianceScore int16) {
	updateTeamAllianceScore(ctx, s.db, tbaId, allianceScore)
}
