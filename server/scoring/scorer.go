package scoring

import (
	"database/sql"
	"fmt"
	"log"
	"server/model"
	"strings"
	"time"
)

type Scorer struct {
	TbaHandler *TbaHandler
    database *sql.DB
}

func NewScorer(tbaHandler *TbaHandler) *Scorer {
	scorer := Scorer{
		TbaHandler: tbaHandler,
	}
	return &scorer
}

func (s *Scorer) scoreMatchIfNecessary(matchId string, override bool) *model.Match {
	//Check if the match exists in the database and is scored
	//Use Db score if possible
	//If not, query tba and score the match
	fmt.Printf("Scoring match %s\n", matchId)
	dbMatch := model.GetMatch(s.database, matchId)
	if (dbMatch.Played) && !override {
		return dbMatch
	}

	//Get match from tba and score it and then save it in the database
	tbaMatch := s.TbaHandler.makeMatchReq(matchId)
	dbMatch = s.scoreMatch(tbaMatch)
	model.UpdateScore(s.database, dbMatch.TbaId, dbMatch.RedScore, dbMatch.BlueScore)

	return dbMatch
}

func (s *Scorer) scoreMatch(match model.Match) *model.Match {
    redScore := 0
	blueScore := 0

	if match.CompLevel == "qm" {
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
	} else if match.CompLevel == "f" {
        fmt.Println("Scoring Finals")
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
        fmt.Println("Scoring Semi Finals")
		if match.MatchNumber == 5 || match.MatchNumber == 6 || match.MatchNumber == 9 || match.MatchNumber == 10 || match.MatchNumber == 12 || match.MatchNumber == 13 {
			//Lower Bracket
            fmt.Println("Scoring Lower Bracket")
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
		} else {
			//Upper Breacker
            fmt.Println("Scoring Upper Bracket")
			if match.EventKey == "cmptx" {
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

	dqedTeams := match.Alliances.Red.DqTeamKeys
	dqedTeams = append(dqedTeams, match.Alliances.Blue.DqTeamKeys...)
	dqedTeams = append(dqedTeams, match.Alliances.Red.SurrogateTeamKeys...)
	dqedTeams = append(dqedTeams, match.Alliances.Blue.SurrogateTeamKeys...)

    dbMatch := &models.DbMatch{
		TbaId:             match.Key,
		RedAllianceScore:  redScore,
		BlueAllianceScore: blueScore,
		CompLevel:         match.CompLevel,
		WinningAlliance:   match.WinningAlliance,
		Played:            match.ActualTime != 0,
		RedAllianceTeams:  match.Alliances.Red.TeamKeys,
		BlueAllianceTeams: match.Alliances.Blue.TeamKeys,
		Dqed:              dqedTeams,
	}

    return dbMatch
}

func (s *Scorer) getChampEventForTeam(teamId string) string {
	//Get list of teams events from tba
	//Check which event is in the list of champ events
	//We are going to ignore Einstein here since we just use this to determin the ranking score
	//which does not apply to Einstein
	events := s.TbaHandler.makeEventListReq(strings.TrimSpace(teamId))
	//Even though this is O(e*f), where e is the number of events the team played during the season and f is
	//the number of champs field, both will be small so this is probably faster than a hashset
	for _, event := range events {
		for _, champEvent := range s.getChampEvents() {
			if event == champEvent {
				return event
			}
		}
	}
	return ""
}

func (s *Scorer) getChampEvents() []string {
	return []string{"2024cthar", "2024casj"} //TODO add the rest of the events
}

//This was capital but that seemed wrong
//TODO Can we write this without requiring the database
//I think we could just pass in a list of matches and
//the required event information
//something like scoreTeam(matches []Match, event Event)
func (s *Scorer) scoreTeam(teamId string) int {
	//Query all matches for team
	//Get all of the scores
	//Add ranking score
	driver := s.DbDriver
	var score int;
	err := driver.Connection.QueryRow("Select rankingScore From Teams Where tbaId = '" + strings.TrimSpace(teamId) + "';").Scan(&score)
    if err != nil {
        log.Print(err)
    }

	fmt.Printf("-------- Scoring %s --------\n", teamId)
	fmt.Println("Getting previous match scores")
	rows := driver.RunQuery(fmt.Sprintf(`Select redAllianceScore, blueAllianceScore, Played, alliance, isDqed From Matches m
    Left Join Matches_Teams mt On m.tbaId = mt.match_tbaId WHERE mt.team_tbaId = '%s'`, teamId))
	defer rows.Close()

	if rows == nil {
		return score
	}

	fmt.Printf("Raning score: %d\n", score)

	for rows.Next() {
		var redScore int
		var blueScore int
		var played bool
		var alliance string
		var dqed bool
		err := rows.Scan(&redScore, &blueScore, &played, &alliance, &dqed)
		if err != nil {
			return score
		}

		if !played || dqed {
			continue
		}

		if alliance == "red" {
			score += redScore
		} else if alliance == "blue" {
			score += blueScore
		}
	}

	return score
}

func (s *Scorer) updateTeamValidity() {
    currentTeams := models.GetTeamValidity(s.DbDriver)

    for teamName := range currentTeams {
        currentTeams[teamName] = false
    }

    for _, eventName := range s.getChampEvents() {
        for _, teamName := range s.TbaHandler.makeTeamsAtEventRequest(eventName) {
            currentTeams[teamName.Name] = true
        }
    }

    for team, valid := range currentTeams {
        models.UpdateTeamValidity(team, valid, s.DbDriver)
    }
}

func (s *Scorer) getAllPickedTeams() []string {
	var teams []string

	driver := s.DbDriver
	fmt.Println("Getting all picked teams")
	rows := driver.RunQuery("Select pickedTeam from Picks")
	defer rows.Close()

	if rows == nil {
		return teams
	}

	for rows.Next() {
		var pick string
		err := rows.Scan(&pick)
		if err != nil {
			return teams
		}
		teams = append(teams, pick)
	}

	return teams
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
		iteration := 0
		for {
			fmt.Println("Starting new scoring iteration")
			rescore := iteration % 72 == 0

            if rescore {
                s.updateTeamValidity()
            }

			events := s.getChampEvents()

			eventToTeam := make(map[string]string)
			for _, event := range events {
				fmt.Printf("Scoring event: %s\n", event)
				teams := s.TbaHandler.makeTeamsAtEventRequest(event)
				for _, team := range teams {
					fmt.Printf("Scoring team: %s\n", team.Key)
                    teamModel := &models.Team{
                        TbaId: team.Key,
                        Name: team.Nickname,
                        RankingScore: 0,
                        ValidPick: true,
                    }
					models.UpsertTeam(teamModel, s.DbDriver)
					eventToTeam[team.Key] = event
				}
			}

			matchesToScore := make(map[string]bool)
			for _, team := range s.getAllPickedTeams() {
				for _, match := range s.TbaHandler.makeMatchKeysRequest(strings.TrimSpace(team), eventToTeam[strings.TrimSpace(team)]) {
					matchesToScore[match] = true
				}
			}

			for match := range matchesToScore {
				s.scoreMatchIfNecessary(strings.TrimSpace(match), rescore)
			}

			iteration++
			fmt.Println("Finished scoring iteration")
			time.Sleep(5 * time.Minute)
		}
	}(s)
}
