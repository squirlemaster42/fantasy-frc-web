package model

import (
	"context"
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

func associateTeam(ctx context.Context, db database.DBTX, matchTbaId string, teamTbaId string, alliance string, isDqed bool) error {
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
	if _, err := database.Exec(ctx, db, query, teamTbaId, matchTbaId, alliance, isDqed); err != nil {
		return fmt.Errorf("failed to associate team: %w", err)
	}
	return nil
}
