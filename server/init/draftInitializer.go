package init

import (
	"server/database"
	"strings"
)

func loadCSVIntoDb(dbHandle *database.DatabaseDriver, filepath string) {
    //Load a csv at a give filepath
    //Add the relevant data to the database
}

func parseCSVToPlayers(lines []string) map[string][]string {
    playerNames := strings.Split(lines[0], ",")
    teamsPerPlayer := len(lines) - 1

    teamsForPlayers := make(map[string][]string)
    for _, name := range playerNames {
        teamsForPlayers[name] = make([]string, teamsPerPlayer)
    }

    for i := 0; i < teamsPerPlayer; i++ {
        line := lines[i + 1]
        plitLine := strings.Split(line, ",")
        for j := 0; j < len(splitLine); j++ {
            teamsPerPlayer[playerNames[j]] = append(teamsPerPlayer[playerNames[j], splitLine[j])
        }
    }

    return teamsForPlayers
}
