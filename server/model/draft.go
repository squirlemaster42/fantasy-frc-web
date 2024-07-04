package model

import (
	"database/sql"
	"server/assert"
	"time"
)

const (
    WAITING_TO_START = iota
    PICKING
    TEAMS_PLAYING
    COMPLETE
)

type Draft struct {
    Id int
    DisplayName string
    Owner int //User
    Status int
    Players []DraftPlayer
}

type DraftPlayer struct {
    User User
    Pending bool
}

type Pick struct {
    Id int
    Player int //DraftPlayer
    PickOrder int
    Pick string //Team
    PickTime time.Time
}

type DraftInvite struct {
    Id int
    DraftId int //Draft
    InvitedPlayer int //User
    InvitingPlayer int //User
    SentTime time.Time
    AcceptedTime time.Time
    Accepted bool
}

func GetDraftsForUser(database *sql.DB, user *User) *[]Draft {
    return nil
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
    query := `INSERT INTO DraftInvites (draftId, invitingPlayer, invitedPlayer, sentTime, accepted) Values ($1, $2, $3, $4, $5);`
    assert := assert.CreateAssertWithContext("Invite Player")
    assert.AddContext("Draft", draft)
    assert.AddContext("Inviting Player", invitingPlayer)
    assert.AddContext("Invited Player", invitedPlayer)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(draft, invitingPlayer, invitedPlayer, time.Now(), false)
    assert.NoError(err, "Failed to insert invite player")
}

func AcceptInvite(database *sql.DB, inviteId int) {
    query := `UPDATE DraftInvites Set accepted = $1, acceptedTime = $2 where id = $3;`
    assert := assert.CreateAssertWithContext("Accept Invite")
    assert.AddContext("Invite Id", inviteId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(true, time.Now(), inviteId)
    assert.NoError(err, "Failed to accept invite")
}

func GetInvites(database *sql.DB, player int) []DraftInvite {
    query := `SELECT id, draftId, invitingPlayer, invitedPlayer, sentTime, acceptedTime, accepted Where invitedPlayer = $1;`
    assert := assert.CreateAssertWithContext("Get Invites")
    assert.AddContext("Player", player)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    rows, err := stmt.Query(player)
    var invites []DraftInvite
    for rows.Next() {
        invite := DraftInvite{}
        rows.Scan(&invite.Id, &invite.DraftId, &invite.InvitingPlayer, &invite.InvitedPlayer, &invite.SentTime, &invite.Accepted, &invite.Accepted)
        invites = append(invites, invite)
    }
    return invites
}

func GetPicks(database *sql.DB, draft int) []Pick {
    query := `SELECT
    Picks.id, Picks,player, Picks,pickOrder, Picks,pick, Picks.pickTime
    From Picks
    Inner Join DraftPlayers On DraftPlayers.id = Picks.player
    Where DraftPlayers.draftId = $1;`
    assert := assert.CreateAssertWithContext("Get Picks")
    assert.AddContext("Draft", draft)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    rows, err := stmt.Query(draft)
    assert.NoError(err, "Failed to query for picks")
    var picks []Pick
    for rows.Next() {
        pick := Pick{}
        rows.Scan(&pick.Id, &pick.Player, &pick.PickOrder, &pick.Pick, &pick.PickTime)
        picks = append(picks, pick)
    }
    return picks
}

//TODO Figure out how we want this to work. should we have a next pick field on the draft or something?
func GetNextPick(database *sql.DB, draft int) Pick {
    return Pick{}
}

//Is using the struct here better?
func MakePick(database *sql.DB, pick Pick) {
    query := `INSERT INTO Picks (player, pickOrder, pick, pickTime) Values ($1, $2, $3, $4);`
    assert := assert.CreateAssertWithContext("Make Pick")
    assert.AddContext("Player", pick.Player)
    assert.AddContext("Pick Order", pick.PickOrder)
    assert.AddContext("Team", pick.Pick)
    assert.AddContext("Pick Time", pick.PickTime)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(pick.Player, pick.PickOrder, pick.Pick, pick.PickTime)
    assert.NoError(err, "Failed to insert pick")
}

func GetAllPicks(database *sql.DB) []string {
    query := `Select pick From Picks;`
    assert := assert.CreateAssertWithContext("Get All Picks")
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    rows, err := stmt.Query()
    assert.NoError(err, "Failed to query picks")
    var picks []string
    for rows.Next() {
        var pick string
        rows.Scan(&pick)
        picks = append(picks, pick)
    }

    return picks
}
