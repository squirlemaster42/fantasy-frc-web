package model

import (
	"database/sql"
	"fmt"
	"math/rand"
	"server/assert"
	"strings"
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
	Id          int
	DisplayName string
	Description string
	Interval    int //Number of seconds to pick
	StartTime   time.Time
	EndTime     time.Time
	Owner       User //User
	Status      string
	Players     []DraftPlayer
    NextPick DraftPlayer
}

func (d *Draft) String() string {
    var stringBuilder strings.Builder
    for i, p := range d.Players {
        stringBuilder.WriteString("\nDraftPlayer - ")
        stringBuilder.WriteString(string(i))
        stringBuilder.WriteString(" {\n")
        stringBuilder.WriteString(p.String())
        stringBuilder.WriteString(" \n}")
    }

    return fmt.Sprintf("Draft: {\nId: %d\n Displayname: %s\n Description: %s\n Interval: %d\n StartTime: %s\n EndTime: %s\n Owner: %s\n Status: %s\n Players: %s\n NextPick: %s\n}",
        d.Id, d.DisplayName, d.Description, d.Interval, d.StartTime.String(), d.EndTime.String(), d.Owner.String(), d.Status, stringBuilder.String(), d.NextPick.String())
}

type DraftPlayer struct {
    Id int
	User    User
    PlayerOrder int
	Pending bool
}

func (d *DraftPlayer) String() string {
    return fmt.Sprintf("DraftPlayer: {\nId: %d\n User: %d\n PlayerOrder: %d\n Pending: %t\n}", d.Id, d.User.Id, d.PlayerOrder, d.Pending)
}


type Pick struct {
	Id        int
	Player    int //DraftPlayer
	Pick      string //Team
	PickTime  time.Time
}

func (p *Pick) String() string {
    return fmt.Sprintf("Pick: {\nId: %d\n Player: %d\n Pick: %s\n PickTime: %s\n}", p.Id, p.Player, p.Pick, p.PickTime.String())
}

type DraftInvite struct {
	Id int
	DraftId int //Draft
    DraftName string
	InvitedPlayer int //User
	InvitingPlayer int //User
    InvitingPlayerName string
	SentTime time.Time
	AcceptedTime time.Time
	Accepted bool
}

func (d *DraftInvite) String() string {
    return fmt.Sprintf("DraftInvite: {\nId: %d\n DraftId: %d\n InvitingPlayer: %d\n InvitedPlayer: %d\n SentTime: %s\n AcceptedTime: %s\n Accepted: %t\n}",
        d.Id, d.DraftId, d.InvitingPlayer, d.InvitedPlayer, d.SentTime.String(), d.AcceptedTime.String(), d.Accepted)
}

func GetStatusString(status int) string {
	switch status {
	case FILLING:
		return "Filling"
	case WAITING_TO_START:
		return "Waiting To Start"
	case PICKING:
		return "Picking"
	case TEAMS_PLAYING:
		return "Teams Playing"
	case COMPLETE:
		return "Complete"
	default:
		return "Invalid"
	}
}

// TODO Need to include next pick in this and profile picture
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

        nextPick := DraftPlayer{}
        if GetStatusString(status) == GetStatusString(PICKING) {
            nextPick = NextPick(database, draftId)
        }

		draft := Draft{
			Id:          draftId,
			DisplayName: displayName,
			Owner: User{
				Id:       ownerId,
				Username: ownerUsername,
			},
			Status: GetStatusString(status),
			Players: make([]DraftPlayer, 0),
            NextPick: nextPick,
		}

		playerQuery := `Select Distinct
            Users.Id As UserId,
            Users.Username,
            COALESCE(DraftInvites.accepted, 't') As Accepted
        From Users
        Left Join DraftPlayers On DraftPlayers.Player = Users.Id
        Left Join DraftInvites On DraftInvites.InvitedPlayer = Users.Id
        Where (DraftInvites.DraftId = $1 Or DraftPlayers.DraftId = $1)`;

		playerStmt, err := database.Prepare(playerQuery)
		assert.NoError(err, "Failed to prepare player query")
		playerRows, err := playerStmt.Query(draftId)

		for playerRows.Next() {
			var userId int
			var username string
			var accepted bool

			playerRows.Scan(&userId, &username, &accepted)
			draftPlayer := DraftPlayer{
				User: User{
					Id:       userId,
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

func CreateDraft(database *sql.DB, draft *Draft) int {
	query := `INSERT INTO Drafts (DisplayName, Owner, Description, StartTime, EndTime, Interval) Values ($1, $2, $3, $4, $5, $6) RETURNING Id;`
	assert := assert.CreateAssertWithContext("Create Draft")
	assert.AddContext("Owner", draft.Owner)
	assert.AddContext("Display Name", draft.DisplayName)
	assert.AddContext("Interval", draft.Interval)
	assert.AddContext("Start Time", draft.StartTime)
	assert.AddContext("End Time", draft.EndTime)
	assert.AddContext("Status", draft.Status)
	assert.AddContext("Description", draft.Description)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	var draftId int
	err = stmt.QueryRow(draft.DisplayName, draft.Owner.Id, draft.Description, draft.StartTime, draft.EndTime, draft.Interval).Scan(&draftId)
	assert.NoError(err, "Failed to insert draft")
	playerQuery := `INSERT INTO DraftPlayers (draftId, player) Values ($1, $2);`
	stmt, err = database.Prepare(playerQuery)
	assert.NoError(err, "Failed to prepare statement")
	_, err = stmt.Exec(draftId, draft.Owner.Id)
	assert.NoError(err, "Failed to insert draft")
	return draftId
}

// TODO Do we need to get the draft owner
func GetDraft(database *sql.DB, draftId int) Draft {
    query := `Select DisplayName, COALESCE(Description, ''), COALESCE(Status, '') As Status, StartTime, EndTime, extract('epoch' from Interval)::int As Interval, Owner From Drafts Where Id = $1;`
	assert := assert.CreateAssertWithContext("Get Draft")
	assert.AddContext("Draft Id", draftId)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	draft := Draft{
		Id: draftId,
	}
    var ownerId int
	err = stmt.QueryRow(draftId).Scan(&draft.DisplayName, &draft.Description, &draft.Status, &draft.StartTime, &draft.EndTime, &draft.Interval, &ownerId)
	assert.NoError(err, "Failed to get draft")

    //TODO we need to include pick order, maybe we also want to include the picks
    //We probably want to show the status
    playerQuery := `Select
                        Users.Id As UserId,
                        Users.Username,
                        COALESCE(DraftInvites.accepted, 't') As Accepted,
                        COALESCE(DraftPlayers.PlayerOrder, -1) As PlayerOrder,
                        COALESCE(DraftPlayers.Id, DraftInvites.Id) As Id
                    From Users
                    Left Join DraftPlayers On DraftPlayers.Player = Users.Id
                    Left Join DraftInvites On DraftInvites.InvitedPlayer = Users.Id
                    Where (DraftInvites.DraftId = $1 And DraftPlayers.DraftId = $1)
                        Or (DraftInvites.Id Is Null And DraftPlayers.DraftId = $1)
                        Or (DraftPlayers.Id Is Null And DraftInvites.DraftId = $1)
                    Order By DraftPlayers.PlayerOrder ASC;`

    playerStmt, err := database.Prepare(playerQuery)
    assert.NoError(err, "Failed to prepare player query")
    playerRows, err := playerStmt.Query(draftId)

    for playerRows.Next() {
        var userId int
        var username string
        var accepted bool
        var playerOrder int
        var playerId int

        if userId == ownerId {
            draft.Owner = User{
                Id: userId,
                Username: username,
            }
        }

        playerRows.Scan(&userId, &username, &accepted, &playerOrder, &playerId)
        draftPlayer := DraftPlayer{
            Id: playerId,
            User: User{
                Id: userId,
                Username: username,
            },
            PlayerOrder: playerOrder,
            Pending: !accepted,
        }

        draft.Players = append(draft.Players, draftPlayer)
    }

    return draft
}

func UpdateDraft(database *sql.DB, draft *Draft) {
    //TODO This should update the draft instead
    query := `INSERT INTO Drafts (DisplayName, Owner, Description, StartTime, EndTime, Interval) Values ($1, $2, $3, $4, $5, $6) RETURNING Id;`
    assert := assert.CreateAssertWithContext("Create Draft")
    assert.AddContext("Owner", draft.Owner)
    assert.AddContext("Display Name", draft.DisplayName)
    assert.AddContext("Interval", draft.Interval)
    assert.AddContext("Start Time", draft.StartTime)
    assert.AddContext("End Time", draft.EndTime)
    assert.AddContext("Status", draft.Status)
    assert.AddContext("Description", draft.Description)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(draft.DisplayName, draft.Owner, draft.Description, draft.StartTime, draft.EndTime, draft.Interval)
    assert.NoError(err, "Failed to insert draft")
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
    query := `SELECT
            di.Id,
            draftId,
            invitingPlayer,
            u.username,
            invitedPlayer,
            sentTime,
            acceptedTime,
            accepted,
            d.Displayname
        From DraftInvites di
        Inner Join Drafts d On di.DraftId = d.Id
        Inner Join Users u On di.InvitingPlayer = u.Id
        Where invitedPlayer = $1;`
    assert := assert.CreateAssertWithContext("Get Invites")
    assert.AddContext("Player", player)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    rows, err := stmt.Query(player)
    var invites []DraftInvite
    for rows.Next() {
        invite := DraftInvite{}
        rows.Scan(&invite.Id,
            &invite.DraftId,
            //TODO we will want to get the name here
            &invite.InvitingPlayer,
            &invite.InvitingPlayerName,
            &invite.InvitedPlayer,
            &invite.SentTime,
            &invite.AcceptedTime,
            &invite.Accepted,
            &invite.DraftName)
        invites = append(invites, invite)
    }
    return invites
}

func GetPicks(database *sql.DB, draft int) []Pick {
    query := `SELECT
    Picks.id, Picks.player, Picks.pick, Picks.pickTime
    From Picks
    Inner Join DraftPlayers On DraftPlayers.id = Picks.player
    Where DraftPlayers.draftId = $1
    Order By PickTime Asc;`
    assert := assert.CreateAssertWithContext("Get Picks")
    assert.AddContext("Draft", draft)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    rows, err := stmt.Query(draft)
    assert.NoError(err, "Failed to query for picks")
    var picks []Pick
    for rows.Next() {
        pick := Pick{}
        rows.Scan(&pick.Id, &pick.Player, &pick.Pick, &pick.PickTime)
        picks = append(picks, pick)
    }
    return picks
}

func GetDraftPlayerId(database *sql.DB, draftId int, playerId int) int {
    query := `Select Id From DraftPlayers Where draftId = $1 And player = $2`
    assert := assert.CreateAssertWithContext("Get Draft Player Id")
	assert.AddContext("Draft Id", draftId)
	assert.AddContext("Player Id", playerId)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	var draftPlayerId int
	err = stmt.QueryRow(draftId, playerId).Scan(&draftPlayerId)
	assert.NoError(err, "Failed to get draft player")
	return draftPlayerId
}

func MakePick(database *sql.DB, pick Pick) {
	query := `INSERT INTO Picks (player, pick, pickTime) Values ($1, $2, $3);`
	assert := assert.CreateAssertWithContext("Make Pick")
	assert.AddContext("Player", pick.Player)
	assert.AddContext("Team", pick.Pick)
	assert.AddContext("Pick Time", pick.PickTime)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	_, err = stmt.Exec(pick.Player, pick.Pick, pick.PickTime)
	assert.NoError(err, "Failed to insert pick")
}

func SetPlayerOrder(database *sql.DB, draftPlayerId int, playerOrder int) {
	query := `Update DraftPlayers Set PlayerOrder = $1 Where DraftPlayers.Id = $2;`
	assert := assert.CreateAssertWithContext("Set Player Order")
	assert.AddContext("Draft Player", draftPlayerId)
	assert.AddContext("Player Order", playerOrder)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	_, err = stmt.Exec(playerOrder, draftPlayerId)
	assert.NoError(err, "Failed to set player order")
}

func GetAllPicks(database *sql.DB) []string {
	query := `Select Distinct pick From Picks;`
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

func HasBeenPicked(database *sql.DB, draftId int, team string) bool {
    query := `SELECT
    Count(*) As num
    From Picks
    Inner Join DraftPlayers On DraftPlayers.id = Picks.player
    Where DraftPlayers.draftId = $1
    And Picks.pick = $2;`
    assert := assert.CreateAssertWithContext("Has Been Picked")
    assert.AddContext("Draft", draftId)
    assert.AddContext("Team", team)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var numPicked int
    err = stmt.QueryRow(draftId, team).Scan(&numPicked)
    assert.NoError(err, "Failed to query for picks")
    return numPicked != 0
}

func RandomizePickOrder(database *sql.DB, draftId int) {
    draftModel := GetDraft(database, draftId)
    awaitingAssignment := draftModel.Players
    order := 0
    assert := assert.CreateAssertWithContext("Randomize Pick Order")

    for len(awaitingAssignment) > 0 {
        selectedPlayer := rand.Intn(len(awaitingAssignment))
        player := awaitingAssignment[selectedPlayer]
        removePlayer(awaitingAssignment, selectedPlayer)
        draftPlayerId := GetDraftPlayerId(database, draftId, player.User.Id)

        query := `Update DraftPlayers Set PlayerOrder = $1 Where Id = $2`
        stmt, err := database.Prepare(query)
        assert.NoError(err, "Failed to prepare statement")
        stmt.Exec(order, draftPlayerId)
        order++
    }
}

func removePlayer(arr []DraftPlayer, i int) []DraftPlayer {
    arr[i] = arr[len(arr)-1]
    return arr[:len(arr)-1]
}

func NextPick(database *sql.DB, draftId int) DraftPlayer {
    //We need to get the last two picks
    picks := GetPicks(database, draftId)
    draft := GetDraft(database, draftId)
    var nextPlayer DraftPlayer

    //I dont think we need to account for the case where there are only two players
    if len(picks) < 2 {
        for _, player := range draft.Players {
            if player.PlayerOrder == len(picks) {
                nextPlayer = player
            }
        }
    } else {
        //We can then figure out what direction
        //we are going and if we hit the
        //End then we decide what the next pick is
        lastPlayer := GetDraftPlayerFromDraft(draft, picks[len(picks) - 1].Player)
        secondLastPick := GetDraftPlayerFromDraft(draft, picks[len(picks) - 2].Player)
        direction := lastPlayer.PlayerOrder - secondLastPick.PlayerOrder
        if lastPlayer.User.Id == secondLastPick.User.Id {
            if lastPlayer.PlayerOrder == len(draft.Players) - 1 {
                direction = -1
            } else {
                direction = 1
            }
        }
        if len(picks) % 8 == 0 { //TODO Change to number of picks in draft
            direction = 0
        }
        //We know draft.players is order by player order
        nextPlayer = draft.Players[lastPlayer.PlayerOrder + direction]
    }

    //Take the pick and make it into a draft player

    return nextPlayer
}

func GetDraftPlayerFromDraft(draft Draft, draftPlayerId int) DraftPlayer {
    for _, p := range draft.Players {
        if p.Id == draftPlayerId {
            return p
        }
    }
    return DraftPlayer{} //TODO Error if we fail to find?
}
