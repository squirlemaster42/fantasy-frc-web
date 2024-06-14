package scoring

import (
	"database/sql"
	"fmt"
	"regexp"
	"server/assert"
	"server/model"
	"strconv"
	"strings"
	"time"
)

var RESCORE_INTERATION_COUNT = 72

type Scorer struct {
	tbaHandler *TbaHandler
    database *sql.DB
    scoringIteration int
}

func NewScorer(tbaHandler *TbaHandler, database *sql.DB) *Scorer {
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

func (s *Scorer) scoreMatch(matchId string) model.Match {
    match := s.tbaHandler.makeMatchReq(matchId)

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

    return scoredMatch
}

func getQualMatchScore(match Match) (int, int) {
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
        12: true,
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
func getPlayoffMatchScore(match Match) (int, int) {
    redScore := 0
    blueScore := 0

	if match.CompLevel == "f" {
		if match.EventKey == "cmptx" {
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
		if getLowerBracketMatchIds()[match.MatchNumber] {
			//Lower Bracket
			if match.EventKey == "cmptx" {
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
		} else if getUpperBracketMatchIds()[match.MatchNumber] {
			//Upper Bracket
			if match.EventKey == "cmptx" { //TODO is there a better way to check the champ event?
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

func sortMatchesByPlayOrder(matches []string) []string {
    //Matches are almost sorted
    //We need to sort it so that matches so qm -> qf -> sf -> f and then sort by match id

    return []string{}
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
			fmt.Println("Starting new scoring iteration")

            //Get a list of matches to score and
            //Sort matches by id (they are almost sorted, but we need to move finals matches to the end)
            matches := make(map[string][]string)
            for _, event := range events() {
                matches[event] = sortMatchesByPlayOrder(s.tbaHandler.makeEventMatchKeysRequest(event))
            }


            //Score matches until we hit one that has not been played

			s.scoringIteration++
			fmt.Println("Finished scoring iteration")
			time.Sleep(5 * time.Minute)
		}
	}(s)
}
