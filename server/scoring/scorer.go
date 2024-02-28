package scoring

import db "server/database"

type Scorer struct {
	TbaHandler *TbaHandler
	DbDriver   *db.DatabaseDriver
}

type DbMatch struct {
    tbaId string
    redAllianceScore int
    blueAllianceScore int
    compLevel string
    winningAlliance string
    Played bool
}

func NewScorer(tbaHandler *TbaHandler, dbDriver *db.DatabaseDriver) *Scorer {
	scorer := Scorer{
		TbaHandler: tbaHandler,
		DbDriver:   dbDriver,
	}
	return &scorer
}

func (s *Scorer) scoreMatch(matchId string) *DbMatch {
	//Check if the match exists in the database and is scored
	//Use Db score if possible
	//If not, query tba and score the match
    dbMatch := s.getMatchFromDb(matchId)
    if dbMatch != nil {
        return dbMatch
    }

    //Get match from tba and score it and then save it in the database
    tbaMatch := s.TbaHandler.makeMatchReq(matchId)
    redScore := 0
    blueScore := 0

    if (tbaMatch.CompLevel == "qm") {
        if (tbaMatch.WinningAlliance == "red") {
            redScore += 4
        } else if (tbaMatch.WinningAlliance == "blue") {
            blueScore += 4
        }

        if (tbaMatch.ScoreBreakdown.Red.MelodyBonusAchieved) {
            redScore += 2
        }

        if (tbaMatch.ScoreBreakdown.Red.EnsembleBonusAchieved) {
            redScore += 2
        }

        if (tbaMatch.ScoreBreakdown.Blue.MelodyBonusAchieved) {
            blueScore += 2
        }

        if (tbaMatch.ScoreBreakdown.Blue.EnsembleBonusAchieved) {
            blueScore += 2
        }
    } else if (tbaMatch.CompLevel == "f") {
        if (tbaMatch.EventKey == "cmptx") {
            if (tbaMatch.WinningAlliance == "red") {
                redScore += 36
            } else if (tbaMatch.WinningAlliance == "blueScore") {
                blueScore += 36
            }
        } else {
            if (tbaMatch.WinningAlliance == "red") {
                redScore += 18
            } else if (tbaMatch.WinningAlliance == "blueScore") {
                blueScore += 18
            }
        }
    } else if (tbaMatch.CompLevel == "sf") {
        if (tbaMatch.MatchNumber == 5 || tbaMatch.MatchNumber == 6 || tbaMatch.MatchNumber == 9 || tbaMatch.MatchNumber == 10 || tbaMatch.MatchNumber == 12 || tbaMatch.MatchNumber == 13) {
            //Lower Bracket
            if (tbaMatch.EventKey == "cmptx") {
                if (tbaMatch.WinningAlliance == "red") {
                    redScore += 18
                } else if (tbaMatch.WinningAlliance == "blueScore") {
                    blueScore += 18
                }
            } else {
                if (tbaMatch.WinningAlliance == "red") {
                    redScore += 9
                } else if (tbaMatch.WinningAlliance == "blueScore") {
                    blueScore += 9
                }
            }
        } else {
            //Upper Breacker
            if (tbaMatch.EventKey == "cmptx") {
                if (tbaMatch.WinningAlliance == "red") { redScore += 30 } else if (tbaMatch.WinningAlliance == "blueScore") {
                    blueScore += 30
                }
            } else {
                if (tbaMatch.WinningAlliance == "red") {
                    redScore += 15
                } else if (tbaMatch.WinningAlliance == "blueScore") {
                    blueScore += 15
                }
            }
        }
    }

    dbMatch = &DbMatch{
        tbaId: tbaMatch.Key,
        redAllianceScore: redScore,
        blueAllianceScore: blueScore,
        compLevel: tbaMatch.CompLevel,
        winningAlliance: tbaMatch.WinningAlliance,
        Played: true,
    }
    s.saveMatchToDb(dbMatch)

    return dbMatch
}

func (s *Scorer) getMatchFromDb(matchId string) *DbMatch {
	driver := s.DbDriver
	rows := driver.RunQuery("Select tbaId, redAllianceScore, blueAllianceScore, compLevel, winningAlliance, compLevel, Played From matches where tbaId = " + matchId + " and Played = 'true'")
    defer rows.Close()

    match := DbMatch{}

    rows.Next()
    err := rows.Scan(match.tbaId, match.redAllianceScore, match.blueAllianceScore, match.compLevel, match.winningAlliance, match.compLevel, match.Played)

    if err != nil {
        return nil
    }

	return &match
}

func (s *Scorer) saveMatchToDb(match *DbMatch) {

}
