package model

import (
	"database/sql"
	"fmt"
	"server/assert"
)

type MatchTeam struct {
    TeamTbaId string
    MatchTbaId string
    Alliance string
    IsDqed bool
}

func (m *MatchTeam) String() string {
    return fmt.Sprintf("MatchTeam: {\nTeamTbaId: %s\n MatchTbaId: %s\n Alliance: %s\n IsDqed: %t\n}",
        m.TeamTbaId, m.MatchTbaId, m.Alliance, m.IsDqed)
}

func AssocateTeam(database *sql.DB, matchTbaId string, teamTbaId string, alliance string, isDqed bool) {
    query := `INSERT INTO Matches_Teams (team_tbaId, match_tbaId, alliance, isDqed) Values ($1, $2, $3, $4);`
    assert := assert.CreateAssertWithContext("Associate Team")
    assert.AddContext("Match Id", matchTbaId)
    assert.AddContext("Team Id", teamTbaId)
    assert.AddContext("Alliance", alliance)
    assert.AddContext("Is Dqed", isDqed)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(teamTbaId, matchTbaId, alliance, isDqed)
    assert.NoError(err, "Failed to associate team")
}
