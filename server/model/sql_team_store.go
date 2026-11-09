package model

import (
	"context"
	"database/sql"

	"server/database"
)

type SQLTeamStore struct {
	database.SQLStore
}

func NewSQLTeamStore(db *sql.DB) *SQLTeamStore {
	return &SQLTeamStore{SQLStore: database.NewSQLStore(db)}
}

func (s *SQLTeamStore) GetScore(ctx context.Context, tbaId string) (map[string]int, error) {
	return getScore(ctx, s.DB, tbaId)
}

func (s *SQLTeamStore) GetMatchScores(ctx context.Context, tbaId string) ([]MatchTeamScore, error) {
	return getMatchScores(ctx, s.DB, tbaId)
}

func (s *SQLTeamStore) GetTeam(ctx context.Context, tbaId string) (*Team, error) {
	return getTeam(ctx, s.DB, tbaId)
}

func (s *SQLTeamStore) CreateTeam(ctx context.Context, tbaId string, name string) error {
	return createTeam(ctx, s.DB, tbaId, name)
}

func (s *SQLTeamStore) UpdateTeamAllianceScore(ctx context.Context, tbaId string, allianceScore int) error {
	return updateTeamAllianceScore(ctx, s.DB, tbaId, allianceScore)
}
