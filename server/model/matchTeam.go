package model

type MatchTeam struct {
    TeamTbaId string
    MatchTbaId string
    Alliance string
    IsDqed bool
}

func AssocateTeam(matchTbaId string, teamTbaId string, alliance string, isDqed bool) error {
    return nil
}

func GetTeamForMatch(matchTbaId string, teamTbaId string) (error, MatchTeam) {
    return nil, MatchTeam{}
}
