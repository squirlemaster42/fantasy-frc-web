package model

import (
	"fmt"
	"html"
	"log"
	"server/database"
)

type Team struct {
    tbaId string
    name string
    rankingScore int
    validPick bool
}

func upsertTeam(team *Team, dbDriver *database.DatabaseDriver) {
	query := fmt.Sprintf(`INSERT INTO Teams (tbaId, name, rankingScore, validPick)
    VALUES ('%s', '%s', %d, %t)
    ON CONFLICT(tbaId)
    DO UPDATE SET
    name = EXCLUDED.name, rankingScore = EXCLUDED.rankingScore;`, team.tbaId, html.EscapeString(team.name), team.rankingScore, team.validPick)
    dbDriver.RunExec(query)
}

func getTeam(tbaId string, dbDriver *database.DatabaseDriver) *Team {
    query := `SELECT tbaId, name, rankingScore, validPick FROM Teams WHERE tbaId = $1`
    stmt, err := dbDriver.Connection.Prepare(query)

    if err != nil {
        log.Println(err)
        return &Team{}
    }

    var team Team

    stmt.QueryRow(tbaId).Scan(&team.tbaId, &team.name, &team.rankingScore, &team.validPick)

    return &team
}

func UpdateTeamValidity(tbaId string, isValid bool, dbDriver *database.DatabaseDriver) {
    query := fmt.Sprintf(`INSERT INTO Teams (tbaId, validPick)
    VALUES ('%s', %t)
    ON CONFLICT(tbaId)
    DO UPDATE SET
    validPick = EXCLUDED.validPick;`, tbaId, isValid)
    dbDriver.RunExec(query)

}

func GetTeamValidity(dbDriver *database.DatabaseDriver) map[string]bool {
    teamValidity := make(map[string]bool)

    query := `Select tbaId, validPick From Teams`
    stmt, err := dbDriver.Connection.Prepare(query)

    if err != nil {
        log.Println(err)
    }

    rows, err := stmt.Query()
    defer rows.Close()

    if err != nil {
        log.Println(err)
    }

    for rows.Next() {
        var tbaId string
        var isValid bool

        err = rows.Scan(&tbaId, &isValid)

        if err != nil {
            log.Println(err)
        }

        teamValidity[tbaId] = isValid
    }

    return teamValidity
}
