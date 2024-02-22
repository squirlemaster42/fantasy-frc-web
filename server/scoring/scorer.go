package scoring

type Scorer struct {
    tbaToken string
}

type Match struct {
    matchId string
    redAllianceScore int
    blueAllianceScore int
    redAllianceTeams []string
    blueAllianceTeams []string
    dqedTeams []string
}

type Team struct {
    teamId string
    matches []Match
}

type Player struct {

}

type Draft struct {

}

func NewScorer(tbaToken string) *Scorer {
    scorer := Scorer{tbaToken: tbaToken}
    return &scorer
}

func (s *Scorer) scoreMatch(matchId string) {

}

func (s *Scorer) getMatches(teamId string) {

}

func (s *Scorer) getTeams(playerId string) {

}

func (s *Scorer) getPlayers(draftId string) {

}

func (s *Scorer) getDrafts() {

}
