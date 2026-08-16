package model

import (
	"context"
	"database/sql"
	"fmt"
	"server/database"
)

type MatchTeam struct {
	TeamTbaId  string
	MatchTbaId string
	Alliance   string
	IsDqed     bool
}

func (m *MatchTeam) String() string {
	return fmt.Sprintf("MatchTeam: {\nTeamTbaId: %s\n MatchTbaId: %s\n Alliance: %s\n IsDqed: %t\n}",
		m.TeamTbaId, m.MatchTbaId, m.Alliance, m.IsDqed)
}

func associateTeam(ctx context.Context, db *sql.DB, matchTbaId string, teamTbaId string, alliance string, isDqed bool) error {
	team, err := getTeam(ctx, db, teamTbaId)
	if err != nil {
		return fmt.Errorf("failed to get team: %w", err)
	}
	if team == nil {
		if err := createTeam(ctx, db, teamTbaId, ""); err != nil {
			return fmt.Errorf("failed to create team: %w", err)
		}
	}

	query := `INSERT INTO Matches_Teams (team_tbaId, match_tbaId, alliance, isDqed) Values ($1, $2, $3, $4)
        On Conflict (team_tbaId, match_tbaId) Do Update Set alliance = excluded.alliance, isDqed = excluded.isDqed;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return err
	}
	defer database.CloseStatement(ctx, stmt, "AssociateTeam")
	_, err = stmt.ExecContext(ctx, teamTbaId, matchTbaId, alliance, isDqed)
	if err != nil {
		return fmt.Errorf("failed to associate team: %w", err)
	}
	return nil
}
