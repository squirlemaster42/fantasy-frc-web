package scorer

import (
	"database/sql"
	"log/slog"
	"server/assert"
	"server/model"
	"server/swagger"
	"server/tbaHandler"
	"server/utils"
	"strconv"
	"strings"
)

type Scorer struct {
    tbaHandler *tbaHandler.TbaHandler
    database *sql.DB
    scoringIteration int
    queue *MatchQueue
}

func NewScorer(tbaHandler *tbaHandler.TbaHandler, database *sql.DB) *Scorer {
    return &Scorer{
        tbaHandler: tbaHandler,
        database: database,
        scoringIteration: 0,
        queue: NewMatchQueue(),
    }
}

func playoffMatchCompLevels() map[string]bool {
    return map[string]bool{
        "f":  true,
        "sf": true,
        "qf": true,
    }
}

// Match, dqed teams
func (s *Scorer) scoreMatch(match swagger.Match, rescore bool) (model.Match, bool) {
    scoredMatch := model.Match{
        TbaId:     match.Key,
        Played:    match.PostResultTime > 0,
        RedScore:  0,
        BlueScore: 0,
    }

    if !scoredMatch.Played && !rescore {
        return scoredMatch, false
    }

    if match.CompLevel == "qm" {
        scoredMatch.RedScore, scoredMatch.BlueScore = getQualMatchScore(match)
    } else if playoffMatchCompLevels()[match.CompLevel] {
        scoredMatch.RedScore, scoredMatch.BlueScore = getPlayoffMatchScore(match)
    }
    scoredMatch.RedAlliance = match.Alliances.Red.TeamKeys
    scoredMatch.BlueAlliance = match.Alliances.Blue.TeamKeys
    scoredMatch.DqedTeams = append(match.Alliances.Blue.DqTeamKeys, match.Alliances.Blue.SurrogateTeamKeys...)

    slog.Info("Scored Match", "Match", scoredMatch.String())

    return scoredMatch, true
}

func getQualMatchScore(match swagger.Match) (int, int) {
    slog.Info("Scoring qual match", "Match", match.Key, "Winning Alliance", match.WinningAlliance)

    redScore, blueScore := getWinningAllianceScores(match, 3)

    if match.ScoreBreakdown == nil {
        return redScore, blueScore
    }

    if match.ScoreBreakdown.Red != nil && match.ScoreBreakdown.Red.AutoBonusAchieved {
        redScore += 1
        slog.Info("Red Auto Bonus Achieved", "Score", redScore)
    }

    if match.ScoreBreakdown.Red != nil && match.ScoreBreakdown.Red.BargeBonusAchieved {
        redScore += 1
        slog.Info("Red Barge Bonus Achieved", "Score", redScore)
    }

    if match.ScoreBreakdown.Red != nil && match.ScoreBreakdown.Red.CoralBonusAchieved {
        redScore += 1
        slog.Info("Red Coral Bonus Achieved", "Score", redScore)
    }

    if match.ScoreBreakdown.Blue != nil && match.ScoreBreakdown.Blue.AutoBonusAchieved {
        blueScore += 1
        slog.Info("Blue Auto Bonus Achieved", "Score", blueScore)
    }

    if match.ScoreBreakdown.Blue != nil && match.ScoreBreakdown.Blue.BargeBonusAchieved {
        blueScore += 1
        slog.Info("Blue Barge Bonus Achieved", "Score", blueScore)
    }

    if match.ScoreBreakdown.Blue != nil && match.ScoreBreakdown.Blue.CoralBonusAchieved {
        blueScore += 1
        slog.Info("Blue Coral Bonus Achieved", "Score", blueScore)
    }

    return redScore, blueScore
}

func getUpperBracketMatchIds() map[int32]bool {
    return map[int32]bool{
        1:  true,
        2:  true,
        3:  true,
        4:  true,
        7:  true,
        8:  true,
        11: true,
    }
}

func getLowerBracketMatchIds() map[int32]bool {
    return map[int32]bool{
        5:  true,
        6:  true,
        9:  true,
        10: true,
        12: true,
        13: true,
    }
}

//Red score, blue score
func getWinningAllianceScores(match swagger.Match, winningPoints int) (int, int) {
    redScore := 0
    blueScore := 0

    switch match.WinningAlliance {
    case "red":
        redScore = winningPoints
    case "blue":
        blueScore = winningPoints
    default:
        slog.Info("No winning alliance found", "Match", match.Key, "Winning Alliance", match.WinningAlliance)
    }

    return redScore, blueScore
}

// RedScore, BlueScore
func getPlayoffMatchScore(match swagger.Match) (int, int) {
    var matchPoints int

    switch match.CompLevel {
    case "f":
        matchPoints = 18
    case "sf":
        if getLowerBracketMatchIds()[match.SetNumber] {
            matchPoints = 9
        } else if getUpperBracketMatchIds()[match.SetNumber] {
            matchPoints = 15
        }
    default:
        slog.Warn("Attempted to get playoff score for non playoff match", "Match", match.Key, "Comp Level", match.CompLevel)
    }

    if match.EventKey == utils.Einstein() {
        matchPoints *= 2
    }


    return getWinningAllianceScores(match, matchPoints)
}

// Matches are almost sorted
// We need to sort it so that matches so qm -> qf -> sf -> f and then sort by match id
func (s *Scorer) sortMatchesByPlayOrder(matches []string) []string {
    if len(matches) <= 1 {
        return matches
    }

    mid := len(matches) / 2
    left := matches[:mid]
    right := matches[mid:]

    sortedLeft := s.sortMatchesByPlayOrder(left)
    sortedRight := s.sortMatchesByPlayOrder(right)

    return s.merge(sortedLeft, sortedRight)
}

func (s *Scorer) merge(left []string, right []string) []string {
    var result []string
    i := 0
    j := 0

    for i < len(left) && j < len(right) {
        if utils.CompareMatchOrder(left[i], right[j]) {
            result = append(result, left[i])
            i++
        } else {
            result = append(result, right[j])
            j++
        }
    }

    result = append(result, left[i:]...)
    result = append(result, right[j:]...)

    return result
}

func isDqed(team string, dqedTeams []string) bool {
    for _, dqed := range dqedTeams {
        if team == dqed {
            return true
        }
    }
    return false
}

var ALLIANCE_SCORES = map[int][]int16{
    1: {32, 31, 9, 8},
    2: {30, 29, 10, 7},
    3: {28, 27, 11, 6},
    4: {26, 25, 12, 5},
    5: {24, 23, 13, 4},
    6: {22, 21, 14, 3},
    7: {20, 19, 15, 2},
    8: {18, 17, 16, 1},
}

func (s *Scorer) GetAllianceSelectionScore(alliance swagger.EliminationAlliance) map[string]int16 {
    assert.CreateAssertWithContext("Get Alliance Selection Score")
    scores := make(map[string]int16)

    splitAllianceName := strings.Split(alliance.Name, " ")
    if len(splitAllianceName) != 2 {
        slog.Error("Alliance name was not in an expected format", "Name", alliance.Name)
    }

    allianceNum, err := strconv.Atoi(splitAllianceName[1])
    if err != nil {
        slog.Error("Got bad TBA data when computing alliance selection scores", "Name", alliance.Name)
    }

    if allianceNum > 8 || allianceNum < 1 {
        slog.Error("Unsupported alliance number", "Alliance", alliance.Name)
    }

    scoreArr := ALLIANCE_SCORES[allianceNum]
    for i, team := range alliance.Picks {
        scores[team] = scoreArr[i] * 2
    }

    return scores
}

func (s *Scorer) RunScorer() {
    //This function will run on its own routine
    //We will first update our list of teams with all of the teams at all of the events in getChampEvents
    //We do not need to account for Einstein since all of the teams on Einstein will have been in a previous champ event
    //We then score each match that this team has played and has not already been scored
    //We choose the matches to score from the picks table
    //Periodically we will want to rescore everything to ensure that we account for replays
    //We will will have this process run every five minutes and we will rescore all matches every 6 hours
    //In this iteration we also update the valid teams

    go s.scoringRunner()
}

func (s *Scorer) AddMatchToScore(match swagger.Match) {
    s.queue.PushMatch(match)
}

func (s *Scorer) getNextMatchToScore() swagger.Match {
    return s.queue.PopMatch()
}

func (s *Scorer) updateMatchInDB(dbMatch model.Match) {
    slog.Info("Updating Match Scores", "Match", dbMatch.String())
    model.UpdateScore(s.database, dbMatch.TbaId, dbMatch.RedScore, dbMatch.BlueScore)
    for _, team := range dbMatch.BlueAlliance {
        model.AssocateTeam(s.database, dbMatch.TbaId, team, "Blue", isDqed(team, dbMatch.DqedTeams))
    }
    for _, team := range dbMatch.RedAlliance {
        model.AssocateTeam(s.database, dbMatch.TbaId, team, "Red", isDqed(team, dbMatch.DqedTeams))
    }
}

func (s *Scorer) scoringRunner() {
    for _, event := range utils.Events() {
        for _, match := range s.tbaHandler.MakeEventMatchKeysRequest(event) {
            s.AddMatchToScore(swagger.Match{
                Key: match,
            })
        }
    }

    //Update alliance selection scores
    for _, event := range utils.Events() {
        if event == utils.Einstein() {
            continue
        }
        s.ScoreAllianceSelection(event)
    }

    for {
        slog.Info("Starting scoring iteration")

        match := s.getNextMatchToScore()

        slog.Info("Checking if we need to get match data", "Match", match)
        if match.MatchNumber == 0 {
            slog.Info("Loading match data", "Match", match.Key)
            match = s.tbaHandler.MakeMatchReq(match.Key)
        }

        slog.Info("Starting scoring run", "Match", match.Key)
        dbMatchPtr := model.GetMatch(s.database, match.Key)

        if dbMatchPtr == nil {
            model.AddMatch(s.database, match.Key)
            dbMatchPtr = &model.Match{
                TbaId:        match.Key,
                BlueAlliance: []string{},
                RedAlliance:  []string{},
                DqedTeams:    []string{},
                Played:       false,
            }
        }
        slog.Info("Scoring match", "Match", dbMatchPtr.String())

        dbMatch, _ := s.scoreMatch(match, true)
        s.updateMatchInDB(dbMatch)
    }
}

func (s *Scorer) ScoreAllianceSelection(event string) {
    alliances := s.tbaHandler.MakeEliminationAllianceRequest(event)
    for _, alliance := range alliances {
        scores := s.GetAllianceSelectionScore(alliance)
        for team, score := range scores {
            slog.Info("Update alliance score for team", "Team", team, "Score", score)
            model.UpdateTeamAllianceScore(s.database, team, score)
        }
    }
}
