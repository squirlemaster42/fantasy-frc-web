package scorer

import (
	"database/sql"
	"server/swagger"
	"log/slog"
	"regexp"
	"server/assert"
	"server/model"
	"server/tbaHandler"
	"server/utils"
	"strconv"
	"strings"
)

var RESCORE_INTERATION_COUNT = 72

type Scorer struct {
    tbaHandler       *tbaHandler.TbaHandler
    database         *sql.DB
    scoringIteration int
}

func NewScorer(tbaHandler *tbaHandler.TbaHandler, database *sql.DB) *Scorer {
    return &Scorer{
        tbaHandler:       tbaHandler,
        database:         database,
        scoringIteration: 0,
    }
}

func (s *Scorer) shouldScoreMatch(matchId string) bool {
    return s.scoringIteration%RESCORE_INTERATION_COUNT == 0 || !model.GetMatch(s.database, matchId).Played
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
    redScore := 0
    blueScore := 0

    slog.Info("Scoring qual match", "Match", match.Key, "Winning Alliance", match.WinningAlliance)
    if match.WinningAlliance == "red" {
        redScore += 6
    } else if match.WinningAlliance == "blue" {
        blueScore += 6
    }

    if match.ScoreBreakdown.Red.AutoBonusAchieved{
        redScore += 2
        slog.Info("Red Auto Bonus Achieved", "Score", redScore)
    }

    if match.ScoreBreakdown.Red.BargeBonusAchieved{
        redScore += 2
        slog.Info("Red Barge Bonus Achieved", "Score", redScore)
    }

    if match.ScoreBreakdown.Red.CoralBonusAchieved{
        redScore += 2
        slog.Info("Red Coral Bonus Achieved", "Score", redScore)
    }

    if match.ScoreBreakdown.Blue.AutoBonusAchieved{
        blueScore += 2
        slog.Info("Blue Auto Bonus Achieved", "Score", blueScore)
    }

    if match.ScoreBreakdown.Blue.BargeBonusAchieved{
        blueScore += 2
        slog.Info("Blue Barge Bonus Achieved", "Score", blueScore)
    }

    if match.ScoreBreakdown.Blue.CoralBonusAchieved{
        blueScore += 2
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

// RedScore, BlueScore
func getPlayoffMatchScore(match swagger.Match) (int, int) {
    redScore := 0
    blueScore := 0

    if match.CompLevel == "f" {
        if match.EventKey == einstein() {
            if match.WinningAlliance == "red" {
                redScore += 36
            } else if match.WinningAlliance == "blue" {
                blueScore += 36
            }
        } else {
            if match.WinningAlliance == "red" {
                redScore += 18
            } else if match.WinningAlliance == "blue" {
                blueScore += 18
            }
        }
    } else if match.CompLevel == "sf" {
        if getLowerBracketMatchIds()[match.SetNumber] {
            //Lower Bracket
            if match.EventKey == einstein() {
                if match.WinningAlliance == "red" {
                    redScore += 18
                } else if match.WinningAlliance == "blue" {
                    blueScore += 18
                }
            } else {
                if match.WinningAlliance == "red" {
                    redScore += 9
                } else if match.WinningAlliance == "blue" {
                    blueScore += 9
                }
            }
        } else if getUpperBracketMatchIds()[match.SetNumber] {
            //Upper Bracket
            if match.EventKey == einstein() {
                if match.WinningAlliance == "red" {
                    redScore += 30
                } else if match.WinningAlliance == "blue" {
                    blueScore += 30
                }
            } else {
                if match.WinningAlliance == "red" {
                    redScore += 15
                } else if match.WinningAlliance == "blue" {
                    blueScore += 15
                }
            }
        }
    }

    return redScore, blueScore
}

func (s *Scorer) getTeamRankingScore(team string) int {
    event := s.getChampEventForTeam(team)
    if event == "" {
        return 0
    }
    status := s.tbaHandler.MakeTeamEventStatusRequest(team, event)
    slog.Info("Getting ranking score", "Team", team, "Rank", status.Qual.Ranking.Rank)
    score := max((25-status.Qual.Ranking.Rank)*2, 0)
    return int(score)
}

func einstein() string {
    return "2025cmptx"
}

func (s *Scorer) getChampEventForTeam(teamId string) string {
    //Get list of teams events from tba
    //Check which event is in the list of champ events
    //We are going to ignore Einstein here since we just use this to determine the ranking score
    //which does not apply to Einstein
    slog.Info("Getting Events For Team", "Team", teamId)
    eventsList := s.tbaHandler.MakeEventListReq(strings.TrimSpace(teamId))
    //Even though this is O(e*f), where e is the number of events the team played during the season and f is
    //the number of champs field, both will be small so this is probably faster than a hashset
    for _, event := range eventsList {
        for _, champEvent := range utils.Events() {
            if event == champEvent {
                return event
            }
        }
    }
    return ""
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
        if s.compareMatchOrder(left[i], right[j]) {
            result = append(result, left[i])
            i++
        } else {
            result = append(result, right[j])
            j++
        }
    }

    for _, elem := range left[i:] {
        result = append(result, elem)
    }

    for _, elem := range right[j:] {
        result = append(result, elem)
    }

    return result
}

func matchPrecidence() map[string]int {
    return map[string]int{
        "qm": 0,
        "qf": 1,
        "sf": 2,
        "f":  3,
    }
}

// Return true if matchA comes before matchB
func (s *Scorer) compareMatchOrder(matchA string, matchB string) bool {
    assert := assert.CreateAssertWithContext("Compare Match Order")
    assert.AddContext("Match A", matchA)
    assert.AddContext("Match B", matchB)
    matchALevel := getMatchLevel(matchA)
    matchBLevel := getMatchLevel(matchB)
    assert.AddContext("Match A Level", matchALevel)
    assert.AddContext("Match B Level", matchBLevel)
    aPrecidence, ok := matchPrecidence()[matchALevel]
    assert.RunAssert(ok, "Match Precidence Was Not Found")
    bPrecidence, ok := matchPrecidence()[matchBLevel]
    assert.RunAssert(ok, "Match Precidence Was Not Found")

    if aPrecidence != bPrecidence {
        return aPrecidence < bPrecidence
    }

    assert.RunAssert(matchALevel == matchBLevel, "Match levels are not the same")

    if matchALevel == "qm" {
        splitMatchA := strings.Split(matchA, "_")
        splitMatchB := strings.Split(matchB, "_")
        assert.RunAssert(len(splitMatchA) == 2, "Match A string was invalid")
        assert.RunAssert(len(splitMatchB) == 2, "Match B string was invalid")
        matchANum, err := strconv.Atoi(splitMatchA[1][2:])
        assert.NoError(err, "Match A num Atoi failed")
        matchBNum, err := strconv.Atoi(splitMatchB[1][2:])
        assert.NoError(err, "Match B num Atoi failed")
        return matchANum < matchBNum
    }

    if matchALevel == "f" {
        splitMatchA := strings.Split(matchA, "_")
        splitMatchB := strings.Split(matchB, "_")
        assert.RunAssert(len(splitMatchA) == 2, "Match A string was invalid")
        assert.RunAssert(len(splitMatchB) == 2, "Match B string was invalid")
        splitMatchA = strings.Split(splitMatchA[1][1:], "m")
        splitMatchB = strings.Split(splitMatchB[1][1:], "m")
        assert.RunAssert(len(splitMatchA) == 2, "Match A string was invalid")
        assert.RunAssert(len(splitMatchB) == 2, "Match B string was invalid")
        matchANum, err := strconv.Atoi(splitMatchA[0])
        assert.NoError(err, "Match A num Atoi failed")
        matchBNum, err := strconv.Atoi(splitMatchB[0])
        assert.NoError(err, "Match B num Atoi failed")

        if matchANum != matchBNum {
            return matchANum < matchBNum
        }

        assert.RunAssert(matchANum == matchBNum, "Match nums are the same but shouldn't be")

        matchANum, err = strconv.Atoi(splitMatchA[1])
        assert.NoError(err, "Match A num Atoi failed")
        matchBNum, err = strconv.Atoi(splitMatchB[1])
        assert.NoError(err, "Match B num Atoi failed")

        return matchANum < matchBNum
    }

    if matchALevel == "sf" {
        splitMatchA := strings.Split(matchA, "_")
        splitMatchB := strings.Split(matchB, "_")
        assert.RunAssert(len(splitMatchA) == 2, "Match A string was invalid")
        assert.RunAssert(len(splitMatchB) == 2, "Match B string was invalid")
        splitMatchA = strings.Split(splitMatchA[1][2:], "m")
        splitMatchB = strings.Split(splitMatchB[1][2:], "m")
        assert.RunAssert(len(splitMatchA) == 2, "Match A string was invalid")
        assert.RunAssert(len(splitMatchB) == 2, "Match B string was invalid")
        matchANum, err := strconv.Atoi(splitMatchA[0])
        assert.NoError(err, "Match A num Atoi failed")
        matchBNum, err := strconv.Atoi(splitMatchB[0])
        assert.NoError(err, "Match B num Atoi failed")

        if matchANum != matchBNum {
            return matchANum < matchBNum
        }

        assert.RunAssert(matchANum == matchBNum, "Match nums are the same but shouldn't be")

        matchANum, err = strconv.Atoi(splitMatchA[1])
        assert.NoError(err, "Match A num Atoi failed")
        matchBNum, err = strconv.Atoi(splitMatchB[1])
        assert.NoError(err, "Match B num Atoi failed")

        return matchANum < matchBNum
    }

    assert.RunAssert(1 == 0, "Unknown match type found")
    return false // This is unreachable
}

func getMatchLevel(matchKey string) string {
    assert := assert.CreateAssertWithContext("Get Match Level")
    assert.AddContext("Match Key", matchKey)
    pattern := regexp.MustCompile("_[a-z]+")
    match := pattern.FindString(matchKey)[1:]
    assert.AddContext("Match", match)
    assert.RunAssert(len(match) == 2 || len(match) == 1, "Match did not return string of expected length")
    return match
}

func isDqed(team string, dqedTeams []string) bool {
    for _, dqed := range dqedTeams {
        if team == dqed {
            return true
        }
    }
    return false
}

var ALLIANCE_SCORES = map[int][]int16 {
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
        scores[team] = scoreArr[i]
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

func (s *Scorer) AddMatchToScore() {

}

func (s *Scorer) scoringRunner() {
    for {
        //We need to rewrite this to support websockets
        //We need to score matches that have been played when the scorer starts up and then
        //score matches that come in over the websocket

        //TODO Skip scoring if we are not on an event day
        //Get a list of matches to score and
        //Sort matches by id (they are almost sorted, but we need to move finals matches to the end (no they are not, I dont see any corrilation))
        slog.Info("Starting scoring iteration")
        allTeams := make(map[string]bool)
        matches := make(map[string][]string)
        for _, event := range utils.Events() {
            matches[event] = s.sortMatchesByPlayOrder(s.tbaHandler.MakeEventMatchKeysRequest(event))
        }

        //Score matches until we hit one that has not been played
        var scoringQueue []string
        currentMatch := make(map[string]int)
        for event := range matches {
            currentMatch[event] = 1
            scoringQueue = append(scoringQueue, matches[event][0])
        }

        for {
            match := scoringQueue[0]
            scoringQueue = scoringQueue[1:]

            dbMatchPtr := model.GetMatch(s.database, match)

            if dbMatchPtr == nil {
                model.AddMatch(s.database, match)
                dbMatchPtr = &model.Match{
                    TbaId:        match,
                    BlueAlliance: []string{},
                    RedAlliance:  []string{},
                    DqedTeams:    []string{},
                    Played:       false,
                }
            }

            dbMatch := *dbMatchPtr
            slog.Info("Scoring match", "Match", dbMatchPtr.String())

            scored := false
            if !dbMatch.Played || s.scoringIteration % RESCORE_INTERATION_COUNT == 0 {
                slog.Info("Match was not played or rescoring all matches")
                tbaMatch := s.tbaHandler.MakeMatchReq(dbMatch.TbaId)
                dbMatch, scored = s.scoreMatch(tbaMatch, s.scoringIteration % RESCORE_INTERATION_COUNT == 0)
            }

            event := strings.Split(match, "_")[0]
            currentMatch[event] = currentMatch[event] + 1
            if len(matches[event]) > 0 {
                scoringQueue = append(scoringQueue, matches[event][0])
                matches[event] = matches[event][1:]
            }

            if scored {
                slog.Info("Updating Match Scores", "Match", dbMatch.String())
                model.UpdateScore(s.database, dbMatch.TbaId, dbMatch.RedScore, dbMatch.BlueScore)
                for _, team := range dbMatch.BlueAlliance {
                    allTeams[team] = true
                    model.AssocateTeam(s.database, dbMatch.TbaId, team, "Blue", isDqed(team, dbMatch.DqedTeams))
                }
                for _, team := range dbMatch.RedAlliance {
                    allTeams[team] = true
                    model.AssocateTeam(s.database, dbMatch.TbaId, team, "Red", isDqed(team, dbMatch.DqedTeams))
                }
            }

            if len(scoringQueue) == 0 {
                break
            }
        }

        //Update alliance selection scores
        //TODO this should only run after all quals are complete, there is probably some webhook stuff we can do
        for _, event := range utils.Events() {
            alliances := s.tbaHandler.MakeEliminationAllianceRequest(event)
            for _, alliance := range alliances {
                scores := s.GetAllianceSelectionScore(alliance)
                for team, score := range scores {
                    slog.Info("Update alliance score for team", "Team", team, "Score", score)
                    model.UpdateTeamAllianceScore(s.database, team, score)
                }
            }
        }
    }
}
