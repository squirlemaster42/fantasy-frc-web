package scorer

import (
	"context"
	"database/sql"
	"server/assert"
	"server/log"
	"server/model"
	"server/swagger"
	"server/tbaHandler"
	"server/utils"
	"strconv"
	"strings"
)

type Scorer struct {
	tbaHandler       *tbaHandler.TbaHandler
	database         *sql.DB
	scoringIteration int
	queue            *MatchQueue
}

func NewScorer(tbaHandler *tbaHandler.TbaHandler, database *sql.DB) *Scorer {
	return &Scorer{
		tbaHandler:       tbaHandler,
		database:         database,
		scoringIteration: 0,
		queue:            NewMatchQueue(),
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
		scoredMatch.RedScore, scoredMatch.BlueScore = getQualMatchScore(context.TODO(), match)
	} else if playoffMatchCompLevels()[match.CompLevel] {
		scoredMatch.RedScore, scoredMatch.BlueScore = getPlayoffMatchScore(context.TODO(), match)
	}
	scoredMatch.RedAlliance = match.Alliances.Red.TeamKeys
	scoredMatch.BlueAlliance = match.Alliances.Blue.TeamKeys
	scoredMatch.DqedTeams = append(match.Alliances.Blue.DqTeamKeys, match.Alliances.Blue.SurrogateTeamKeys...)

	log.DebugNoContext("Scored Match", "Match", scoredMatch.String())

	return scoredMatch, true
}

func getQualMatchScore(context context.Context, match swagger.Match) (int, int) {
	log.Debug(context, "Scoring qual match", "Match", match.Key, "Winning Alliance", match.WinningAlliance)

	redScore, blueScore := getWinningAllianceScores(match, 3)

	if match.ScoreBreakdown == nil {
		return redScore, blueScore
	}

	if match.ScoreBreakdown.Red != nil && match.ScoreBreakdown.Red.EnergizedAchieved {
		redScore += 1
		log.Debug(context, "Red Energized Achieved", "Score", redScore)
	}

	if match.ScoreBreakdown.Red != nil && match.ScoreBreakdown.Red.SuperchargedAchieved {
		redScore += 1
		log.Debug(context, "Red Supercharded Bonus Achieved", "Score", redScore)
	}

	if match.ScoreBreakdown.Red != nil && match.ScoreBreakdown.Red.TraversalAchieved {
		redScore += 2
		log.Debug(context, "Red Traversal Bonus Achieved", "Score", redScore)
	}

	if match.ScoreBreakdown.Blue != nil && match.ScoreBreakdown.Blue.EnergizedAchieved {
		blueScore += 1
		log.Debug(context, "Blue Energized Bonus Achieved", "Score", blueScore)
	}

	if match.ScoreBreakdown.Blue != nil && match.ScoreBreakdown.Blue.SuperchargedAchieved {
		blueScore += 1
		log.Debug(context, "Blue Supercharged Bonus Achieved", "Score", blueScore)
	}

	if match.ScoreBreakdown.Blue != nil && match.ScoreBreakdown.Blue.TraversalAchieved {
		blueScore += 2
		log.Debug(context, "Blue Traversal Bonus Achieved", "Score", blueScore)
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

// Red score, blue score
func getWinningAllianceScores(match swagger.Match, winningPoints int) (int, int) {
	redScore := 0
	blueScore := 0

	switch match.WinningAlliance {
	case "red":
		redScore = winningPoints
	case "blue":
		blueScore = winningPoints
	default:
		log.DebugNoContext("No winning alliance found", "Match", match.Key, "Winning Alliance", match.WinningAlliance)
	}

	return redScore, blueScore
}

// RedScore, BlueScore
func getPlayoffMatchScore(context context.Context, match swagger.Match) (int, int) {
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
		log.Warn(context, "Attempted to get playoff score for non playoff match", "Match", match.Key, "Comp Level", match.CompLevel)
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
		val, err := utils.CompareMatchOrder(context.TODO(), left[i], right[j])
		if err != nil {
			log.Warn(context.TODO(), "Failed to compare match order", "Error", err)
		}

		if val {
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
		log.Error(context.TODO(), "Alliance name was not in an expected format", "Name", alliance.Name)
	}

	allianceNum, err := strconv.Atoi(splitAllianceName[1])
	if err != nil {
		log.Error(context.TODO(), "Got bad TBA data when computing alliance selection scores", "Name", alliance.Name)
	}

	if allianceNum > 8 || allianceNum < 1 {
		log.Error(context.TODO(), "Unsupported alliance number", "Alliance", alliance.Name)
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
	log.Debug(context.TODO(), "Updating Match Scores", "Match", dbMatch.String())
	model.UpdateScore(context.TODO(), s.database, dbMatch.TbaId, dbMatch.RedScore, dbMatch.BlueScore)
	for _, team := range dbMatch.BlueAlliance {
		model.AssocateTeam(context.TODO(), s.database, dbMatch.TbaId, team, "Blue", isDqed(team, dbMatch.DqedTeams))
	}
	for _, team := range dbMatch.RedAlliance {
		model.AssocateTeam(context.TODO(), s.database, dbMatch.TbaId, team, "Red", isDqed(team, dbMatch.DqedTeams))
	}
}

func (s *Scorer) scoringRunner() {
	for _, event := range utils.Events() {
		for _, match := range s.tbaHandler.MakeEventMatchKeysRequest(context.TODO(), event) {
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
		s.ScoreAllianceSelection(context.TODO(), event)
	}

	for {
		log.DebugNoContext("Starting scoring iteration")

		match := s.getNextMatchToScore()

		log.DebugNoContext("Checking if we need to get match data", "Match", match)
		if match.MatchNumber == 0 {
			log.DebugNoContext("Loading match data", "Match", match.Key)
			match = s.tbaHandler.MakeMatchReq(context.TODO(), match.Key)
		}

		log.DebugNoContext("Starting scoring run", "Match", match.Key)
		dbMatchPtr := model.GetMatch(context.TODO(), s.database, match.Key)

		if dbMatchPtr == nil {
			model.AddMatch(context.TODO(), s.database, match.Key)
			dbMatchPtr = &model.Match{
				TbaId:        match.Key,
				BlueAlliance: []string{},
				RedAlliance:  []string{},
				DqedTeams:    []string{},
				Played:       false,
			}
		}
		log.DebugNoContext("Scoring match", "Match", dbMatchPtr.String())

		dbMatch, _ := s.scoreMatch(match, true)
		s.updateMatchInDB(dbMatch)
	}
}

func (s *Scorer) ScoreAllianceSelection(context context.Context, event string) {
	alliances := s.tbaHandler.MakeEliminationAllianceRequest(context, event)
	log.Info(context, "Made alliance selection request", "Alliance length", len(alliances))
	for _, alliance := range alliances {
		scores := s.GetAllianceSelectionScore(alliance)
		for team, score := range scores {
			log.Info(context, "Update alliance score for team", "Team", team, "Score", score)
			model.UpdateTeamAllianceScore(context, s.database, team, score)
		}
	}
}
