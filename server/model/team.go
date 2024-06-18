package model

import (
	"database/sql"
	"server/assert"
)

type Team struct {
    TbaId string
    Name string
    RankingScore int
}

func GetTeam(database *sql.DB, tbaId string) Team {
    query := `Select tbaId, name, rankingScore From Teams Where tbaId = $1;`
    assert := assert.CreateAssertWithContext("Get Team")
    assert.AddContext("TbaId", tbaId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    team := Team{}
    _, err = stmt.Exec(&team.TbaId, &team.Name, &team.RankingScore)
    assert.NoError(err, "Failed to Get Teab")
    return team
}

func CreateTeam(database *sql.DB, tbaId string, name string) {
    query := `INSERT INTO Teams (tbaId, name) Values ($1, $2);`
    assert := assert.CreateAssertWithContext("Create Team")
    assert.AddContext("Tba Id", tbaId)
    assert.AddContext("Name", name)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(tbaId, name)
    assert.NoError(err, "Failed to create team")

}

func UpdateTeamRankingScore(database *sql.DB, tbaId string, rankingScore int) {
    query := `Update Teams Set rankingScore = $1 where tbaId = $2;`
    assert := assert.CreateAssertWithContext("Update Team Ranking Score")
    assert.AddContext("Tba Id", tbaId)
    assert.AddContext("Ranking Score", rankingScore)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(rankingScore, tbaId)
    assert.NoError(err, "Failed to associate team")
}