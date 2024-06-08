package model

import "database/sql"

type Team struct {
    TbaId string
    Name string
    RankingScore int
}

func GetTeam(database *sql.DB, tbaId string) Team {
    return Team{}
}

func CreateTeam(database *sql.DB, team Team) {

}

func UpdateTeamRankingScore(datbase *sql.DB, tbaId string, rankingScore int) {

}
