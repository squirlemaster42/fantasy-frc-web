package model

import (
	"database/sql"
	"server/assert"
	"time"
)

type Draft struct {
    Id int
    DisplayName string
    Owner int //User
}

type Pick struct {
    Id int
    Player int //User
    PickOrder int
    Pick string //Team
    PickTime time.Time
}

type DraftInvite struct {
    Id int
    draftId int //Draft
    invitedPlayer int //User
    invitingPlayer int //User
    sentTime time.Time
    acceptedTime time.Time
    accepted bool
}

//TODO should this return the draft id? probably
func CreateDraft(database *sql.DB, owner int, displayName string) {
    query := `INSERT INTO Drafts (DisplayName, Owner) Values ($1, $2);`
    assert := assert.CreateAssertWithContext("Create Draft")
    assert.AddContext("Owner", owner)
    assert.AddContext("Display Name", displayName)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(displayName, owner)
    assert.NoError(err, "Failed to insert draft")
}

//TODO should this return the invite id? probably
func InvitePlayer(database *sql.DB, draft int, invitingPlayer int, invitedPlayer int) {
    query := `INSERT INTO DraftInvites (draftId, invitingPlayer, invitedPlayer, sentTime, accepted) Values ($1, $2, $3, $4, $5)`
    assert := assert.CreateAssertWithContext("Invite Player")
    assert.AddContext("Draft", draft)
    assert.AddContext("Inviting Player", invitingPlayer)
    assert.AddContext("Invited Player", invitedPlayer)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(draft, invitingPlayer, invitedPlayer, time.Now(), false)
    assert.NoError(err, "Failed to insert invite player")
}

func AcceptInvite(draft int, player int) error {
    return nil
}

func GetInvites(player int) (error, []int) {
    return nil, []int{}
}

func GetPicks(draft int) (error, []Pick) {
    return nil, []Pick{}
}

func GetNextPick(draft int) (error, Pick) {
    return nil, Pick{}
}

func MakePick(pick Pick) error {
    return nil
}
