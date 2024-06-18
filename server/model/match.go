package model

import (
	"database/sql"
	"server/assert"
)

type Match struct {
    TbaId string
    Played bool
    RedScore int
    BlueScore int
    RedAlliance []string
    BlueAlliance []string
    DqedTeams []string
}

func AddMatch(database *sql.DB, tbaId string) {
    query := `INSERT INTO Matches (tbaid, played, redscore, bluescore) Values ($1, $2, $3, $4);`
    stmt, err := database.Prepare(query)
    a := assert.CreateAssertWithContext("Add Match")
    a.AddContext("MatchTbaId", tbaId)
    a.NoError(err, "Failed to prepare query")
    _, err = stmt.Exec(tbaId, false, 0, 0)
    a.NoError(err, "Failed to insert into database")
}

func UpdateScore(database *sql.DB, tbaId string, redScore int, blueScore int) {
    query := `UPDATE Matches Set played = $1, redscore = $2, bluescore = $3 Where tbaid = $4;`
    a := assert.CreateAssertWithContext("Update Match")
    a.AddContext("MatchTbaId", tbaId)
    a.AddContext("RedScore", redScore)
    a.AddContext("BlueScore", blueScore)
    stmt, err := database.Prepare(query)
    a.NoError(err, "Failed to prepare query")
    _, err = stmt.Exec(true, 0, 0, tbaId)
    a.NoError(err, "Failed to update in database")
}

//All validity checks should be done before now, so we can have this many asserts here
func GetMatch(database *sql.DB, tbaId string) *Match {
    query := `Select tbaid, played, redscore, bluescore Where tbaid = $1;`
    stmt, err := database.Prepare(query)
    a := assert.CreateAssertWithContext("Add Match")
    a.AddContext("MatchTbaId", tbaId)
    a.NoError(err, "Failed to prepare query")
    match := Match{}
    err = stmt.QueryRow(tbaId).Scan(&match.TbaId, &match.Played, &match.RedScore, &match.BlueScore)
    a.NoError(err, "Failed to query from database")
    return &match
}
