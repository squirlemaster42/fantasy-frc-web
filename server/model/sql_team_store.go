package model

import (
	"context"
	"database/sql"
)

type SQLTeamStore struct {
	database *sql.DB
}

func NewSQLTeamStore(database *sql.DB) *SQLTeamStore {
	return &SQLTeamStore{database: database}
}

func (s *SQLTeamStore) GetScore(ctx context.Context, tbaId string) (map[string]int, error) {
	return getScore(ctx, s.database, tbaId)
}

func (s *SQLTeamStore) GetMatchScores(ctx context.Context, tbaId string) ([]MatchTeamScore, error) {
	return getMatchScores(ctx, s.database, tbaId)
}

func (s *SQLTeamStore) GetTeam(ctx context.Context, tbaId string) (*Team, error) {
	return getTeam(ctx, s.database, tbaId)
}

func (s *SQLTeamStore) CreateTeam(ctx context.Context, tbaId string, name string) error {
	return createTeam(ctx, s.database, tbaId, name)
}

func (s *SQLTeamStore) UpdateTeamAllianceScore(ctx context.Context, tbaId string, allianceScore int16) error {
	return updateTeamAllianceScore(ctx, s.database, tbaId, allianceScore)
}
