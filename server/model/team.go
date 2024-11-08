package model

import (
	"database/sql"
	"fmt"
	"server/assert"
	"server/tbaHandler"
)

type Team struct {
    TbaId string
    Name string
    RankingScore int
}

func (t *Team) String() string {
    return fmt.Sprintf("Team: {\n TbaId: %s\n Name: %s\n RankingScore: %d\n}", t.TbaId, t.Name, t.RankingScore)
}

func GetTeam(database *sql.DB, tbaId string) *Team {
    query := `Select tbaId, name, COALESCE(rankingScore, 0) As rankingScore From Teams Where tbaId = $1;`
    assert := assert.CreateAssertWithContext("Get Team")
    assert.AddContext("TbaId", tbaId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    team := Team{}
    err = stmt.QueryRow(tbaId).Scan(&team.TbaId, &team.Name, &team.RankingScore)
    if err != nil || team.TbaId == "" {
        return nil
    }
    return &team
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

func ValidPick(database *sql.DB, handler *tbaHandler.TbaHandler, tbaId string, draftId int) bool {
    if tbaId == "" {
        return false
    }

    picked := HasBeenPicked(database, draftId, tbaId)

    events := handler.MakeEventListReq(tbaId)
    //TODO we need to load events for this draft
    draftEvents := []string{"a"}

    validEvent := false
    //Looping here should always be faster because of the small lists
    for _, event := range events {
        for _, draftEvent := range draftEvents {
            if event  == draftEvent {
                validEvent = true
                break
            }
        }
    }

    //TODO Remove
    validEvent = true

    return !picked && validEvent
}
