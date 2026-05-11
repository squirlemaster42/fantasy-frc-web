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

func AssocateTeam(ctx context.Context, database *sql.DB, matchTbaId string, teamTbaId string, alliance string, isDqed bool) {
	if GetTeam(ctx, database, teamTbaId) == nil {
		CreateTeam(ctx, database, teamTbaId, "")
	}

	query := `INSERT INTO Matches_Teams (team_tbaId, match_tbaId, alliance, isDqed) Values ($1, $2, $3, $4)
        On Conflict (team_tbaId, match_tbaId) Do Update Set alliance = excluded.alliance, isDqed = excluded.isDqed;`
	assert := assert.CreateAssertWithContext("Associate Team")
	assert.AddContext("Match Id", matchTbaId)
	assert.AddContext("Team Id", teamTbaId)
	assert.AddContext("Alliance", alliance)
	assert.AddContext("Is Dqed", isDqed)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "AssocateTeam: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, teamTbaId, matchTbaId, alliance, isDqed)
	assert.NoError(ctx, err, "Failed to associate team")
}
