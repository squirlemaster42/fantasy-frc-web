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

    player1Id := CreateUser(player1, *dbDriver)
    player2Id := CreateUser(player2, *dbDriver)
    player3Id := CreateUser(player3, *dbDriver)
    player4Id := CreateUser(player4, *dbDriver)
    player5Id := CreateUser(player5, *dbDriver)
    player6Id := CreateUser(player6, *dbDriver)
    player7Id := CreateUser(player7, *dbDriver)
    player8Id := CreateUser(player8, *dbDriver)

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
}
