package scorer

import (
	"database/sql"
	"fmt"
	"regexp"
	"server/assert"
	"server/model"
	"server/tbaHandler"
	"strconv"
	"strings"
	"time"
)

var RESCORE_INTERATION_COUNT = 72

type Scorer struct {
	tbaHandler *tbaHandler.TbaHandler
    database *sql.DB
    scoringIteration int
}

func NewScorer(tbaHandler *tbaHandler.TbaHandler, database *sql.DB) *Scorer {
	return &Scorer{
		tbaHandler: tbaHandler,
        database: database,
        scoringIteration:  0,
	}
}

func (s *Scorer) shouldScoreMatch(matchId string) bool {
    return s.scoringIteration % RESCORE_INTERATION_COUNT == 0 || !model.GetMatch(s.database, matchId).Played
}

func playoffMatchCompLevels() map[string]bool {
    return map[string]bool{
        "f": true,
        "sf": true,
        "qf": true,
    }
}

//TODO This should take in a match and not a match id
//Match, dqed teams
func (s *Scorer) scoreMatch(match tbaHandler.Match) model.Match {
    scoredMatch := model.Match{
        TbaId: match.Key,
        Played: match.PostResultTime > 0,
        RedScore: 0,
        BlueScore: 0,
    }

    if !scoredMatch.Played {
        return scoredMatch
    }

    if match.CompLevel == "qm" {
        scoredMatch.RedScore, scoredMatch.BlueScore = getQualMatchScore(match)
    } else if playoffMatchCompLevels()[match.CompLevel] {
        scoredMatch.RedScore, scoredMatch.BlueScore = getPlayoffMatchScore(match)
    }
    scoredMatch.RedAlliance = match.Alliances.Red.TeamKeys
    scoredMatch.BlueAlliance = match.Alliances.Blue.TeamKeys
    scoredMatch.DqedTeams = append(match.Alliances.Blue.DqTeamKeys, match.Alliances.Blue.SurrogateTeamKeys...)
    return scoredMatch
}

func getQualMatchScore(match tbaHandler.Match) (int, int) {
    redScore := 0
    blueScore := 0

    if match.WinningAlliance == "red" {
        redScore += 4
    } else if match.WinningAlliance == "blue" {
        blueScore += 4
    }

    if match.ScoreBreakdown.Red.MelodyBonusAchieved {
        redScore += 2
    }

    if match.ScoreBreakdown.Red.EnsembleBonusAchieved {
        redScore += 2
    }

    if match.ScoreBreakdown.Blue.MelodyBonusAchieved {
        blueScore += 2
    }

    if match.ScoreBreakdown.Blue.EnsembleBonusAchieved {
        blueScore += 2
    }

    return redScore, blueScore
}

func getUpperBracketMatchIds() map[int]bool {
    return map[int]bool{
        1: true,
        2: true,
        3: true,
        4: true,
        7: true,
        8: true,
        11: true,
    }
}

func getLowerBracketMatchIds() map[int]bool {
    return map[int]bool{
        5: true,
        6: true,
        9: true,
        10: true,
        12: true,
        13: true,
    }
}

//RedScore, BlueScore
func getPlayoffMatchScore(match tbaHandler.Match) (int, int) {
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
    status := s.tbaHandler.MakeTeamEventStatusRequest(team, event)
    score := max((25 - status.Qual.Ranking.Rank) * 2, 0)
    return score
}

func einstein() string {
    return "2024cmptx"
}

func events() []string {
    //TODO can we do this programatically?
    return []string{
        "2024new",
        "2024mil",
        "2024joh",
        "2024hop",
        "2024gal",
        "2024dal",
        "2024cur",
        "2024arc",
    }
}

func (s *Scorer) getChampEventForTeam(teamId string) string {
	//Get list of teams events from tba
	//Check which event is in the list of champ events
	//We are going to ignore Einstein here since we just use this to determin the ranking score
	//which does not apply to Einstein
	eventsList := s.tbaHandler.MakeEventListReq(strings.TrimSpace(teamId))
	//Even though this is O(e*f), where e is the number of events the team played during the season and f is
	//the number of champs field, both will be small so this is probably faster than a hashset
	for _, event := range eventsList {
		for _, champEvent := range events() {
			if event == champEvent {
				return event
			}
		}
	}
    panic(fmt.Sprintf("Champ event not found for team %s", teamId))
}

//Matches are almost sorted
//We need to sort it so that matches so qm -> qf -> sf -> f and then sort by match id
func sortMatchesByPlayOrder(matches []string) []string {
    if len(matches) <= 1 {
        return matches
    }

    mid := len(matches) / 2
    left := matches[:mid]
    right := matches[mid:]

    sortedLeft := sortMatchesByPlayOrder(left)
    sortedRight := sortMatchesByPlayOrder(right)

    return merge(sortedLeft, sortedRight)
}

func merge(left []string, right []string) []string {
    var result []string
    i := 0
    j := 0

    for i < len(left) && j < len(right) {
        if compareMatchOrder(left[i], right[j]) {
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
        "f": 3,
    }
}

//Return true if matchA comes before matchB
func compareMatchOrder(matchA string, matchB string) bool {
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

    panic("Unhandled match type")
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

func (s *Scorer) RunScorer() {
	//This function will run on its own routine
	//We will first update our list of teams with all of the teams at all of the events in getChampEvents
	//We do not need to account for Einstein since all of the teams on Einstein will have been in a previous champ event
	//We then score each match that this team has played and has not already been scored
	//We choose the matches to score from the picks table
	//Periodically we will want to rescore everything to ensure that we account for replays
	//We will will have this process run every five minutes and we will rescore all matches every 6 hours
    //In this iteration we also update the valid teams

	go func(s *Scorer) {
		for {
            //TODO Need to add something that allows for rescoring all matches on a set interval
            //Get a list of matches to score and
            //Sort matches by id (they are almost sorted, but we need to move finals matches to the end (no they are not, I dont see any corrilation))
            matches := make(map[string][]string)
            for _, event := range events() {
                matches[event] = sortMatchesByPlayOrder(s.tbaHandler.MakeEventMatchKeysRequest(event))
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

                dbMatch := *model.GetMatch(s.database, match)

                if !dbMatch.Played || s.scoringIteration % RESCORE_INTERATION_COUNT == 0 {
                    match := s.tbaHandler.MakeMatchReq(dbMatch.TbaId)
                    dbMatch = s.scoreMatch(match)
                }

                if dbMatch.Played {
                    event := strings.Split(match, "_")[1]
                    currentMatch[event] = currentMatch[event] + 1
                    scoringQueue = append(scoringQueue, matches[event][0])
                }

                model.UpdateScore(s.database, dbMatch.TbaId, dbMatch.RedScore, dbMatch.BlueScore)
                for _, team := range dbMatch.BlueAlliance {
                    model.AssocateTeam(s.database, dbMatch.TbaId, team, "Blue", isDqed(team, dbMatch.DqedTeams))
                }
                for _, team := range dbMatch.RedAlliance {
                    model.AssocateTeam(s.database, dbMatch.TbaId, team, "Red", isDqed(team, dbMatch.DqedTeams))
                }

                if len(scoringQueue) == 0 {
                    break
                }
            }

            //Update ranking scores
            //Get all picked teams
            picks := model.GetAllPicks(s.database)

            //Update the ranking scores for all picked teams
            for _, pick := range picks {
                model.UpdateTeamRankingScore(s.database, pick, s.getTeamRankingScore(pick))
            }

			s.scoringIteration++
			fmt.Println("Finished scoring iteration")
			time.Sleep(5 * time.Minute)
		}
	}(s)
}