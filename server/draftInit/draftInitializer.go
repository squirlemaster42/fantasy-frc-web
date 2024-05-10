package draftInit

import (
	"fmt"
	"log"
	"os"

	"path/filepath"
	"server/database"
	"strings"
)

func LoadCSVIntoDb(dbHandler *database.DatabaseDriver, path string) {
    //Load a csv at a give filepath
    //Add the relevant data to the database
    lines := loadFile(path)
    players := parseCSVToPlayers(lines)
    draftName := filepath.Base(path)
    fmt.Printf("Parsing important data from draft file %s\n", draftName)

    //Delete old data related to this draft if it exists
    fmt.Println("Deleting old picks")
    draftId := getDraftId(strings.TrimSpace(draftName), dbHandler)
    if draftId != -1 {
        dbHandler.RunExec(fmt.Sprintf("Delete From Picks Where draftId = %d", draftId))
    }

    //Add draft to database
    if draftId == -1 {
        fmt.Println("Adding draft to the database")
        dbHandler.RunExec(fmt.Sprintf("Insert Into Drafts (Name) Values ('%s')", strings.TrimSpace(draftName)))
        draftId = getDraftId(strings.TrimSpace(draftName), dbHandler)
    }

    //Add players
    fmt.Println("Adding players to the database")
    for player := range players {
        if getPlayerId(strings.TrimSpace(player), dbHandler) == -1 {
            fmt.Printf("Adding player: %s to the database\n", player)
            dbHandler.RunExec(fmt.Sprintf("Insert Into Players (NAME) Values ('%s')", strings.TrimSpace(player)))
        }
    }

    //Add picks for the players
    fmt.Println("Adding picks to database")
    for playerOrder, playerName := range parsePlayerOrder(lines[0] ){
        playerId := getPlayerId(strings.TrimSpace(playerName), dbHandler)

        //Add player order to database
        dbHandler.RunExec(fmt.Sprintf("Insert Into DraftPlayers (draftId, playerOrder, player) Values (%d, %d, %d)", draftId, playerOrder, playerId))

        for index, pick := range players[playerName] {
            if !teamExists(strings.TrimSpace(pick), dbHandler) {
                fmt.Printf("Adding team %s to database\n", strings.TrimSpace(pick))
                dbHandler.RunExec(fmt.Sprintf("INSERT INTO Teams (tbaid) VALUES ('%s')", strings.TrimSpace(pick)))
            }
            dbHandler.RunExec(fmt.Sprintf("INSERT INTO Picks (draftId, player, PickOrder, pickedTeam) VALUES (%d, %d, %d, '%s')", draftId, playerId, index, strings.TrimSpace(pick)))
        }
    }
}

func getDraftId(draftName string, dbHandler *database.DatabaseDriver) int {
    var draftId int
    err := dbHandler.Connection.QueryRow(fmt.Sprintf("Select Id From drafts Where Name = '%s'", draftName)).Scan(&draftId)
    if err != nil {
        log.Print(err)
        return -1
    }
    return draftId
}

func getPlayerId(playerName string, dbHandler *database.DatabaseDriver) int {
    var playerId int
    err := dbHandler.Connection.QueryRow(fmt.Sprintf("Select Id From Players Where Name = '%s'", playerName)).Scan(&playerId)
    if err != nil {
        return -1
    }
    return playerId
}

func teamExists(tbaId string, dbHandler *database.DatabaseDriver) bool {
    var teamId string
    err := dbHandler.Connection.QueryRow(fmt.Sprintf("Select tbaid From Teams Where tbaid = '%s'", tbaId)).Scan(&teamId)
    if err != nil {
        log.Print(err)
        return false
    }
    return true
}

func parsePlayerOrder(line string) []string {
    return strings.Split(line, ",")
}

func parseCSVToPlayers(lines []string) map[string][]string {
    playerNames := strings.Split(lines[0], ",")
    teamsPerPlayer := len(lines) - 1

    teamsForPlayers := make(map[string][]string)
    for _, player := range playerNames {
        teamsForPlayers[player] = make([]string, teamsPerPlayer)
    }

    for i := 0; i < teamsPerPlayer; i++ {
        line := lines[i + 1]
        splitLine := strings.Split(line, ",")
        for j := 0; j < len(splitLine); j++ {
            p := playerNames[j]
            teamsForPlayers[p][i] = fmt.Sprintf("frc%s", splitLine[j])
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
