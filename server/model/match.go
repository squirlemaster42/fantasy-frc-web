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
}

func AddMatch(database *sql.DB, tbaId string) {
    query := `INSERT INTO Matches (tbaid, played, redscore, bluescore) Values ($1, $2, $3, $4);`
    stmt, err := database.Prepare(query)
    assert.NoErrorCF(err, "Failed to prepare add match query")
    _, err = stmt.Exec(tbaId, false, 0, 0)
    assert.NoErrorCF(err, "Failed to insert match into database")
}

func UpdateScore(database *sql.DB, tbaId string, redScore int, blueScore int) {

}

func GetMatch(database *sql.DB, tbaId string) (error, Match) {
    return nil, Match{}
}
