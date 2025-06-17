package model

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"server/database"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

//Returns the Id of the User
func GetOrCreateUser(database *sql.DB, username string) uuid.UUID {
    if !UsernameTaken(database, username) {
        //Use the username as the password because we are just testing
        RegisterUser(database, username, username)
    }
    return GetUserUuidByUsername(database, username)
}

func GetOrCreateTeam(database *sql.DB, tbaId string) *Team {
    team := GetTeam(database, tbaId)
    if team != nil {
        return team
    }
    CreateTeam(database, tbaId, tbaId);
    return GetTeam(database, tbaId)
}

func CreateDBConnection(t *testing.T) *sql.DB {
    err := godotenv.Load(filepath.Join("../", ".env"))

    if err != nil {
        t.Fatal(err)
    }

    password := os.Getenv("DB_PASSWORD")
    username := os.Getenv("DB_USERNAME")
    ip := os.Getenv("DB_IP")
    name := os.Getenv("DB_NAME")

    return database.RegisterDatabaseConnection(username, password, ip, name)
}

func TestGetDraftsForUser(t *testing.T) {
    db := CreateDBConnection(t)

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
    d := DraftModel {
        DisplayName: t.Name(),
        Owner: User{ UserUuid: userOne},
        Status: FILLING,
    }
    draftId := CreateDraft(db, &d)

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
    acceptedPlayers := make(map[uuid.UUID]bool)
    pendingPlayers := make(map[uuid.UUID]bool)

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
    var draft DraftModel
    for _, d := range drafts {
        if d.Id == draftId {
            draft = d
            break
        }
    }
    assert.Equal(t, t.Name(), draft.DisplayName)
    assert.Equal(t, userOne, draft.Owner.UserUuid)
    assert.Equal(t, FILLING, draft.Status)

    var foundPlayers []uuid.UUID
    for _, player := range draft.Players {
        foundPlayers = append(foundPlayers, player.User.UserUuid)

        if !acceptedPlayers[player.User.UserUuid] && !pendingPlayers[player.User.UserUuid] {
            assert.Fail(t, "Player %s is not in the draft", player.User.Username)
        }
    }

    assert.Equal(t, 8, len(foundPlayers))
}

func TestGetPicksInDraft(t *testing.T) {
    db := CreateDBConnection(t)

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
    d := DraftModel {
        DisplayName: t.Name(),
        Owner: User{ UserUuid: userOne},
        Status: FILLING,
    }
    draftId := CreateDraft(db, &d)

    // Invite all other users to the draft
    userTwoInvite := InvitePlayer(db, draftId, userOne, userTwo)
    userThreeInvite := InvitePlayer(db, draftId, userOne, userThree)
    userFourInvite := InvitePlayer(db, draftId, userOne, userFour)
    userFiveInvite := InvitePlayer(db, draftId, userOne, userFive)
    userSixInvite := InvitePlayer(db, draftId, userOne, userSix)
    userSevenInvite := InvitePlayer(db, draftId, userOne, userSeven)
    userEightInvite := InvitePlayer(db, draftId, userOne, userEight)

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
    acceptedPlayers := make(map[uuid.UUID]bool)
    pendingPlayers := make(map[uuid.UUID]bool)

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
    var draft DraftModel
    for _, d := range drafts {
        if d.Id == draftId {
            draft = d
            break
        }
    }
    assert.Equal(t, t.Name(), draft.DisplayName)
    assert.Equal(t, userOne, draft.Owner.UserUuid)
    assert.Equal(t, FILLING, draft.Status)

    var foundPlayers []uuid.UUID
    for _, player := range draft.Players {
        foundPlayers = append(foundPlayers, player.User.UserUuid)

        if !acceptedPlayers[player.User.UserUuid] && !pendingPlayers[player.User.UserUuid] {
            assert.Fail(t, "Player %s is not in the draft", player.User.Username)
        }
    }

    assert.Equal(t, 8, len(foundPlayers))

    AcceptInvite(db, userSevenInvite)
    AddPlayerToDraft(db, draftId, userSeven)
    AcceptInvite(db, userEightInvite)
    AddPlayerToDraft(db, draftId, userEight)

    teamOne := *GetOrCreateTeam(db, "TeamOne")
    teamTwo := *GetOrCreateTeam(db, "TeamTwo")
    teamThree := *GetOrCreateTeam(db, "TeamThree")
    teamFour := *GetOrCreateTeam(db, "TeamFour")
    teamFive := *GetOrCreateTeam(db, "TeamFive")
    teamSix := *GetOrCreateTeam(db, "TeamSix")
    teamSeven := *GetOrCreateTeam(db, "TeamSeven")
    teamEight := *GetOrCreateTeam(db, "TeamEight")
    teamNine:= *GetOrCreateTeam(db, "TeamNine")

    draftPlayerOne := GetDraftPlayerId(db, draftId, userOne)
    draftPlayerTwo := GetDraftPlayerId(db, draftId, userTwo)
    draftPlayerThree := GetDraftPlayerId(db, draftId, userThree)
    draftPlayerFour := GetDraftPlayerId(db, draftId, userFour)
    draftPlayerFive := GetDraftPlayerId(db, draftId, userFive)
    draftPlayerSix := GetDraftPlayerId(db, draftId, userSix)
    draftPlayerSeven := GetDraftPlayerId(db, draftId, userSeven)
    draftPlayerEight := GetDraftPlayerId(db, draftId, userEight)

    SetPlayerOrder(db, draftPlayerOne, 0)
    SetPlayerOrder(db, draftPlayerTwo, 1)
    SetPlayerOrder(db, draftPlayerThree, 2)
    SetPlayerOrder(db, draftPlayerFour, 3)
    SetPlayerOrder(db, draftPlayerFive, 4)
    SetPlayerOrder(db, draftPlayerSix, 5)
    SetPlayerOrder(db, draftPlayerSeven, 6)
    SetPlayerOrder(db, draftPlayerEight, 7)

    assert.Equal(t, draftPlayerOne, NextPick(db, draftId).Id, fmt.Sprintf("Expected player One, got %s.", NextPick(db, draftId).User.Username))
    pick := Pick{
        Player: draftPlayerOne,
        Pick: sql.NullString{
            Valid: true,
            String: teamOne.TbaId,
        },
        PickTime: sql.NullTime{
            Valid: true,
            Time: time.Now(),
        },
    }
    MakePick(db, pick)
    assert.Equal(t, draftPlayerTwo, NextPick(db, draftId).Id, fmt.Sprintf("Expected player Two, got %s.", NextPick(db, draftId).User.Username))
    time.Sleep(1 * time.Second)
    pick = Pick{
        Player: draftPlayerTwo,
        Pick: sql.NullString{
            Valid: true,
            String: teamTwo.TbaId,
        },
        PickTime: sql.NullTime{
            Valid: true,
            Time: time.Now(),
        },
    }
    MakePick(db, pick)
    assert.Equal(t, draftPlayerThree, NextPick(db, draftId).Id, fmt.Sprintf("Expected player Three, got %s.", NextPick(db, draftId).User.Username))
    time.Sleep(1 * time.Second)
    pick = Pick{
        Player: draftPlayerThree,
        Pick: sql.NullString{
            Valid: true,
            String: teamThree.TbaId,
        },
        PickTime: sql.NullTime{
            Valid: true,
            Time: time.Now(),
        },
    }
    MakePick(db, pick)
    assert.Equal(t, draftPlayerFour, NextPick(db, draftId).Id, fmt.Sprintf("Expected player Four, got %s.", NextPick(db, draftId).User.Username))
    time.Sleep(1 * time.Second)
    pick = Pick{
        Player: draftPlayerFour,
        Pick: sql.NullString{
            Valid: true,
            String: teamFour.TbaId,
        },
        PickTime: sql.NullTime{
            Valid: true,
            Time: time.Now(),
        },
    }
    MakePick(db, pick)
    assert.Equal(t, draftPlayerFive, NextPick(db, draftId).Id, fmt.Sprintf("Expected player Five, got %s.", NextPick(db, draftId).User.Username))
    time.Sleep(1 * time.Second)
    pick = Pick{
        Player: draftPlayerFive,
        Pick: sql.NullString{
            Valid: true,
            String: teamFive.TbaId,
        },
        PickTime: sql.NullTime{
            Valid: true,
            Time: time.Now(),
        },
    }
    MakePick(db, pick)
    assert.Equal(t, draftPlayerSix, NextPick(db, draftId).Id, fmt.Sprintf("Expected player Six, got %s.", NextPick(db, draftId).User.Username))
    time.Sleep(1 * time.Second)
    pick = Pick{
        Player: draftPlayerSix,
        Pick: sql.NullString{
            Valid: true,
            String: teamSix.TbaId,
        },
        PickTime: sql.NullTime{
            Valid: true,
            Time: time.Now(),
        },
    }
    MakePick(db, pick)
    assert.Equal(t, draftPlayerSeven, NextPick(db, draftId).Id, fmt.Sprintf("Expected player Seven, got %s.", NextPick(db, draftId).User.Username))
    time.Sleep(1 * time.Second)
    pick = Pick{
        Player: draftPlayerSeven,
        Pick: sql.NullString{
            Valid: true,
            String: teamSeven.TbaId,
        },
        PickTime: sql.NullTime{
            Valid: true,
            Time: time.Now(),
        },
    }
    MakePick(db, pick)
    assert.Equal(t, draftPlayerEight, NextPick(db, draftId).Id, fmt.Sprintf("Expected player Eight, got %s.", NextPick(db, draftId).User.Username))
    time.Sleep(1 * time.Second)
    pick = Pick{
        Player: draftPlayerEight,
        Pick: sql.NullString{
            Valid: true,
            String: teamEight.TbaId,
        },
        PickTime: sql.NullTime{
            Valid: true,
            Time: time.Now(),
        },
    }
    MakePick(db, pick)
    assert.Equal(t, draftPlayerEight, NextPick(db, draftId).Id, fmt.Sprintf("Expected player Eight, got %s.", NextPick(db, draftId).User.Username))
    time.Sleep(1 * time.Second)
    pick = Pick{
        Player: draftPlayerEight,
        Pick: sql.NullString{
            Valid: true,
            String: teamNine.TbaId,
        },
        PickTime: sql.NullTime{
            Valid: true,
            Time: time.Now(),
        },
    }
    MakePick(db, pick)
    assert.Equal(t, draftPlayerSeven, NextPick(db, draftId).Id, fmt.Sprintf("Expected player Seven, got %s.", NextPick(db, draftId).User.Username))
    picks, err := GetPicks(db, draft.Id)
    if err != nil {
        t.Fatal(err)
    }
    assert.Equal(t, 9, len(picks))

    assert.Equal(t, teamOne.TbaId, picks[0].Pick)
    assert.Equal(t, teamTwo.TbaId, picks[1].Pick)
    assert.Equal(t, teamThree.TbaId, picks[2].Pick)
    assert.Equal(t, teamFour.TbaId, picks[3].Pick)
    assert.Equal(t, teamFive.TbaId, picks[4].Pick)
    assert.Equal(t, teamSix.TbaId, picks[5].Pick)
    assert.Equal(t, teamSeven.TbaId, picks[6].Pick)
    assert.Equal(t, teamEight.TbaId, picks[7].Pick)
    assert.Equal(t, teamNine.TbaId, picks[8].Pick)
}
