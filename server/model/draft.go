package model

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"server/assert"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type DraftState string

const (
    FILLING DraftState = "Filling"
    WAITING_TO_START DraftState = "Waiting to Start"
    PICKING DraftState = "Picking"
    TEAMS_PLAYING DraftState = "Teams Playing"
    COMPLETE DraftState = "Complete"
)

type DraftModel struct {
    Id int
    DisplayName string
    Description string
    Interval int //Number of seconds to pick
    StartTime time.Time
    EndTime time.Time
    Owner User
    Status DraftState
    Players []DraftPlayer
    NextPick DraftPlayer
}

func (d *DraftModel) String() string {
	var stringBuilder strings.Builder
	for i, p := range d.Players {
		stringBuilder.WriteString("\nDraftPlayer - ")
		stringBuilder.WriteString(strconv.Itoa(i))
		stringBuilder.WriteString(" {\n")
		stringBuilder.WriteString(p.String())
		stringBuilder.WriteString(" \n}")
	}

	return fmt.Sprintf("Draft: {\nId: %d\n Displayname: %s\n Description: %s\n Interval: %d\n StartTime: %s\n EndTime: %s\n Owner: %s\n Status: %s\n Players: %s\n NextPick: %s\n}",
		d.Id, d.DisplayName, d.Description, d.Interval, d.StartTime.String(), d.EndTime.String(), d.Owner.String(), d.Status, stringBuilder.String(), d.NextPick.String())
}

type DraftPlayer struct {
    Id int
    User User
    PlayerOrder sql.NullInt16
    Pending bool
    Score int
    Picks []Pick
}

func (d *DraftPlayer) String() string {
	return fmt.Sprintf("DraftPlayer: {\nId: %d\n User: %s\n PlayerOrder: %d\n Pending: %t\n}", d.Id, d.User.UserUuid.String(), d.PlayerOrder, d.Pending)
}

type Pick struct {
    Id int
    Player int //DraftPlayerId
    Pick sql.NullString //TeamTbaId
    PickTime sql.NullTime
    AvailableTime time.Time
    ExpirationTime time.Time
    Skipped bool
    Score int
}

func (p *Pick) String() string {
	return fmt.Sprintf("Pick: {\nId: %d\n Player: %d\n Pick: %s\n PickTime: %s\n}", p.Id, p.Player, p.Pick.String, p.PickTime.Time.String())
}

type DraftInvite struct {
    Id int
    DraftId int //Draft
    DraftName string
    InvitedUserUuid uuid.UUID//User
    invitingUserUuid uuid.UUID //User
    InvitingPlayerName string
    SentTime time.Time
    AcceptedTime time.Time
    Accepted bool
}

func (d *DraftInvite) String() string {
	return fmt.Sprintf("DraftInvite: {\nId: %d\n DraftId: %d\n InvitingUserUuid: %s\n InvitedUserUuid: %s\n SentTime: %s\n AcceptedTime: %s\n Accepted: %t\n DraftName: %s\n InvitingPlayerName: %s\n}",
		d.Id, d.DraftId, d.invitingUserUuid.String(), d.InvitedUserUuid.String(), d.SentTime.String(), d.AcceptedTime.String(), d.Accepted, d.DraftName, d.InvitingPlayerName)
}

func GetDraftsByName(database *sql.DB, searchString string) *[]DraftModel {
    query := `SELECT
        Drafts.Id,
        DisplayName
    From Drafts
    Where DisplayName LIKE CONCAT('%', Cast($1 As varchar), '%');`
    assert := assert.CreateAssertWithContext("Get Drafts For User")
    assert.AddContext("Search", searchString)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    rows, err := stmt.Query(searchString)

    if err != nil {
        slog.Error("Failed to get drafts by name", "Search string", searchString, "Error", err)
        return nil
    }

    var drafts []DraftModel
    for rows.Next() {
        var draftId int
        var displayName string
        err = rows.Scan(&draftId, &displayName)

        if err != nil {
            slog.Error("Failed to get drafts by name", "Search string", searchString, "Error", err)
            return nil
        }

        draft := DraftModel {
            Id:          draftId,
            DisplayName: displayName,
        }

        drafts = append(drafts, draft)
    }

    return &drafts
}

func GetDraftsForUser(database *sql.DB, userUuid uuid.UUID) *[]DraftModel {
    query := `SELECT DISTINCT
        Drafts.Id,
        displayName,
        owners.UserUuid As ownerId,
        owners.Username As OwnerUsername,
        COALESCE(status, '0') As Status
    From Drafts
    Left Join DraftPlayers On DraftPlayers.DraftId = Drafts.Id
    Left Join DraftInvites On DraftInvites.DraftId = Drafts.Id And Drafts.Status = $1
    Left Join Users dpUsers On DraftPlayers.UserUuid = dpUsers.UserUuid
    Left Join Users diUsers On DraftInvites.InvitedUserUuid = diUsers.UserUuid
    Left Join Users owners On Drafts.OwnerUserUuid = owners.UserUuid
    Where DraftPlayers.UserUuid = $2 Or DraftInvites.InvitedUserUuid = $2;`

    outerAssert := assert.CreateAssertWithContext("Get Drafts For User")
    outerAssert.AddContext("User Uuid", userUuid)
    stmt, err := database.Prepare(query)
    outerAssert.NoError(err, "Failed to prepare statement")
    rows, err := stmt.Query(FILLING, userUuid)
    outerAssert.NoError(err, "Failed to get drafts for user")

    var drafts []DraftModel
    for rows.Next() {
        var draftId int
        var displayName string
        var ownerId uuid.UUID
        var ownerUsername string
        var status DraftState
        err = rows.Scan(&draftId, &displayName, &ownerId, &ownerUsername, &status)
        outerAssert.NoError(err, "Failed to load draft data")

        draftModel := DraftModel {
            Id:          draftId,
            DisplayName: displayName,
            Owner: User {
                UserUuid: ownerId,
                Username: ownerUsername,
            },
            Status: status,
            Players: make([]DraftPlayer, 0),
        }

		pick := Pick{}
        outerAssert.AddContext("Status", status)
		if status == PICKING {
            pick = GetCurrentPick(database, draftId)

            if pick.Id != 0 {
                draftModel.NextPick = DraftPlayer {
                    Id: pick.Player,
                    User: GetDraftPlayerUser(database, pick.Player),
                }
            }
        }

        playerQuery := `SELECT
	                    UserUuid,
	                    USERNAME,
	                    BOOL_OR(ACCEPTED) AS ACCEPTED
                    FROM (
		                    SELECT
			                    USERS.UserUuid AS UserUuid,
			                    USERS.USERNAME,
			                    't' AS ACCEPTED,
			                    DRAFTPLAYERS.PLAYERORDER,
			                    DraftPlayers.Id As PlayerId
		                    FROM USERS
		                    INNER JOIN DRAFTPLAYERS ON DRAFTPLAYERS.UserUuid = USERS.UserUuid
		                    WHERE DRAFTPLAYERS.DRAFTID = $1
		                    UNION
		                    SELECT
			                    USERS.UserUuid AS UserUuid,
			                    USERS.USERNAME,
			                    DRAFTINVITES.ACCEPTED AS ACCEPTED,
			                    -1 AS PLAYERORDER,
			                    -1 As PlayerId
		                    FROM USERS
		                    INNER JOIN DRAFTINVITES ON DRAFTINVITES.InvitedUserUuid = USERS.UserUuid
		                    WHERE DRAFTINVITES.DRAFTID = $1
	                    ) U
                    GROUP BY UserUuid, USERNAME
                    ORDER BY MAX(PLAYERORDER);`

        playerStmt, err := database.Prepare(playerQuery)
        outerAssert.NoError(err, "Failed to prepare player query")
        playerRows, err := playerStmt.Query(draftId)
        outerAssert.NoError(err, "Failed to get users for draft")

        for playerRows.Next() {
            var userUuid uuid.UUID
            var username string
            var accepted bool

            err = playerRows.Scan(&userUuid, &username, &accepted)
            outerAssert.NoError(err, "Failed to load player data")
            draftPlayer := DraftPlayer{
                User: User{
                    UserUuid: userUuid,
                    Username: username,
                },
                Pending: !accepted,
            }

            draftModel.Players = append(draftModel.Players, draftPlayer)
        }

        drafts = append(drafts, draftModel)
    }

    return &drafts
}

func CreateDraft(database *sql.DB, draft *DraftModel) int {
    query := `INSERT INTO Drafts (DisplayName, OwnerUserUuid, Description, StartTime,
        EndTime, Interval, Status) Values ($1, $2, $3, $4, $5, $6, $7) RETURNING Id;`
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
    err = stmt.QueryRow(draft.DisplayName, draft.Owner.UserUuid, draft.Description, draft.StartTime, draft.EndTime, draft.Interval, draft.Status).Scan(&draftId)
    assert.NoError(err, "Failed to insert draft")
    playerQuery := `INSERT INTO DraftPlayers (draftId, useruuid) Values ($1, $2);`
    stmt, err = database.Prepare(playerQuery)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(draftId, draft.Owner.UserUuid)
    assert.NoError(err, "Failed to insert draft")
    return draftId
}

func UpdateDraftStatus(database *sql.DB, draftId int, status DraftState) {
    query := `Update Drafts Set Status = $1 Where Id = $2;`
    assert := assert.CreateAssertWithContext("Update Draft Status")
    assert.AddContext("Draft Id", draftId)
    assert.AddContext("Status", status)

    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")

    _, err = stmt.Exec(status, draftId)
    if err != nil {
        slog.Error("Failed to update draft status", "Draft Id", draftId, "Status", status, "Error", err)
    }
}

func GetDraft(database *sql.DB, draftId int) (DraftModel, error) {
    query := `Select
        DisplayName,
        COALESCE(Description, '') As Description,
        COALESCE(Status, '') As Status,
        StartTime,
        EndTime,
        extract('epoch' from Interval)::int As Interval,
        OwnerUserUuid
    From Drafts Where Id = $1;`
    assert := assert.CreateAssertWithContext("Get Draft")
    assert.AddContext("Draft Id", draftId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    draftModel := DraftModel {
        Id: draftId,
    }
    var ownerId uuid.UUID
    err = stmt.QueryRow(draftId).Scan(
        &draftModel.DisplayName,
        &draftModel.Description,
        &draftModel.Status,
        &draftModel.StartTime,
        &draftModel.EndTime,
        &draftModel.Interval,
        &ownerId,
    )
    if err != nil {
        slog.Warn("Failed to load draft", "Draft Id", draftId)
        return DraftModel{}, errors.New("failed to load draft")
    }

	playerQuery := `SELECT
                        UserUuid,
	                    USERNAME,
	                    BOOL_OR(ACCEPTED) AS ACCEPTED,
	                    MAX(PLAYERORDER) AS PLAYERORDER,
	                    Max(PlayerId) As PlayerId
                    FROM (
		                    SELECT
			                    USERS.UserUuid AS UserUuid,
			                    USERS.USERNAME,
			                    't' AS ACCEPTED,
			                    DRAFTPLAYERS.PLAYERORDER,
			                    DraftPlayers.Id As PlayerId
		                    FROM USERS
		                    INNER JOIN DRAFTPLAYERS ON DRAFTPLAYERS.UserUuid = USERS.UserUuid
		                    WHERE DRAFTPLAYERS.DRAFTID = $1
		                    UNION
		                    SELECT
			                    USERS.UserUuid AS UserUuid,
			                    USERS.USERNAME,
			                    DRAFTINVITES.ACCEPTED AS ACCEPTED,
			                    -1 AS PLAYERORDER,
			                    -1 As PlayerId
		                    FROM USERS
		                    INNER JOIN DRAFTINVITES ON DRAFTINVITES.InvitedUserUuid = USERS.UserUuid
		                    WHERE DRAFTINVITES.DRAFTID = $1
	                    ) U
                    GROUP BY UserUuid, USERNAME
                    ORDER BY PLAYERORDER;`

	playerStmt, err := database.Prepare(playerQuery)
	assert.NoError(err, "Failed to prepare player query")
	playerRows, err := playerStmt.Query(draftId)
    if err != nil {
        slog.Warn("Failed to load players for draft", "Draft Id", draftId)
        return DraftModel{}, errors.New("failed to load draft")
    }

    slog.Info("Checking if we need to get the current pick for the draft", "Status", draftModel.Status, "Picking", PICKING)
    if draftModel.Status == PICKING {
        slog.Info("Getting the current pick for the draft")
        draftModel.NextPick = DraftPlayer {
            Id: GetCurrentPick(database, draftId).Player,
        }
    }

	for playerRows.Next() {
		var userUuid uuid.UUID
		var username string
		var accepted bool
		var playerOrder sql.NullInt16
		var playerId int

        err = playerRows.Scan(&userUuid, &username, &accepted, &playerOrder, &playerId)

        if err != nil {
            return DraftModel{}, err
        }

        if userUuid == ownerId {
            draftModel.Owner = User{
                UserUuid: userUuid,
                Username: username,
            }
        }

        if playerId == draftModel.NextPick.Id {
            draftModel.NextPick.User.UserUuid = userUuid
        }

        draftPlayer := DraftPlayer{
            Id: playerId,
            User: User{
                UserUuid: userUuid,
                Username: username,
            },
            PlayerOrder: playerOrder,
            Pending: !accepted,
        }

        //Get picks for the player
        picks := GetDraftPlayerPicks(database, draftPlayer.Id)
        draftPlayer.Picks = picks

		draftModel.Players = append(draftModel.Players, draftPlayer)
	}

	return draftModel, nil
}

func GetDraftPlayerPicks(database *sql.DB, draftPlayerId int) []Pick {
    query := `SELECT
                Picks.id,
                Picks.player,
                Picks.pick,
                Picks.pickTime,
                Picks.ExpirationTime
              From Picks
              Where Picks.player = $1
              Order By Picks.PickTime Asc;`

	assert := assert.CreateAssertWithContext("Get Picks")
	assert.AddContext("Draft Player Id", draftPlayerId)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")

	rows, err := stmt.Query(draftPlayerId)
	assert.NoError(err, "Failed to query for picks")

	var picks []Pick
	for rows.Next() {
		pick := Pick{}
        err = rows.Scan(&pick.Id, &pick.Player, &pick.Pick, &pick.PickTime, &pick.ExpirationTime)

        if err != nil {
            slog.Warn("Failed to get draft player picks", "Draft Player", draftPlayerId, "Error", err)
            return nil
        }

		picks = append(picks, pick)
	}

	return picks

}

func UpdateDraft(database *sql.DB, draft *DraftModel) error {
	query := `Update Drafts Set DisplayName = $1, Description = $2, StartTime = $3, EndTime = $4, Interval = $5 Where Id = $6;`
	assert := assert.CreateAssertWithContext("Update Draft")
	assert.AddContext("Display Name", draft.DisplayName)
	assert.AddContext("Interval", draft.Interval)
	assert.AddContext("Start Time", draft.StartTime)
	assert.AddContext("End Time", draft.EndTime)
	assert.AddContext("Description", draft.Description)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	_, err = stmt.Exec(draft.DisplayName, draft.Description, draft.StartTime, draft.EndTime, draft.Interval, draft.Id)
    return err
}

func InvitePlayer(database *sql.DB, draft int, invitingUserUuid uuid.UUID, invitedUserUuid uuid.UUID) int {
    query := `INSERT INTO DraftInvites (draftId, invitingUserUuid, invitedUserUuid,
    sentTime, accepted) Values ($1, $2, $3, $4, $5) RETURNING Id;`
	assert := assert.CreateAssertWithContext("Invite Player")
	assert.AddContext("Draft", draft)
	assert.AddContext("Inviting Inviting User Uuid", invitingUserUuid)
	assert.AddContext("Invited Invited User Uuid", invitedUserUuid)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	var inviteId int
	err = stmt.QueryRow(draft, invitingUserUuid, invitedUserUuid, time.Now(), false).Scan(&inviteId)
	assert.NoError(err, "Failed to insert invite player")
	return inviteId
}

// Returns draftId, UserUuid
func AcceptInvite(database *sql.DB, inviteId int) (int, uuid.UUID) {
	query := `UPDATE DraftInvites Set accepted = $1, acceptedTime = $2 where id = $3;`
	assert := assert.CreateAssertWithContext("Accept Invite")
	assert.AddContext("Invite Id", inviteId)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	_, err = stmt.Exec(true, time.Now(), inviteId)
	assert.NoError(err, "Failed to accept invite")

	query = `Select DraftId, InvitedUserUuid From DraftInvites Where Id = $1;`
	stmt, err = database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	var draftId int
	var userUuid uuid.UUID
	err = stmt.QueryRow(inviteId).Scan(&draftId, &userUuid)
	assert.NoError(err, "Failed to insert invite player")

	return draftId, userUuid
}

// TODO We should be able to uninvite someone from the draft

func AddPlayerToDraft(database *sql.DB, draft int, player uuid.UUID) {
	query := `INSERT INTO DraftPlayers (draftId, UserUuid) Values ($1, $2);`
	assert := assert.CreateAssertWithContext("Accept Invite")
	assert.AddContext("Draft", draft)
	assert.AddContext("Player", player)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	_, err = stmt.Exec(draft, player)
	assert.NoError(err, "Failed to accept invite")
}

func CancelOutstandingInvites(database *sql.DB, draftId int) {
	query := `Update DraftInvites Set Canceled = true Where DraftId = $1;`
	assert := assert.CreateAssertWithContext("Cancel Outstanding Invites")
	assert.AddContext("Draft Id", draftId)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	_, err = stmt.Exec(draftId)
	assert.NoError(err, "Failed to update drafts")
}

func GetInvite(database *sql.DB, inviteId int) DraftInvite {
	query := `SELECT
            di.Id,
            u.username,
            di.InvitedUserUuid,
            d.DisplayName,
            d.Id As DraftId
        From DraftInvites di
        Inner Join Drafts d On di.DraftId = d.Id
        Inner Join Users u On di.InvitingUserUuid = u.UserUuid
        Where di.Id = $1;`
    assert := assert.CreateAssertWithContext("Get Invite")
    assert.AddContext("Invite", inviteId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    invite := DraftInvite{}
    err = stmt.QueryRow(inviteId).Scan(
        &invite.Id,
        &invite.InvitingPlayerName,
        &invite.InvitedUserUuid,
        &invite.DraftName,
        &invite.DraftId)
    assert.NoError(err, "Failed to query invite")
    return invite
}

func GetInvites(database *sql.DB, userUuid uuid.UUID) []DraftInvite {
	query := `SELECT
            di.Id,
            u.username,
            d.DisplayName
        From DraftInvites di
        Inner Join Drafts d On di.DraftId = d.Id
        Inner Join Users u On di.InvitingUserUuid = u.UserUuid
        Where di.InvitedUserUuid = $1
        And di.Accepted = false;`
	assert := assert.CreateAssertWithContext("Get Invites")
	assert.AddContext("User Uuid", userUuid)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	rows, err := stmt.Query(userUuid)

    if err != nil {
        slog.Error("Failed to get invites", "User Uuid", userUuid, "Error", err)
        return nil
    }

	var invites []DraftInvite
	for rows.Next() {
		invite := DraftInvite{}
        err = rows.Scan(&invite.Id, &invite.InvitingPlayerName, &invite.DraftName)

        if err != nil {
            slog.Warn("Failed to get invite", "User Uuid", userUuid, "Error", err)
            continue
        }

		invites = append(invites, invite)
	}
	return invites
}

func GetPicks(database *sql.DB, draft int) ([]Pick, error) {
	query := `SELECT
        Picks.id, Picks.player, Picks.pick, Picks.pickTime, Picks.ExpirationTime
    From Picks
    Inner Join DraftPlayers On DraftPlayers.id = Picks.player
    Where DraftPlayers.draftId = $1
    Order By Picks.Id Asc;`

	assert := assert.CreateAssertWithContext("Get Picks")
	assert.AddContext("Draft", draft)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	rows, err := stmt.Query(draft)
	assert.NoError(err, "Failed to query for picks")

	var picks []Pick
	for rows.Next() {
		pick := Pick{}
		err = rows.Scan(&pick.Id, &pick.Player, &pick.Pick, &pick.PickTime, &pick.ExpirationTime)

        if err != nil {
            return nil, err
        }

		picks = append(picks, pick)
	}

	return picks, nil
}

func GetDraftPlayerId(database *sql.DB, draftId int, userUuid uuid.UUID) int {
    query := `Select Id From DraftPlayers Where draftId = $1 And userUuid = $2`

    assert := assert.CreateAssertWithContext("Get Draft Player Id")
    assert.AddContext("Draft Id", draftId)
    assert.AddContext("Player Id", userUuid)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")

    var draftPlayerId int
    err = stmt.QueryRow(draftId, userUuid).Scan(&draftPlayerId)
    assert.NoError(err, "Failed to get draft player")

    return draftPlayerId
}

func GetDraftPlayerUser(database *sql.DB, draftPlayerId int) User {
    query := `Select
        u.UserUuid,
        u.Username
    From DraftPlayers dp
    Inner Join Users u On dp.UserUuid = u.UserUuid
    Where dp.Id = $1;`

    assert := assert.CreateAssertWithContext("Get Draft Player User")
    assert.AddContext("Draft Player Id", draftPlayerId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")

    var user User
    err = stmt.QueryRow(draftPlayerId).Scan(&user.UserUuid, &user.Username)
    assert.NoError(err, "Failed to get User")

    return user
}

func MakePickAvailable(database *sql.DB, draftPlayerId int, availableTime time.Time, expirationTime time.Time) int {
    query := `Insert Into Picks (Player, AvailableTime, ExpirationTime) Values ($1, $2, $3) Returning Id;`

    assert := assert.CreateAssertWithContext("Make Pick Available")
    assert.AddContext("Draft Player Id", draftPlayerId)
    assert.AddContext("Available Time", availableTime)
    assert.AddContext("Expiration Time", expirationTime)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statment")

    var pickId int
    err = stmt.QueryRow(draftPlayerId, availableTime, expirationTime).Scan(&pickId)

    assert.NoError(err, "Failed to make pick available")

    return pickId
}

func MakePick(database *sql.DB, pick Pick) {
    query := `Update Picks Set pick = $1, pickTime = $2 Where Id = $3 Returning Id;`

    assert := assert.CreateAssertWithContext("Make Pick")
    assert.AddContext("Player", pick.Player)
    assert.AddContext("Team", pick.Pick)
    assert.AddContext("Pick Time", pick.PickTime)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var updatedId int
    err = stmt.QueryRow(pick.Pick, pick.PickTime, pick.Id).Scan(&updatedId)
    assert.RunAssert(pick.Id == updatedId, "Updated id does not match or no pick was updated")
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
        err = rows.Scan(&pick)

        if err != nil {
            slog.Error("Failed to get all picks", "Error", err)
            return nil
        }

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
	draftModel, err := GetDraft(database, draftId)
    if err != nil {
        slog.Warn("Attempting to randomize pick order for invalid draft", "Draft Id", draftId)
        return
    }
	awaitingAssignment := draftModel.Players
	assert := assert.CreateAssertWithContext("Randomize Pick Order")

    for i := range awaitingAssignment {
        j := rand.Intn(i + 1)
        awaitingAssignment[i], awaitingAssignment[j] = awaitingAssignment[j], awaitingAssignment[i]
	}

    for i, player := range awaitingAssignment {
		draftPlayerId := GetDraftPlayerId(database, draftId, player.User.UserUuid)
		query := `Update DraftPlayers Set PlayerOrder = $1 Where Id = $2`
		stmt, err := database.Prepare(query)
		assert.NoError(err, "Failed to prepare statement")
        _, err = stmt.Exec(i, draftPlayerId)
        if err != nil {
            slog.Warn("Failed to write pick order", "Draft Id", draftId, "Player", player.User.UserUuid , "Order", i, "Error", err)
        }
    }
}

func GetAvailablePickId (database *sql.DB, draftId int) Pick {
    query := `SELECT
        p.Id,
        p.Player
    From Picks p
    Inner Join (SELECT
        Max(Picks.Id) As Id
    From Picks
    Inner Join DraftPlayers On Picks.Player = DraftPlayers.Id
    Where Skipped = false
    And Pick Is Null
    And DraftPlayers.DraftId = $1) fp On p.Id = fp.Id;`

    assert := assert.CreateAssertWithContext("Get Available Picks Id")
    assert.AddContext("Draft Id", draftId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")

    var pick Pick
    err = stmt.QueryRow(draftId).Scan(&pick.Id, &pick.Player)
    assert.NoError(err, "Failed to get pick id")

    return pick
}

func NextPick(database *sql.DB, draftId int) DraftPlayer {
	//We need to get the last two picks
    assert := assert.CreateAssertWithContext("Next Pick")
	picks, err := GetPicks(database, draftId)

    if err != nil {
        slog.Warn("Failed to get picks", "Draft Id", draftId, "Error", err)
        return DraftPlayer{}
    }

	draft, err := GetDraft(database, draftId)
    if err != nil {
        slog.Warn("Attempting to find next pick for invalid draft", "Draft Id", draftId, "Error", err)
        return DraftPlayer{}
    }
	var nextPlayer DraftPlayer

    //I dont think we need to account for the case where there are only two players
    if len(picks) < 2 {
        for _, player := range draft.Players {
            assert.RunAssert(player.PlayerOrder.Valid, "Got player order which was not set when finding next pick")
            if int(player.PlayerOrder.Int16) == len(picks) {
                nextPlayer = player
            }
        }
    } else {
        //We can then figure out what direction
        //we are going and if we hit the
        //end then we decide what the next pick is
        lastPlayer := GetDraftPlayerFromDraft(draft, picks[len(picks)-1].Player)
        secondLastPick := GetDraftPlayerFromDraft(draft, picks[len(picks)-2].Player)
        assert.RunAssert(lastPlayer.PlayerOrder.Valid, "Got player order which was not set when finding next pick")
        direction := lastPlayer.PlayerOrder.Int16 - secondLastPick.PlayerOrder.Int16
        if lastPlayer.User.UserUuid == secondLastPick.User.UserUuid {
            if int(lastPlayer.PlayerOrder.Int16) == len(draft.Players) - 1 {
                direction = -1
            } else {
                direction = 1
            }
        }
        if len(picks) % 8 == 0 { //TODO Change to number of picks in draft
            direction = 0
        }
        assert.AddContext("Last Player Id", lastPlayer.Id)
        assert.AddContext("Second Last Player Id", secondLastPick.Id)
        assert.AddContext("Last Player Order", lastPlayer.PlayerOrder)
        assert.AddContext("Second Last Player Order", secondLastPick.PlayerOrder)
        assert.AddContext("Direction", direction)
        //We know draft.players is order by player order
        assert.RunAssert(int16(len(draft.Players)) > lastPlayer.PlayerOrder.Int16 + direction && lastPlayer.PlayerOrder.Int16 + direction >= 0, "Next pick is out of bounds")
        nextPlayer = draft.Players[lastPlayer.PlayerOrder.Int16 + direction]
    }

	//Take the pick and make it into a draft player
	return nextPlayer
}

func GetNumPlayersInInvitedDraft(database *sql.DB, inviteId int) int {
	query := `Select
                Count(*)
            From DraftInvites ci
            Inner Join Drafts d On d.Id = ci.DraftId
            Inner Join DraftPlayers dp On dp.DraftId = d.Id
            Where ci.Id = $1;`
	assert := assert.CreateAssertWithContext("Get Num Players In Invited Draft")
	assert.AddContext("InviteId", inviteId)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	var numPlayers int
	err = stmt.QueryRow(inviteId).Scan(&numPlayers)
	assert.NoError(err, "Failed to query for num players")
	return numPlayers
}

func GetDraftPlayerFromDraft(draft DraftModel, draftPlayerId int) DraftPlayer {
	for _, p := range draft.Players {
		if p.Id == draftPlayerId {
			return p
		}
	}
	return DraftPlayer{}
}

func StartDraft(database *sql.DB, draftId int) {
	query := `Update Drafts Set Status = $1 Where Id = $2;`

	assert := assert.CreateAssertWithContext("Start Draft")
	assert.AddContext("Draft Id", draftId)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	_, err = stmt.Exec(PICKING, draftId)
	assert.NoError(err, "Failed to update draft status")
}

func ShoudSkipPick(database *sql.DB, draftPlayer int) bool {
    assert := assert.CreateAssertWithContext("Shoud Skip Pick")
    assert.AddContext("Draft Player", draftPlayer)

    query := `SELECT
        COALESCE(skipPicks, false) As skipPicks
    From DraftPlayers dp
    Where dp.Id = $1;`

    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to preparte statement")
    var shoudSkip bool
    err = stmt.QueryRow(draftPlayer).Scan(&shoudSkip)

    if err != nil {
        slog.Warn("Failed to query if player should be skipped", "Player", draftPlayer, "Error", err)
        return false
    }

    return shoudSkip
}

func MarkShouldSkipPick(database *sql.DB, draftPlayer int, shouldSkip bool) error {
    assert := assert.CreateAssertWithContext("Mark Should Skip Pick")
    assert.AddContext("Draft Player", draftPlayer)

    query := `Update DraftPlayers Set skipPicks = $2 Where Id = $1;`

    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to preparte statement")
    _, err = stmt.Exec(draftPlayer, shouldSkip)

    return err
}

func GetCurrentPick(database *sql.DB, draftId int) Pick {
	query := `Select
                p.Id,
                p.Player,
                COALESCE(p.Pick, '') As Pick,
                p.PickTime,
                p.Skipped,
                p.AvailableTime,
                p.ExpirationTime At Time Zone 'America/New_York'
            From Picks p
            Inner Join (
                Select
	                Max(p.Id) As Id
                From Picks p
                Inner Join DraftPlayers dp On p.Player = dp.Id
                Where dp.DraftId = $1
            ) m On m.Id = p.Id;`

	assert := assert.CreateAssertWithContext("Get Current Pick")
	assert.AddContext("Draft Id", draftId)
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	var pick Pick
	err = stmt.QueryRow(draftId).Scan(
        &pick.Id,
		&pick.Player,
		&pick.Pick,
		&pick.PickTime,
		&pick.Skipped,
		&pick.AvailableTime,
        &pick.ExpirationTime,
    )

    if err != nil {
        //There is no current pick
        slog.Warn(err.Error())
        return Pick{}
    }

	return pick
}

func SkipPick(database *sql.DB, pickId int) {
    query := `Update Picks Set Skipped = true Where Id = $1`

    assert := assert.CreateAssertWithContext("Skip Pick")
    assert.AddContext("Pick Id", pickId)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(pickId)
    assert.NoError(err, "Failed to skip pick")
}

func GetDraftsInStatus(database *sql.DB, status DraftState) []int {
    assert := assert.CreateAssertWithContext("Get Drafts In Status")
    assert.AddContext("Status", status)

    query := `Select
        Id
    From Drafts
    Where Status = $1;`

    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")

    rows, err := stmt.Query(status)
    assert.NoError(err, "Failed to Query Drafts")

    var drafts []int
    for rows.Next() {
        var draftId int
        err = rows.Scan(&draftId)

        if err != nil {
            slog.Warn("Failed to load drafts in status", "Status", status, "Error", err)
            return nil
        }

        drafts = append(drafts, draftId)
    }

    return drafts
}


func GetDraftScore(database *sql.DB, draftId int) []DraftPlayer {
    assert := assert.CreateAssertWithContext("Get Draft Score")
    assert.AddContext("Draft Id", draftId)
    assert.RunAssert(draftId != 0, "Draft Id Should Not Be 0")

    query := `Select
        dp.Id,
        u.Username,
        p.Pick
    From Picks p
    Inner Join DraftPlayers dp On p.Player = dp.Id
    Inner Join Users u On u.UserUuid = dp.UserUuid
    Where dp.DraftId = $1;`

    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare query")

    rows, err := stmt.Query(draftId)
    assert.NoError(err, "Failed to get picks for draft")

    picks := make(map[int][]string)
    usernames := make(map[int]string)
    for rows.Next() {
        var playerId int
        var username string
        var pick string
        err = rows.Scan(&playerId, &username, &pick)

        if err != nil {
            slog.Warn("Failed to get draft scores", "Draft Id", draftId, "Error", err)
            return nil
        }

        usernames[playerId] = username
        picks[playerId] = append(picks[playerId], pick)
    }

    var playerScores []DraftPlayer
    for player, playerPicks := range picks {
        draftPlayer := DraftPlayer {
            Id: player,
            User: User{
                Username: usernames[player],
            },
            Score: 0,
        }

        for _, pick := range playerPicks {
            score := GetScore(database, pick)
            draftPlayer.Score += score["Total Score"]

            team := Pick {
                Pick: sql.NullString{
                    Valid: true,
                    String: pick,
                },
                Score: score["Total Score"],
            }
            draftPlayer.Picks = append(draftPlayer.Picks, team)
        }
        playerScores = append(playerScores, draftPlayer)
    }

    return playerScores
}
