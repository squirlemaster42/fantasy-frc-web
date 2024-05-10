package model

import (
	"os"
	"server/database"
	"testing"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func TestCreateDraft(t *testing.T) {

    godotenv.Load()
    dbPassword := os.Getenv("DB_PASSWORD")
    dbUsername := os.Getenv("DB_USERNAME")
    dbIp := os.Getenv("DB_IP")
    dbName := os.Getenv("DB_NAME")


    dbDriver := database.CreateDatabaseDriver(dbUsername, dbPassword, dbIp, dbName)

    player1Name := uuid.New().String()
    player2Name := uuid.New().String()
    player3Name := uuid.New().String()
    player4Name := uuid.New().String()
    player5Name := uuid.New().String()
    player6Name := uuid.New().String()
    player7Name := uuid.New().String()
    player8Name := uuid.New().String()

    player1 := Player{
        Username: player1Name,
        Password: "",
    }
    player2 := Player{
        Username: player2Name,
        Password: "",
    }
    player3 := Player{
        Username: player3Name,
        Password: "",
    }
    player4 := Player{
        Username: player4Name,
        Password: "",
    }
    player5 := Player{
        Username: player5Name,
        Password: "",
    }
    player6 := Player{
        Username: player6Name,
        Password: "",
    }
    player7 := Player{
        Username: player7Name,
        Password: "",
    }
    player8 := Player{
        Username: player8Name,
        Password: "",
    }

    player1Id, _ := CreateUser(player1, *dbDriver)
    player2Id, _ := CreateUser(player2, *dbDriver)
    player3Id, _ := CreateUser(player3, *dbDriver)
    player4Id, _ := CreateUser(player4, *dbDriver)
    player5Id, _ := CreateUser(player5, *dbDriver)
    player6Id, _ := CreateUser(player6, *dbDriver)
    player7Id, _ := CreateUser(player7, *dbDriver)
    player8Id, _ := CreateUser(player8, *dbDriver)

    draftName := uuid.New().String()

    players := []int{
        player1Id,
        player2Id,
        player3Id,
        player4Id,
        player5Id,
        player6Id,
        player7Id,
        player8Id,
    }

    draftId, err := CreateDraft(draftName, players, dbDriver)

    if err != nil {
        t.Error(err)
    }

    if draftId < 0 {
        t.Errorf("Draft with id %d has an invalid id", draftId)
    }

    //Check player orders are correct
    query := `SELECT playerOrder, player FROM DraftPlayers where draftId = $1`
    stmt, err := dbDriver.Connection.Prepare(query)

    if err != nil {
        t.Error(err)
    }

    rows, err := stmt.Query(draftId)

    for rows.Next() {
        var playerOrder int
        var player int
        err = rows.Scan(&playerOrder, &player)

        if err != nil {
            t.Error(err)
        }

        if player == player1Id  && playerOrder != 0{
            t.Errorf("Player %d had expected player order %d but was %d", player, 0, playerOrder)
        } else if player == player2Id  && playerOrder != 1{
            t.Errorf("Player %d had expected player order %d but was %d", player, 1, playerOrder)
        } else if player == player3Id  && playerOrder != 2{
            t.Errorf("Player %d had expected player order %d but was %d", player, 2, playerOrder)
        } else if player == player4Id  && playerOrder != 3{
            t.Errorf("Player %d had expected player order %d but was %d", player, 3, playerOrder)
        } else if player == player5Id  && playerOrder != 4{
            t.Errorf("Player %d had expected player order %d but was %d", player, 4, playerOrder)
        } else if player == player6Id  && playerOrder != 5{
            t.Errorf("Player %d had expected player order %d but was %d", player, 5, playerOrder)
        } else if player == player7Id  && playerOrder != 6{
            t.Errorf("Player %d had expected player order %d but was %d", player, 6, playerOrder)
        } else if player == player8Id  && playerOrder != 7{
            t.Errorf("Player %d had expected player order %d but was %d", player, 7, playerOrder)
        }
    }

    //Delete draft and players
    err = DeleteDraft(draftId, dbDriver)

    if err != nil {
        t.Error(err)
    }

    err = DeletePlayer(player1Id, dbDriver)

    if err != nil {
        t.Error(err)
    }

    err = DeletePlayer(player2Id, dbDriver)

    if err != nil {
        t.Error(err)
    }

    err = DeletePlayer(player3Id, dbDriver)

    if err != nil {
        t.Error(err)
    }

    err = DeletePlayer(player4Id, dbDriver)

    if err != nil {
        t.Error(err)
    }

    err = DeletePlayer(player5Id, dbDriver)

    if err != nil {
        t.Error(err)
    }

    err = DeletePlayer(player6Id, dbDriver)

    if err != nil {
        t.Error(err)
    }

    err = DeletePlayer(player7Id, dbDriver)

    if err != nil {
        t.Error(err)
    }

    err = DeletePlayer(player8Id, dbDriver)

    if err != nil {
        t.Error(err)
    }
}
