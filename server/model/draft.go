package model

import (
	"database/sql"
	"fmt"
	"server/assert"
	"time"
)

const (
    FILLING = iota
    WAITING_TO_START
    PICKING
    TEAMS_PLAYING
    COMPLETE
)

type Draft struct {
    Id int
    DisplayName string
    Owner User //User
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

func GetDraftsForUser(database *sql.DB, user int) *[]Draft {
    query := `SELECT DISTINCT
        Drafts.Id,
        displayName,
        owners.Id As ownerId,
        owners.Username As OwnerUsername,
        status
    From Drafts
    Left Join DraftPlayers On DraftPlayers.DraftId = Drafts.Id
    Left Join DraftInvites On DraftInvites.DraftId = Drafts.Id And Drafts.Status = $1
    Left Join Users dpUsers On DraftPlayers.Player = dpUsers.Id
    Left Join Users diUsers On DraftInvites.InvitedPlayer = diUsers.Id
    Left Join Users owners On Drafts.owner = owners.Id
    Where DraftPlayers.Player = $2 Or DraftInvites.InvitedPlayer = $2;`
    assert := assert.CreateAssertWithContext("Get Invites")
    assert.AddContext("User", user)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    rows, err := stmt.Query(FILLING, user)
    var drafts []Draft
    for rows.Next() {
        var draftId int
        var displayName string
        var ownerId int
        var ownerUsername string
        var status int
        rows.Scan(&draftId, &displayName, &ownerId, &ownerUsername, &status)

        draft := Draft{
            Id: draftId,
            DisplayName: displayName,
            Owner: User{
                Id: ownerId,
                Username: ownerUsername,
            },
            Status: status,
            Players: make([]DraftPlayer, 0),
        }

        playerQuery := `Select
            Users.Id As UserId,
            Users.Username,
            COALESCE(DraftInvites.accepted, 't') As Accepted
        From Users
        Left Join DraftPlayers On DraftPlayers.Player = Users.Id
        Left Join DraftInvites On DraftInvites.InvitedPlayer = Users.Id
        Where (DraftInvites.DraftId = $1 And DraftPlayers.DraftId = $1)
        Or (DraftInvites.Id Is Null And DraftPlayers.DraftId = $1)
        Or (DraftPlayers.Id Is Null And DraftInvites.DraftId = $1);`

        playerStmt, err := database.Prepare(playerQuery)
        assert.NoError(err, "Failed to prepare player query")
        playerRows, err := playerStmt.Query(draftId)

        for playerRows.Next() {
            var userId int
            var username string
            var accepted bool

            playerRows.Scan(&userId, &username, &accepted)

            fmt.Println(userId)
            fmt.Println(username)
            fmt.Println(accepted)

            draftPlayer := DraftPlayer{
                User: User{
                    Id: userId,
                    Username: username,
                },
                Pending: !accepted,
            }

            draft.Players = append(draft.Players, draftPlayer)
        }

        drafts = append(drafts, draft)
    }

    return &drafts
}

func CreateDraft(database *sql.DB, owner int, displayName string) int {
    query := `INSERT INTO Drafts (DisplayName, Owner) Values ($1, $2) RETURNING Id;`
    assert := assert.CreateAssertWithContext("Create Draft")
    assert.AddContext("Owner", owner)
    assert.AddContext("Display Name", displayName)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var draftId int
    err = stmt.QueryRow(displayName, owner).Scan(&draftId)
    assert.NoError(err, "Failed to insert draft")
    return draftId
}

func InvitePlayer(database *sql.DB, draft int, invitingPlayer int, invitedPlayer int) int {
    query := `INSERT INTO DraftInvites (draftId, invitingPlayer, invitedPlayer, sentTime, accepted) Values ($1, $2, $3, $4, $5) RETURNING Id;`
    assert := assert.CreateAssertWithContext("Invite Player")
    assert.AddContext("Draft", draft)
    assert.AddContext("Inviting Player", invitingPlayer)
    assert.AddContext("Invited Player", invitedPlayer)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var inviteId int
    err = stmt.QueryRow(draft, invitingPlayer, invitedPlayer, time.Now(), false).Scan(&inviteId)
    assert.NoError(err, "Failed to insert invite player")
    return inviteId
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

func AddPlayerToDraft(database *sql.DB, draft int, player int) {
    query := `INSERT INTO DraftPlayers (draftId, player) Values ($1, $2);`
    assert := assert.CreateAssertWithContext("Accept Invite")
    assert.AddContext("Draft", draft)
    assert.AddContext("Player", player)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(draft, player)
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
