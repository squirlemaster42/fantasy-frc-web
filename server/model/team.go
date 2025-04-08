package model

import (
    "database/sql"
    "fmt"
    "server/assert"
    "server/tbaHandler"
    "server/utils"
)

type Team struct {
    TbaId         string
    Name          string
    AllianceScore int
}

func (t *Team) String() string {
    return fmt.Sprintf("Team: {\n TbaId: %s\n Name: %s\n AllianceScore: %d\n}", t.TbaId, t.Name, t.AllianceScore)
}

func GetTeam(database *sql.DB, tbaId string) *Team {
    query := `Select tbaId, name, COALESCE(allianceScore, 0) As allianceScore From Teams Where tbaId = $1;`
    assert := assert.CreateAssertWithContext("Get Team")
    assert.AddContext("TbaId", tbaId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    team := Team{}
    err = stmt.QueryRow(tbaId).Scan(&team.TbaId, &team.Name, &team.AllianceScore)
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

func UpdateTeamAllianceScore(database *sql.DB, tbaId string, allianceScore int16) {
    query := `Update Teams Set allianceScore = $1 where tbaId = $2;`
    assert := assert.CreateAssertWithContext("Update Team Ranking Score")
    assert.AddContext("Tba Id", tbaId)
    assert.AddContext("Ranking Score", allianceScore)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(allianceScore, tbaId)
    assert.NoError(err, "Failed to associate team")
}

func ValidPick(database *sql.DB, handler *tbaHandler.TbaHandler, tbaId string, draftId int) bool {
    if tbaId == "" {
        return false
    }

    picked := HasBeenPicked(database, draftId, tbaId)

    events := handler.MakeEventListReq(tbaId)
    draftEvents := utils.Events()

    validEvent := false
    //Looping here should always be faster because of the small lists
    fmt.Println(events)
    fmt.Println(draftEvents)
    for _, event := range events {
        for _, draftEvent := range draftEvents {
            if event == draftEvent {
                validEvent = true
                break
            }
        }

        if validEvent {
            break
        }
    }

    fmt.Printf("Picked: %t - Valid Event: %t\n", picked, validEvent)
    return !picked && validEvent
}

// Keys are the string that represents display name and the value is the score
// for that display name
// Display names: Qual Score, Playoff Score, Ranking Score, Einstein Score, Total Score
func GetScore(database *sql.DB, tbaId string) map[string]int {
    query := `Select
                t.AllianceScore
            From Teams t
            Where t.TbaId = $1`

    assert := assert.CreateAssertWithContext("Get Score")
    assert.AddContext("TbaId", tbaId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")

    var allianceScore int
    err = stmt.QueryRow(tbaId).Scan(&allianceScore)
    if err != nil {
        return nil
    }

    query = `Select
                Case When mt.match_tbaId Like '%_qm%' Then 'Qual Score'
                     When mt.match_tbaId Like '%cmptx%' Then 'Einstein Score'
                     Else 'Playoff Score' End As DisplayName,
                Sum(Case When mt.Alliance = 'Red' then m.redscore When mt.Alliance = 'Blue' Then m.bluescore Else 0 End) As Score
             From Matches_Teams mt
             Inner Join Matches m On mt.Match_tbaId = m.tbaId
             Where mt.Team_TbaId = $1
             Group By mt.Team_TbaId, Case When mt.match_tbaId Like '%_qm%' Then 'Qual Score'
                     When mt.match_tbaId Like '%cmptx%' Then 'Einstein Score'
                     Else 'Playoff Score' End
             Order By mt.Team_TbaId`

    stmt, err = database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")

    var displayName string
    var matchScore int
    rows, err := stmt.Query(tbaId)
    if err != nil {
        return nil
    }

    scores := make(map[string]int)
    total := 0
    for rows.Next() {
        rows.Scan(&displayName, &matchScore)
        total += matchScore
        scores[displayName] = matchScore
    }

    scores["Alliance Score"] = allianceScore
    scores["Total Score"] = total + allianceScore

    return scores
}
