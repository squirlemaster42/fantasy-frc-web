package model

import (
	"database/sql"
	"os"
	"path/filepath"
	"server/database"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

//Returns the Id of the User
func GetOrCreateUser(database *sql.DB, username string) int {
    if !UsernameTaken(database, username) {
        //Use the username as the password because we are just testing
        RegisterUser(database, username, username)
    }
    return GetUserIdByUsername(database, username)
}

func CreateDBConnection() *sql.DB {
    godotenv.Load(filepath.Join("../", ".env"))
    password := os.Getenv("DB_PASSWORD")
    username := os.Getenv("DB_USERNAME")
    ip := os.Getenv("DB_IP")
    name := os.Getenv("DB_NAME")

    return database.RegisterDatabaseConnection(username, password, ip, name)
}


func TestGetDraftsForUser(t *testing.T) {
    db := CreateDBConnection()

    //Setup eight users
    userOne := GetOrCreateUser(db, "UserOne")
    userTwo := GetOrCreateUser(db, "UserTwo")
    userThree := GetOrCreateUser(db, "UserThree")
    userFour := GetOrCreateUser(db, "UserFour")
    userFive := GetOrCreateUser(db, "UserFive")
    userSix := GetOrCreateUser(db, "UserSix")
    userSeven := GetOrCreateUser(db, "UserSeven")
    userEight := GetOrCreateUser(db, "UserEight")

    //Create a draft with user one as the owner
    draftId := CreateDraft(db, userOne, t.Name())
    AddPlayerToDraft(db, draftId, userOne)

    // Invite all other users to the draft
    userTwoInvite := InvitePlayer(db, draftId, userOne, userTwo)
    userThreeInvite := InvitePlayer(db, draftId, userOne, userThree)
    userFourInvite := InvitePlayer(db, draftId, userOne, userFour)
    userFiveInvite := InvitePlayer(db, draftId, userOne, userFive)
    userSixInvite := InvitePlayer(db, draftId, userOne, userSix)
    InvitePlayer(db, draftId, userOne, userSeven)
    InvitePlayer(db, draftId, userOne, userEight)

    //Have some of the users accept the draft
    AcceptInvite(db, userTwoInvite)
    AddPlayerToDraft(db, draftId, userTwo)
    AcceptInvite(db, userThreeInvite)
    AddPlayerToDraft(db, draftId, userThree)
    AcceptInvite(db, userFourInvite)
    AddPlayerToDraft(db, draftId, userFour)
    AcceptInvite(db, userFiveInvite)
    AddPlayerToDraft(db, draftId, userFive)
    AcceptInvite(db, userSixInvite)
    AddPlayerToDraft(db, draftId, userSix)

    //Store players in maps to be used later in the test
    acceptedPlayers := make(map[int]bool)
    pendingPlayers := make(map[int]bool)

    acceptedPlayers[userOne] = true
    acceptedPlayers[userTwo] = true
    acceptedPlayers[userThree] = true
    acceptedPlayers[userFour] = true
    acceptedPlayers[userFive] = true
    acceptedPlayers[userSix] = true
    pendingPlayers[userSeven] = true
    pendingPlayers[userEight] = true

    //Get Drafts for users
    drafts := *GetDraftsForUser(db, userOne)

    //Check Results
    var draft Draft
    for _, d := range drafts {
        if d.Id == draftId {
            draft = d
            break
        }
    }
    assert.Equal(t, t.Name(), draft.DisplayName)
    assert.Equal(t, userOne, draft.Owner.Id)
    assert.Equal(t, FILLING, draft.Status)

    var foundPlayers []int
    for _, player := range draft.Players {
        foundPlayers = append(foundPlayers, player.User.Id)

        if !(acceptedPlayers[player.User.Id] || pendingPlayers[player.User.Id]) {
            assert.Fail(t, "Player %s is not in the draft", player.User.Username)
        }
    }

    assert.Equal(t, 8, len(foundPlayers))
}
