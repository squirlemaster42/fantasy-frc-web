package model

import (
    "log"
    "server/database"
    "fmt"
)

type DbMatch struct {
	TbaId             string
	RedAllianceScore  int
	BlueAllianceScore int
	CompLevel         string
	WinningAlliance   string
	Played            bool
	RedAllianceTeams  []string
	BlueAllianceTeams []string
	Dqed              []string
}

func GetMatchFromDb(matchId string, driver *database.DatabaseDriver) *DbMatch {
	match := DbMatch{}
	err := driver.Connection.QueryRow("Select tbaId, redAllianceScore, blueAllianceScore, compLevel, winningAlliance, Played From matches where tbaId = '" + matchId + "'").Scan(&match.TbaId, &match.RedAllianceScore, &match.BlueAllianceScore, &match.CompLevel, &match.WinningAlliance, &match.Played)

	if err != nil {
        log.Print(err)
		return nil
	}

	return &match
}

func isTeamDqed(teamId string, match *DbMatch) bool {
	for _, team := range match.Dqed {
		if teamId == team {
			return true
		}
	}
	return false
}

//TODO figure out where we want to put the validity things
func SaveMatchToDb(match *DbMatch, driver *database.DatabaseDriver) {
	//When we save the relationship between teams and matches we should store if the team was dqed or not
	fmt.Printf("Updating match %s in database\n", match.TbaId)
	if GetMatchFromDb(match.TbaId, driver) != nil {
		//We need to run updates
		fmt.Printf("Updating match %s to database\n", match.TbaId)
		driver.RunExec(fmt.Sprintf("UPDATE Matches SET redAllianceScore = %d, blueAllianceScore = %d, compLevel = '%s', winningAlliance = '%s', Played = %t WHERE tbaId = '%s'",
			match.RedAllianceScore, match.BlueAllianceScore, match.CompLevel, match.WinningAlliance, match.Played, match.TbaId))

		for _, team := range match.RedAllianceTeams {
			driver.RunExec(fmt.Sprintf("UPDATE Matches_Teams SET isDqed = %t WHERE team_tbaId = '%s' AND match_tbaId = '%s'",
				isTeamDqed(team, match), team, match.TbaId))
		}

		for _, team := range match.BlueAllianceTeams {
			driver.RunExec(fmt.Sprintf("UPDATE Matches_Teams SET isDqed = %t WHERE team_tbaId = '%s' AND match_tbaId = '%s'",
				isTeamDqed(team, match), team, match.TbaId))
		}
	} else {
		//We need to insert into the db
		fmt.Printf("Adding match %s to database\n", match.TbaId)
		driver.RunExec(fmt.Sprintf("INSERT INTO Matches (tbaId, redAllianceScore, blueAllianceScore, compLevel, winningAlliance, Played) VALUES ('%s', %d, %d, '%s', '%s', %t)",
			match.TbaId, match.RedAllianceScore, match.BlueAllianceScore, match.CompLevel, match.WinningAlliance, match.Played))

		for _, team := range match.RedAllianceTeams {
			driver.RunExec(fmt.Sprintf("INSERT INTO Matches_Teams (team_tbaId, match_TbaId, isDqed, alliance) VALUES ('%s', '%s', %t, 'red')",
				team, match.TbaId, isTeamDqed(team, match)))
		}

		for _, team := range match.BlueAllianceTeams {
			driver.RunExec(fmt.Sprintf("INSERT INTO Matches_Teams (team_tbaId, match_TbaId, isDqed, alliance) VALUES ('%s', '%s', %t, 'blue')",
				team, match.TbaId, isTeamDqed(team, match)))
		}
	}
}
