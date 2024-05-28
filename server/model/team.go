package model

type Team struct {
    TbaId string
    Name string
    RankingScore int
}

func GetTeam(tbaId string) (error, Team) {
    return nil, Team{}
}

func UpsertTeam(team Team) error {
    return nil
}

