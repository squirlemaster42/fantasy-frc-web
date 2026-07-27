package model

import (
	"context"
	"database/sql"
)

type SQLMatchTeamStore struct {
	database *sql.DB
}

func NewSQLMatchTeamStore(database *sql.DB) *SQLMatchTeamStore {
	return &SQLMatchTeamStore{database: database}
}

func (s *SQLMatchTeamStore) AssociateTeam(ctx context.Context, matchTbaId string, teamTbaId string, alliance string, isDqed bool) error {
	return associateTeam(ctx, s.database, matchTbaId, teamTbaId, alliance, isDqed)
}
