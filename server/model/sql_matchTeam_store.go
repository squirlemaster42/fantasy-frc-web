package model

import (
	"context"
	"database/sql"
)

type SQLMatchTeamStore struct {
	db *sql.DB
}

func NewSQLMatchTeamStore(db *sql.DB) *SQLMatchTeamStore {
	return &SQLMatchTeamStore{db: db}
}

func (s *SQLMatchTeamStore) AssocateTeam(ctx context.Context, matchTbaId string, teamTbaId string, alliance string, isDqed bool) error {
	return assocateTeam(ctx, s.db, matchTbaId, teamTbaId, alliance, isDqed)
}
