package model

import (
	"context"
	"database/sql"
	"fmt"
	"server/assert"
	"server/log"
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

func associateTeam(ctx context.Context, database *sql.DB, matchTbaId string, teamTbaId string, alliance string, isDqed bool) error {
	team, err := getTeam(ctx, database, teamTbaId)
	if err != nil {
		return fmt.Errorf("failed to get team: %w", err)
	}
	if team == nil {
		if err := createTeam(ctx, database, teamTbaId, ""); err != nil {
			return fmt.Errorf("failed to create team: %w", err)
		}
	}

	query := `INSERT INTO Matches_Teams (team_tbaId, match_tbaId, alliance, isDqed) Values ($1, $2, $3, $4)
        On Conflict (team_tbaId, match_tbaId) Do Update Set alliance = excluded.alliance, isDqed = excluded.isDqed;`
	assert := assert.CreateAssertWithContext("Associate Team")
	assert.AddContext("Match Id", matchTbaId)
	assert.AddContext("Team Id", teamTbaId)
	assert.AddContext("Alliance", alliance)
	assert.AddContext("Is Dqed", isDqed)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "AssociateTeam: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, teamTbaId, matchTbaId, alliance, isDqed)
	if err != nil {
		return fmt.Errorf("failed to associate team: %w", err)
	}
	return nil
}
