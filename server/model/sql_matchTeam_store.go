package model

import (
	"context"
	"database/sql"

	"server/database"
)

type SQLMatchTeamStore struct {
	database.SQLStore
}

func NewSQLMatchTeamStore(db *sql.DB) *SQLMatchTeamStore {
	return &SQLMatchTeamStore{SQLStore: database.NewSQLStore(db)}
}

func (s *SQLMatchTeamStore) AssociateTeam(ctx context.Context, matchTbaId string, teamTbaId string, alliance string, isDqed bool) error {
	return associateTeam(ctx, s.DB, matchTbaId, teamTbaId, alliance, isDqed)
}
