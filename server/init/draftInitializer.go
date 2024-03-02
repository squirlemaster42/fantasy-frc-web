package init

import (
	"fmt"
	"os"
	"path/filepath"
	"server/database"
	"strings"
)

func loadCSVIntoDb(dbHandler *database.DatabaseDriver, path string) {
    //Load a csv at a give filepath
    //Add the relevant data to the database
    lines := loadFile(path)
    players := parseCSVToPlayers(lines)
    draftName := filepath.Base(path)

    //Delete old data related to this draft if it exists
    draftId := getDraftId(draftName, dbHandler)
    dbHandler.RunExec(fmt.Sprintf("Delete From Picks Where draftId = %d", draftId))

    //Add players
    for player := range players {
        dbHandler.RunExec(fmt.Sprintf("Insert Into Players (NAME) Values (%s)", player))
    }

    //Add picks for the players
    for player, picks := range players {
        playerId := getPlayerId(player, dbHandler)
        for index, pick := range picks {
            dbHandler.RunExec(fmt.Sprintf("INSERT INTO Picks (draftId, player, PickOrder, pickedTeam) VALUES (%d, %d, %d, %s)", draftId, playerId, index, pick))
        }
    }
}

func getDraftId(draftName string, dbHandler *database.DatabaseDriver) int {
    var draftId int
    err := dbHandler.Connection.QueryRow(fmt.Sprintf("Select Id From drafts Where Name = %s", draftName), 1).Scan(&draftId)
    if err != nil {
        return 0
    }
    return draftId
}

func getPlayerId(playerName string, dbHandler *database.DatabaseDriver) int {
    var playerId int
    err := dbHandler.Connection.QueryRow(fmt.Sprintf("Select Id From Players Where Name = %s", playerName), 1).Scan(&playerId)
    if err != nil {
        return 0
    }
    return playerId
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
        splitLine := strings.Split(line, ",")
        for j := 0; j < len(splitLine); j++ {
            p := playerNames[j]
            teamsForPlayers[p] = append(teamsForPlayers[p], "tba" + splitLine[j])
        }
    }

    return teamsForPlayers
}

func loadFile(filePath string) []string {
    body, err := os.ReadFile(filePath)
    if err != nil {
        return nil
    }

    lines := strings.Split(string(body), "\n")

    return lines
}
