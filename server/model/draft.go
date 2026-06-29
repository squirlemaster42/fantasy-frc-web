package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"server/assert"
	"server/log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type DraftState string

const (
	FILLING          DraftState = "Filling"
	WAITING_TO_START DraftState = "Waiting to Start"
	PICKING          DraftState = "Picking"
	TEAMS_PLAYING    DraftState = "Teams Playing"
	COMPLETE         DraftState = "Complete"
)

type DraftModel struct {
	Id             int
	DisplayName    string
	Description    string
	Interval       int //Number of seconds to pick
	DiscordWebhook string
	Owner          User
	Status         DraftState
	Players        []DraftPlayer
	NextPick       DraftPlayer
	CurrentPick    Pick
	Picks 		   []Pick
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

	return fmt.Sprintf("Draft: {\nId: %d\n Displayname: %s\n Description: %s\n Interval: %d\n Owner: %s\n Status: %s\n Players: %s\n NextPick: %s\n}",
		d.Id, d.DisplayName, d.Description, d.Interval, d.Owner.String(), d.Status, stringBuilder.String(), d.NextPick.String())
}

type DraftPlayer struct {
	Id          int
	User        User
	PlayerOrder sql.NullInt16
	Pending     bool
	Score       int
	Picks       []Pick
	InviteId    int
}

func (d *DraftPlayer) String() string {
	var stringBuilder strings.Builder
	for i, p := range d.Picks {
		stringBuilder.WriteString("\nPick - ")
		stringBuilder.WriteString(strconv.Itoa(i))
		stringBuilder.WriteString(" {\n")
		stringBuilder.WriteString(p.String())
		stringBuilder.WriteString(" \n}")
	}

	var playerOrderStr string
	if d.PlayerOrder.Valid {
		playerOrderStr = fmt.Sprintf("%d", d.PlayerOrder.Int16)
	} else {
		playerOrderStr = "NULL"
	}
	return fmt.Sprintf("DraftPlayer: {\nId: %d\n User: %s\n PlayerOrder: %s\n Pending: %t\n Picks: %s\n}",
		d.Id, d.User.UserUuid.String(), playerOrderStr, d.Pending, stringBuilder.String())
}

type Pick struct {
	Id             int
	Player         int            //DraftPlayerId
	Pick           sql.NullString //TeamTbaId
	PickTime       sql.NullTime
	AvailableTime  time.Time
	ExpirationTime time.Time
	Skipped        bool
	Score          int
}

func (p *Pick) String() string {
	pickStr := "NULL"
	if p.Pick.Valid {
		pickStr = p.Pick.String
	}
	pickTimeStr := "NULL"
	if p.PickTime.Valid {
		pickTimeStr = p.PickTime.Time.String()
	}
	return fmt.Sprintf("Pick: {\nId: %d\n Player: %d\n Pick: %s\n PickTime: %s\n Skipped: %t\n AvailableTime: %s\n ExpirationTime: %s\n Score: %d\n}",
		p.Id, p.Player, pickStr, pickTimeStr, p.Skipped, p.AvailableTime.String(), p.ExpirationTime.String(), p.Score)
}

type DraftInvite struct {
	Id                 int
	DraftId            int //Draft
	DraftName          string
	InvitedUserUuid    uuid.UUID //User
	InvitingUserUuid   uuid.UUID //User
	InvitingPlayerName string
	InvitedPlayerName  string
	SentTime           time.Time
	AcceptedTime       time.Time
	Accepted           bool
}

func (d *DraftInvite) String() string {
	return fmt.Sprintf("DraftInvite: {\nId: %d\n DraftId: %d\n InvitingUserUuid: %s\n InvitedUserUuid: %s\n SentTime: %s\n AcceptedTime: %s\n Accepted: %t\n DraftName: %s\n InvitingPlayerName: %s\n InvitedPlayerName: %s\n}",
		d.Id, d.DraftId, d.InvitingUserUuid.String(), d.InvitedUserUuid.String(), d.SentTime.String(), d.AcceptedTime.String(), d.Accepted, d.DraftName, d.InvitingPlayerName, d.InvitedPlayerName)
}

func getDraftsByName(ctx context.Context, database *sql.DB, searchString string) ([]DraftModel, error) {
	query := `SELECT
        Drafts.Id,
        DisplayName
    From Drafts
    Where DisplayName LIKE CONCAT('%', Cast($1 As varchar), '%');`
	assert := assert.CreateAssertWithContext("Get Drafts For User")
	assert.AddContext("Search", searchString)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		err := stmt.Close()
		if err != nil {
			log.Error(ctx, "GetDraftsByName: Failed to close statement", "error", err)
		}
	}()
	rows, err := stmt.QueryContext(ctx, searchString)

	if err != nil {
		return nil, fmt.Errorf("failed to get drafts by name: %w", err)
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			log.Error(ctx, "GetDraftsByName: Failed to close rows", "error", err)
		}
	}()
	var drafts []DraftModel
	for rows.Next() {
		var draftId int
		var displayName string
		err = rows.Scan(&draftId, &displayName)

		if err != nil {
			return nil, fmt.Errorf("failed to scan draft: %w", err)
		}

		draft := DraftModel{
			Id:          draftId,
			DisplayName: displayName,
		}

		drafts = append(drafts, draft)
	}

	return drafts, nil
}

func getDraftsForUser(ctx context.Context, database *sql.DB, userUuid uuid.UUID) ([]DraftModel, error) {
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
	Left Join Users currUser On currUser.UserUuid = $2
    Where DraftPlayers.UserUuid = $2
		Or DraftInvites.InvitedUserUuid = $2
		Or currUser.IsAdmin = true
    Order By Drafts.Id Asc;`

	assert := assert.CreateAssertWithContext("Get Drafts For User")
	assert.AddContext("userUuid", userUuid)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		err := stmt.Close()
		if err != nil {
			log.Error(ctx, "GetDraftsForUser: Failed to close statement", "error", err)
		}
	}()
	rows, err := stmt.QueryContext(ctx, FILLING, userUuid)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := rows.Close()
		if err != nil {
			log.Error(ctx, "GetDraftsForUser: Failed to close rows", "error", err)
		}
	}()
	var drafts []DraftModel

	playerQuery := `SELECT
	                    UserUuid,
	                    USERNAME,
	                    BOOL_OR(ACCEPTED) AS ACCEPTED
                    FROM (
	                    SELECT
		                    USERS.UserUuid AS UserUuid,
		                    USERS.USERNAME,
		                    't' AS ACCEPTED,
							'f' AS CANCELED,
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
							DRAFTINVITES.CANCELED AS CANCELED,
		                    -1 AS PLAYERORDER,
		                    -1 As PlayerId
	                    FROM USERS
	                    INNER JOIN DRAFTINVITES ON DRAFTINVITES.InvitedUserUuid = USERS.UserUuid
	                    WHERE DRAFTINVITES.DRAFTID = $1 AND COALESCE(CANCELED, FALSE) = FALSE
                    ) U
                    GROUP BY UserUuid, USERNAME
                    ORDER BY MAX(PLAYERORDER);`

	playerStmt, err := database.PrepareContext(ctx, playerQuery)
	assert.NoError(ctx, err, "Failed to prepare player statement")
	defer func() {
		if err := playerStmt.Close(); err != nil {
			log.Error(ctx, "GetDraftsForUser: Failed to close statement", "error", err)
		}
	}()

	for rows.Next() {
		var draftId int
		var displayName string
		var ownerId uuid.UUID
		var ownerUsername string
		var status DraftState
		err = rows.Scan(&draftId, &displayName, &ownerId, &ownerUsername, &status)
		if err != nil {
			return nil, err
		}

		draftModel := DraftModel{
			Id:          draftId,
			DisplayName: displayName,
			Owner: User{
				UserUuid: ownerId,
				Username: ownerUsername,
			},
			Status:  status,
			Players: make([]DraftPlayer, 0),
		}

		pick := Pick{}
		if status == PICKING {
			pick, err = getCurrentPick(ctx, database, draftId)
			if err != nil {
				return []DraftModel{}, err
			}

			if pick.Id != 0 {
				user, err := getDraftPlayerUser(ctx, database, pick.Player)
				if err != nil {
					return []DraftModel{}, err
				}

				draftModel.NextPick = DraftPlayer{
					Id:   pick.Player,
					User: user,
				}
			}
		}

		playerRows, err := playerStmt.QueryContext(ctx, draftId)
		if err != nil {
			return nil, err
		}

		for playerRows.Next() {
			var userUuid uuid.UUID
			var username string
			var accepted bool

			err = playerRows.Scan(&userUuid, &username, &accepted)
			if err != nil {
				playerRows.Close()
				return nil, err
			}
			draftPlayer := DraftPlayer{
				User: User{
					UserUuid: userUuid,
					Username: username,
				},
				Pending: !accepted,
			}

			draftModel.Players = append(draftModel.Players, draftPlayer)
		}
		playerRows.Close()

		drafts = append(drafts, draftModel)
	}

	return drafts, nil
}

func createDraft(ctx context.Context, database *sql.DB, draft *DraftModel) (int, error) {
	query := `INSERT INTO Drafts (DisplayName, OwnerUserUuid, Description, Status) Values ($1, $2, $3, $4) RETURNING Id;`
	assert := assert.CreateAssertWithContext("Create Draft")
	assert.AddContext("Owner", draft.Owner)
	assert.AddContext("Display Name", draft.DisplayName)
	assert.AddContext("Interval", draft.Interval)
	assert.AddContext("statusCode", draft.Status)
	assert.AddContext("Description", draft.Description)
	assert.RunAssert(ctx, draft.Owner.UserUuid != uuid.Nil, "Draft owner uuid is nil")
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "CreateDraft: Failed to close statement", "error", err)
		}
	}()
	var draftId int
	err = stmt.QueryRowContext(ctx, draft.DisplayName, draft.Owner.UserUuid, draft.Description, draft.Status).Scan(&draftId)
	if err != nil {
		return -1, err
	}
	playerQuery := `INSERT INTO DraftPlayers (draftId, useruuid) Values ($1, $2);`
	stmt, err = database.PrepareContext(ctx, playerQuery)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "CreateDraft: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, draftId, draft.Owner.UserUuid)
	if err != nil {
		return -1, err
	}
	log.Info(ctx, "Created draft", "draftId", draftId, "ownerUuid", draft.Owner.UserUuid)
	return draftId, nil
}

func updateDraftStatus(ctx context.Context, database *sql.DB, draftId int, status DraftState) error {
	assert := assert.CreateAssertWithContext("Update Draft Status")
	query := `Update Drafts Set Status = $1 Where Id = $2;`

	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "UpdateDraftStatus: Failed to close statement", "error", err)
		}
	}()

	_, err = stmt.ExecContext(ctx, status, draftId)
	if err != nil {
		log.Error(ctx, "Failed to update draft status", "draftId", draftId, "statusCode", status, "error", err)
		return err
	}
	return nil
}

func getDraft(ctx context.Context, database *sql.DB, draftId int) (DraftModel, error) {
	log.Debug(ctx, "model.GetDraft: starting", "draftId", draftId)
	query := `Select
        DisplayName,
        COALESCE(Description, '') As Description,
        COALESCE(Status, '') As Status,
        OwnerUserUuid,
		COALESCE(DiscordWebhook, '')
    From Drafts Where Id = $1;`

	assert := assert.CreateAssertWithContext("Get Draft")
	assert.AddContext("draftId", draftId)

	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare draft statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetDraft: Failed to close statement", "error", err)
		}
	}()
	log.Debug(ctx, "model.GetDraft: executing query", "draftId", draftId)
	draftModel := DraftModel{
		Id: draftId,
	}
	var ownerId uuid.UUID
	err = stmt.QueryRowContext(ctx, draftId).Scan(
		&draftModel.DisplayName,
		&draftModel.Description,
		&draftModel.Status,
		&ownerId,
		&draftModel.DiscordWebhook,
	)
	log.Debug(ctx, "model.GetDraft: query completed", "draftId", draftId)
	if err != nil {
		log.Error(ctx, "Failed to load draft", "draftId", draftId, "error", err)
		return DraftModel{}, errors.New("failed to load draft")
	}

	// Get Current Pick
	currentPick, err := getCurrentPick(ctx, database, draftId)
	if err != nil {
		log.Error(ctx, "Failed to get current pick for draft", "draftId", draftId, "error", err)
	} else {
        draftModel.CurrentPick = currentPick
    }

	picks, err := getPicks(ctx, database, draftId)
	if err != nil {
		log.Error(ctx, "Failed to get picks for draft", "draftId", draftId, "error", err)
		return DraftModel{}, errors.New("failed to get picks for draft")
	}
	draftModel.Picks = picks

	playerQuery := `SELECT
                        UserUuid,
	                    USERNAME,
	                    BOOL_OR(ACCEPTED) AS ACCEPTED,
	                    MAX(PLAYERORDER) AS PLAYERORDER,
	                    Max(PlayerId) As PlayerId,
	                    MAX(InviteId) As InviteId
                    FROM (
		                    SELECT
			                    USERS.UserUuid AS UserUuid,
			                    USERS.USERNAME,
			                    't' AS ACCEPTED,
			                    COALESCE(DRAFTPLAYERS.PLAYERORDER, -1) As PLAYERORDER,
			                    DraftPlayers.Id As PlayerId,
			                    -1 As InviteId
		                    FROM USERS
		                    INNER JOIN DRAFTPLAYERS ON DRAFTPLAYERS.UserUuid = USERS.UserUuid
		                    WHERE DRAFTPLAYERS.DRAFTID = $1
		                    UNION
		                    SELECT
			                    USERS.UserUuid AS UserUuid,
			                    USERS.USERNAME,
			                    DRAFTINVITES.ACCEPTED AS ACCEPTED,
			                    -1 AS PLAYERORDER,
			                    -1 As PlayerId,
			                    DraftInvites.Id As InviteId
		                    FROM USERS
		                    INNER JOIN DRAFTINVITES ON DRAFTINVITES.InvitedUserUuid = USERS.UserUuid
		                    WHERE DRAFTINVITES.DRAFTID = $1 AND COALESCE(DRAFTINVITES.CANCELED, FALSE) = FALSE
	                    ) U
                    GROUP BY UserUuid, USERNAME
                    ORDER BY PLAYERORDER;`

	playerStmt, err := database.PrepareContext(ctx, playerQuery)
	assert.NoError(ctx, err, "Failed to prepare player statement")
	defer func() {
		if err := playerStmt.Close(); err != nil {
			log.Error(ctx, "GetDraft: Failed to close statement", "error", err)
		}
	}()
	playerRows, err := playerStmt.QueryContext(ctx, draftId)
	if err != nil {
		log.Error(ctx, "Failed to load players for draft", "draftId", draftId, "error", err)
		return DraftModel{}, errors.New("failed to load draft")
	}
	defer func() {
		if err := playerRows.Close(); err != nil {
			log.Error(ctx, "GetDraft: Failed to close rows", "error", err)
		}
	}()

	log.Debug(ctx, "Checking if we need to get the current pick for the draft", "statusCode", draftModel.Status, "picking", PICKING)
	if draftModel.Status == PICKING {
		log.Debug(ctx, "Getting the current pick for the draft")
		currPick, err := getCurrentPick(ctx, database, draftId)
		if err != nil {
			return DraftModel{}, err
		}
		draftModel.NextPick = DraftPlayer{
			Id: currPick.Player,
		}
	}

	for playerRows.Next() {
		var userUuid uuid.UUID
		var username string
		var accepted bool
		var playerOrder sql.NullInt16
		var playerId int
		var inviteId int

		err = playerRows.Scan(&userUuid, &username, &accepted, &playerOrder, &playerId, &inviteId)

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
			draftModel.NextPick.User.Username = username
		}

		draftPlayer := DraftPlayer{
			Id: playerId,
			User: User{
				UserUuid: userUuid,
				Username: username,
			},
			PlayerOrder: playerOrder,
			Pending:     !accepted,
			InviteId:    inviteId,
		}

		//Get picks for the player
		picks := getDraftPlayerPicks(ctx, database, draftPlayer.Id)
		draftPlayer.Picks = picks

		draftModel.Players = append(draftModel.Players, draftPlayer)
	}

	return draftModel, nil
}

func getDraftPlayerPicks(ctx context.Context, database *sql.DB, draftPlayerId int) []Pick {
	query := `SELECT
                Picks.id,
                Picks.player,
                Picks.pick,
                Picks.pickTime,
                Picks.ExpirationTime,
				Picks.Skipped
              From Picks
              Where Picks.player = $1
              Order By Picks.AvailableTime Asc;`

	assert := assert.CreateAssertWithContext("Get Picks")
	assert.AddContext("draftPlayerId", draftPlayerId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetDraftPlayerPicks: Failed to close statement", "error", err)
		}
	}()

	rows, err := stmt.QueryContext(ctx, draftPlayerId)
	assert.NoError(ctx, err, "Failed to query for picks")
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error(ctx, "GetDraftPlayerPicks: Failed to close rows", "error", err)
		}
	}()

	var picks []Pick
	for rows.Next() {
		pick := Pick{}
		err = rows.Scan(&pick.Id, &pick.Player, &pick.Pick, &pick.PickTime, &pick.ExpirationTime, &pick.Skipped)

		if err != nil {
			log.Error(ctx, "Failed to get draft player picks", "draftPlayerId", draftPlayerId, "error", err)
			return nil
		}

		picks = append(picks, pick)
	}

	return picks
}

func updateDraft(ctx context.Context, database *sql.DB, draft *DraftModel) error {
	log.Debug(ctx, "model.UpdateDraft: starting", "draftId", draft.Id)
	query := `Update Drafts Set DisplayName = $1, Description = $2, Interval = $3, DiscordWebhook = $4 Where Id = $5;`
	assert := assert.CreateAssertWithContext("Update Draft")
	assert.AddContext("Display Name", draft.DisplayName)
	assert.AddContext("Interval", draft.Interval)
	assert.AddContext("Description", draft.Description)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "UpdateDraft: Failed to close statement", "error", err)
		}
	}()
	log.Debug(ctx, "model.UpdateDraft: executing query", "draftId", draft.Id)
	_, err = stmt.ExecContext(ctx, draft.DisplayName, draft.Description, draft.Interval, draft.DiscordWebhook, draft.Id)
	log.Debug(ctx, "model.UpdateDraft: query completed", "draftId", draft.Id)
	if err == nil {
		log.Info(ctx, "Updated draft", "draftId", draft.Id)
	}
	return err
}

func invitePlayer(ctx context.Context, database *sql.DB, draft int, invitingUserUuid uuid.UUID, invitedUserUuid uuid.UUID) (int, error) {
	assert := assert.CreateAssertWithContext("Invite Player")
	query := `INSERT INTO DraftInvites (draftId, invitingUserUuid, invitedUserUuid,
    sentTime, accepted) Values ($1, $2, $3, $4, $5) RETURNING Id;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "InvitePlayer: Failed to close statement", "error", err)
		}
	}()

	var inviteId int
	err = stmt.QueryRowContext(ctx, draft, invitingUserUuid, invitedUserUuid, time.Now().UTC(), false).Scan(&inviteId)
	if err != nil {
		return -1, err
	}
	log.Info(ctx, "Invited player to draft", "draftId", draft, "invitedUserUuid", invitedUserUuid, "inviteId", inviteId)
	return inviteId, nil
}

// Returns draftId, UserUuid, error
func acceptInvite(ctx context.Context, database *sql.DB, inviteId int) (int, uuid.UUID, error) {
	query := `UPDATE DraftInvites Set accepted = $1, acceptedTime = $2 where id = $3;`
	assert := assert.CreateAssertWithContext("Accept Invite")
	assert.AddContext("Invite Id", inviteId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare draft statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "AcceptInvite: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, true, time.Now().UTC(), inviteId)
	if err != nil {
		return 0, uuid.UUID{}, fmt.Errorf("failed to accept invite: %w", err)
	}

	query = `Select DraftId, InvitedUserUuid From DraftInvites Where Id = $1;`
	stmt, err = database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare draft statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "AcceptInvite: Failed to close statement", "error", err)
		}
	}()
	var draftId int
	var userUuid uuid.UUID
	err = stmt.QueryRowContext(ctx, inviteId).Scan(&draftId, &userUuid)
	if err != nil {
		return 0, uuid.UUID{}, fmt.Errorf("failed to get invite details: %w", err)
	}

	log.Info(ctx, "Accepted invite", "inviteId", inviteId, "draftId", draftId, "userUuid", userUuid)
	return draftId, userUuid, nil
}

func addPlayerToDraft(ctx context.Context, database *sql.DB, draft int, player uuid.UUID) error {
	query := `INSERT INTO DraftPlayers (draftId, UserUuid) Values ($1, $2);`
	assert := assert.CreateAssertWithContext("Accept Invite")
	assert.AddContext("Draft", draft)
	assert.AddContext("player", player)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "AddPlayerToDraft: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, draft, player)
	if err != nil {
		return fmt.Errorf("failed to add player to draft: %w", err)
	}
	return nil
}

func cancelOutstandingInvites(ctx context.Context, database *sql.DB, draftId int) error {
	query := `Update DraftInvites Set Canceled = true Where DraftId = $1 and Accepted = false;`
	assert := assert.CreateAssertWithContext("Cancel Outstanding Invites")
	assert.AddContext("draftId", draftId)

	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")

	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "CancelOutstandingInvites: Failed to close statement", "error", err)
		}
	}()

	_, err = stmt.ExecContext(ctx, draftId)
	if err != nil {
		return fmt.Errorf("failed to cancel invites for draft %d: %w", draftId, err)
	}

	return nil
}

func getInvite(ctx context.Context, database *sql.DB, inviteId int) (DraftInvite, error) {
	assert := assert.CreateAssertWithContext("Get Invite")
	assert.AddContext("Invite Id", inviteId)
	query := `SELECT
            di.Id,
            u.username,
            di.InvitedUserUuid,
            d.DisplayName,
            d.Id As DraftId
        From DraftInvites di
        Inner Join Drafts d On di.DraftId = d.Id
        Inner Join Users u On di.InvitingUserUuid = u.UserUuid
        Where di.Id = $1
        And COALESCE(di.Canceled, false) = false;`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetInvite: Failed to close statement", "error", err)
		}
	}()
	invite := DraftInvite{}
	err = stmt.QueryRowContext(ctx, inviteId).Scan(
		&invite.Id,
		&invite.InvitingPlayerName,
		&invite.InvitedUserUuid,
		&invite.DraftName,
		&invite.DraftId)
	if err != nil {
		log.Error(ctx, "GetInvite: Failed to query invite", "error", err, "inviteId", inviteId)
		return DraftInvite{}, err
	}
	return invite, nil
}

func getInvites(ctx context.Context, database *sql.DB, userUuid uuid.UUID) ([]DraftInvite, error) {
	query := `SELECT
            di.Id,
            u.username,
            d.DisplayName
        From DraftInvites di
        Inner Join Drafts d On di.DraftId = d.Id
        Inner Join Users u On di.InvitingUserUuid = u.UserUuid
        Where di.InvitedUserUuid = $1
        And di.Accepted = false
        And COALESCE(di.Canceled, false) = false;`
	assert := assert.CreateAssertWithContext("Get Invites")
	assert.AddContext("userUuid", userUuid)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetInvites: Failed to close statement", "error", err)
		}
	}()
	rows, err := stmt.QueryContext(ctx, userUuid)

	if err != nil {
		return nil, fmt.Errorf("failed to get invites: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error(ctx, "GetInvites: Failed to close rows", "error", err)
		}
	}()

	var invites []DraftInvite
	for rows.Next() {
		invite := DraftInvite{}
		err = rows.Scan(&invite.Id, &invite.InvitingPlayerName, &invite.DraftName)

		if err != nil {
			return nil, fmt.Errorf("failed to scan invite: %w", err)
		}

		invites = append(invites, invite)
	}
	return invites, nil
}

func cancelInvite(ctx context.Context, database *sql.DB, inviteId int) error {
	query := `Update DraftInvites Set Canceled = true Where Id = $1;`
	assert := assert.CreateAssertWithContext("Cancel Invite")
	assert.AddContext("inviteId", inviteId)

	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")

	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "CancelInvite: Failed to close statement", "error", err)
		}
	}()

	_, err = stmt.ExecContext(ctx, inviteId)
	if err != nil {
		return fmt.Errorf("failed to cancel invite %d: %w", inviteId, err)
	}

	return nil
}

func uninvitePlayer(ctx context.Context, database *sql.DB, draftId int, ownerUuid uuid.UUID, inviteId int) error {
	ownerQuery := `Select OwnerUserUuid From Drafts Where Id = $1;`
	ownerStmt, err := database.PrepareContext(ctx, ownerQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare owner query: %w", err)
	}

	var dbOwnerUuid string
	err = ownerStmt.QueryRowContext(ctx, draftId).Scan(&dbOwnerUuid)
	ownerStmt.Close()
	if err != nil {
		return fmt.Errorf("failed to get draft owner: %w", err)
	}

	if dbOwnerUuid != ownerUuid.String() {
		return fmt.Errorf("user %s is not the owner of draft %d", ownerUuid, draftId)
	}

	query := `Update DraftInvites Set Canceled = true Where Id = $1 And DraftId = $2;`
	stmt, err := database.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "UninvitePlayer: Failed to close statement", "error", err)
		}
	}()

	result, err := stmt.ExecContext(ctx, inviteId, draftId)
	if err != nil {
		return fmt.Errorf("failed to uninvite player: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("invite %d not found for draft %d", inviteId, draftId)
	}

	log.Info(ctx, "Uninvited player from draft", "draftId", draftId, "inviteId", inviteId)
	return nil
}

func getOutstandingInvitesForDraft(ctx context.Context, database *sql.DB, draftId int) ([]DraftInvite, error) {
	query := `SELECT
		di.Id,
		u.username,
		di.InvitedUserUuid
	From DraftInvites di
	Inner Join Users u On di.InvitedUserUuid = u.UserUuid
	Where di.DraftId = $1
	And di.Accepted = false
	And COALESCE(di.Canceled, false) = false;`

	stmt, err := database.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetOutstandingInvitesForDraft: Failed to close statement", "error", err)
		}
	}()

	rows, err := stmt.QueryContext(ctx, draftId)
	if err != nil {
		return nil, fmt.Errorf("failed to get outstanding invites: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error(ctx, "GetOutstandingInvitesForDraft: Failed to close rows", "error", err)
		}
	}()

	var invites []DraftInvite
	for rows.Next() {
		invite := DraftInvite{}
		err = rows.Scan(&invite.Id, &invite.InvitedPlayerName, &invite.InvitedUserUuid)
		if err != nil {
			return nil, fmt.Errorf("failed to scan invite: %w", err)
		}
		invites = append(invites, invite)
	}

	return invites, nil
}

func getPicks(ctx context.Context, database *sql.DB, draftId int) ([]Pick, error) {
	query := `SELECT
        Picks.id, Picks.player, Picks.pick, Picks.pickTime, Picks.ExpirationTime
    From Picks
    Inner Join DraftPlayers On DraftPlayers.id = Picks.player
    Where DraftPlayers.draftId = $1
    Order By Picks.Id Asc;`

	assert := assert.CreateAssertWithContext("Get Picks")
	assert.AddContext("Draft", draftId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetPicks: Failed to close statement", "error", err)
		}
	}()
	rows, err := stmt.QueryContext(ctx, draftId)
	assert.NoError(ctx, err, "Failed to query for picks")
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error(ctx, "GetPicks: Failed to close rows", "error", err)
		}
	}()

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

func getDraftPlayerId(ctx context.Context, database *sql.DB, draftId int, userUuid uuid.UUID) (int, error) {
	query := `Select Id From DraftPlayers Where draftId = $1 And userUuid = $2`

	assert := assert.CreateAssertWithContext("Get Draft Player Id")
	assert.AddContext("draftId", draftId)
	assert.AddContext("Player Id", userUuid)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetDraftPlayerId: Failed to close statement", "error", err)
		}
	}()

	var draftPlayerId int
	err = stmt.QueryRowContext(ctx, draftId, userUuid).Scan(&draftPlayerId)

	if err != nil {
		return -1, errors.Join(fmt.Errorf("failed to get draft player for user %s in draft %d", userUuid.String(), draftId), err)
	}

	return draftPlayerId, nil
}

func getDraftPlayerUser(ctx context.Context, database *sql.DB, draftPlayerId int) (User, error) {
	query := `Select
        u.UserUuid,
        u.Username
    From DraftPlayers dp
    Inner Join Users u On dp.UserUuid = u.UserUuid
    Where dp.Id = $1;`

	assert := assert.CreateAssertWithContext("Get Draft Player User")
	assert.AddContext("draftPlayerId", draftPlayerId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetDraftPlayerUser: Failed to close statement", "error", err)
		}
	}()

	var user User
	err = stmt.QueryRowContext(ctx, draftPlayerId).Scan(&user.UserUuid, &user.Username)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func makePickAvailable(ctx context.Context, database *sql.DB, draftPlayerId int, availableTime time.Time, expirationTime time.Time) (int, error) {
	query := `Insert Into Picks (Player, AvailableTime, ExpirationTime) Values ($1, $2, $3) Returning Id;`

	assert := assert.CreateAssertWithContext("Make Pick Available")
	assert.AddContext("draftPlayerId", draftPlayerId)
	assert.AddContext("Available Time", availableTime)
	assert.AddContext("Expiration Time", expirationTime)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "MakePickAvailable: Failed to close statement", "error", err)
		}
	}()

	var pickId int
	err = stmt.QueryRowContext(ctx, draftPlayerId, availableTime, expirationTime).Scan(&pickId)

	if err != nil {
		log.Error(ctx, "Failed to make pick available", "draftPlayerId", draftPlayerId, "error", err)
		return 0, err
	}

	return pickId, nil
}

func makePick(ctx context.Context, database *sql.DB, pick Pick) error {
	query := `Update Picks Set pick = $1, pickTime = $2 Where Id = $3 Returning Id;`

	assert := assert.CreateAssertWithContext("Make Pick")
	assert.AddContext("player", pick.Player)
	assert.AddContext("team", pick.Pick)
	assert.AddContext("Pick Time", pick.PickTime)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "MakePick: Failed to close statement", "error", err)
		}
	}()
	var updatedId int
	err = stmt.QueryRowContext(ctx, pick.Pick, pick.PickTime, pick.Id).Scan(&updatedId)
	if err != nil {
		log.Error(ctx, "Failed to make pick", "error", err)
		return err
	}
	assert.RunAssert(ctx, pick.Id == updatedId, "Updated id does not match or no pick was updated")
	log.Info(ctx, "Made pick", "pickId", pick.Id, "team", pick.Pick.String, "player", pick.Player)
	return nil
}

func hasBeenPicked(ctx context.Context, database *sql.DB, draftId int, team string) (bool, error) {
	query := `SELECT
    Count(*) As num
    From Picks
    Inner Join DraftPlayers On DraftPlayers.id = Picks.player
    Where DraftPlayers.draftId = $1
    And Picks.pick = $2;`
	assert := assert.CreateAssertWithContext("Has Been Picked")
	assert.AddContext("Draft", draftId)
	assert.AddContext("team", team)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "HasBeenPicked: Failed to close statement", "error", err)
		}
	}()
	var numPicked int
	err = stmt.QueryRowContext(ctx, draftId, team).Scan(&numPicked)
	if err != nil {
		log.Error(ctx, "Failed to query for picks", "draftId", draftId, "team", team, "error", err)
		return false, err
	}
	return numPicked != 0, nil
}

func randomizePickOrder(ctx context.Context, database *sql.DB, draftId int) error {
	draftModel, err := getDraft(ctx, database, draftId)
	if err != nil {
		log.Warn(ctx, "Attempting to randomize pick order for invalid draft", "draftId", draftId)
		return fmt.Errorf("could not load draft %d", draftId)
	}
	var awaitingAssignment []DraftPlayer
	// We only want to randomize the pick order of players who accepted the draft
	for _, player := range draftModel.Players {
		if !player.Pending {
			awaitingAssignment = append(awaitingAssignment, player)
		}
	}
	assert := assert.CreateAssertWithContext("Randomize Pick Order")

	for i := range awaitingAssignment {
		j := rand.Intn(i + 1)
		awaitingAssignment[i], awaitingAssignment[j] = awaitingAssignment[j], awaitingAssignment[i]
	}

	query := `Update DraftPlayers Set PlayerOrder = $1 Where Id = $2`
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "RandomizePickOrder: Failed to close statement", "error", err)
		}
	}()

	for i, player := range awaitingAssignment {
		draftPlayerId, err := getDraftPlayerId(ctx, database, draftId, player.User.UserUuid)
		if err != nil {
			return fmt.Errorf("could not get draftplayer for user %s in draft %d", player.User.UserUuid.String(), draftId)
		}
		_, err = stmt.ExecContext(ctx, i, draftPlayerId)
		if err != nil {
			log.Error(ctx, "Failed to write pick order", "draftId", draftId, "player", player.User.UserUuid, "order", i, "error", err)
		}
	}

	return nil
}

func nextPick(ctx context.Context, database *sql.DB, draftId int) (DraftPlayer, error) {
	//We need to get the last two picks
	assert := assert.CreateAssertWithContext("Next Pick")
	picks, err := getPicks(ctx, database, draftId)

	if err != nil {
		log.Error(ctx, "Failed to get picks", "draftId", draftId, "error", err)
		return DraftPlayer{}, err
	}

	draft, err := getDraft(ctx, database, draftId)
	if err != nil {
		log.Error(ctx, "Attempting to find next pick for invalid draft", "draftId", draftId, "error", err)
		return DraftPlayer{}, err
	}
	assert.RunAssert(ctx, len(draft.Players) > 0, "Draft has no players when finding next pick")
	var nextPlayer DraftPlayer

	//I dont think we need to account for the case where there are only two players
	if len(picks) < 2 {
		for _, player := range draft.Players {
			assert.RunAssert(ctx, player.PlayerOrder.Valid, "Got player order which was not set when finding next pick")
			if int(player.PlayerOrder.Int16) == len(picks) {
				nextPlayer = player
			}
		}
	} else {
		//We can then figure out what direction
		//we are going and if we hit the
		//end then we decide what the next pick is
		lastPlayer := GetDraftPlayerFromDraft(ctx, draft, picks[len(picks)-1].Player)
		secondLastPick := GetDraftPlayerFromDraft(ctx, draft, picks[len(picks)-2].Player)
		assert.RunAssert(ctx, lastPlayer.PlayerOrder.Valid, "Got player order which was not set when finding next pick")
		direction := lastPlayer.PlayerOrder.Int16 - secondLastPick.PlayerOrder.Int16
		if lastPlayer.User.UserUuid == secondLastPick.User.UserUuid {
			if int(lastPlayer.PlayerOrder.Int16) == len(draft.Players)-1 {
				direction = -1
			} else {
				direction = 1
			}
		}
		if len(picks)%len(draft.Players) == 0 {
			direction = 0
		}
		assert.AddContext("Last Player Id", lastPlayer.Id)
		assert.AddContext("Second Last Player Id", secondLastPick.Id)
		assert.AddContext("Last Player Order", lastPlayer.PlayerOrder)
		assert.AddContext("Second Last Player Order", secondLastPick.PlayerOrder)
		assert.AddContext("Direction", direction)

		//We know draft.players is order by player order
		assert.RunAssert(ctx, int16(len(draft.Players)) > lastPlayer.PlayerOrder.Int16+direction && lastPlayer.PlayerOrder.Int16+direction >= 0, "Next pick is out of bounds")
		nextPlayer = draft.Players[lastPlayer.PlayerOrder.Int16+direction]
	}

	//Take the pick and make it into a draft player
	return nextPlayer, err
}

func getNumPlayersInInvitedDraft(ctx context.Context, database *sql.DB, inviteId int) (int, error) {
	query := `Select
                Count(*)
            From DraftInvites ci
            Inner Join Drafts d On d.Id = ci.DraftId
            Inner Join DraftPlayers dp On dp.DraftId = d.Id
            Where ci.Id = $1;`
	assert := assert.CreateAssertWithContext("Get Num Players In Invited Draft")
	assert.AddContext("InviteId", inviteId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetNumPlayersInInvitedDraft: Failed to close statement", "error", err)
		}
	}()
	var numPlayers int
	err = stmt.QueryRowContext(ctx, inviteId).Scan(&numPlayers)
	if err != nil {
		return 0, fmt.Errorf("failed to query for num players: %w", err)
	}
	return numPlayers, nil
}

func GetDraftPlayerFromDraft(ctx context.Context, draft DraftModel, draftPlayerId int) DraftPlayer {
	for _, p := range draft.Players {
		if p.Id == draftPlayerId {
			return p
		}
	}
	return DraftPlayer{}
}

func shouldSkipPick(ctx context.Context, database *sql.DB, draftPlayer int) (bool, error) {
	assert := assert.CreateAssertWithContext("Should Skip Pick")
	assert.AddContext("draftPlayerId", draftPlayer)

	query := `SELECT
        COALESCE(skipPicks, false) As skipPicks
    From DraftPlayers dp
    Where dp.Id = $1;`

	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "ShouldSkipPick: Failed to close statement", "error", err)
		}
	}()
	var shouldSkip bool
	err = stmt.QueryRowContext(ctx, draftPlayer).Scan(&shouldSkip)

	if err != nil {
		return false, fmt.Errorf("failed to query if player should be skipped: %w", err)
	}

	return shouldSkip, nil
}

func markShouldSkipPick(ctx context.Context, database *sql.DB, draftPlayer int, shouldSkip bool) error {
	assert := assert.CreateAssertWithContext("Mark Should Skip Pick")
	assert.AddContext("draftPlayerId", draftPlayer)

	query := `Update DraftPlayers Set skipPicks = $2 Where Id = $1;`

	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to preparte statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "MarkShouldSkipPick: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, draftPlayer, shouldSkip)

	return err
}

func getCurrentPick(ctx context.Context, database *sql.DB, draftId int) (Pick, error) {
	query := `Select
                p.Id,
                p.Player,
                COALESCE(p.Pick, '') As Pick,
                p.PickTime,
                p.Skipped,
                p.AvailableTime,
                p.ExpirationTime
            From Picks p
            Inner Join (
                Select
	                Max(p.Id) As Id
                From Picks p
                Inner Join DraftPlayers dp On p.Player = dp.Id
                Where dp.DraftId = $1
            ) m On m.Id = p.Id;`

	assert := assert.CreateAssertWithContext("Get Current Pick")
	assert.AddContext("draftId", draftId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetCurrentPick: Failed to close statement", "error", err)
		}
	}()
	var pick Pick
	err = stmt.QueryRowContext(ctx, draftId).Scan(
		&pick.Id,
		&pick.Player,
		&pick.Pick,
		&pick.PickTime,
		&pick.Skipped,
		&pick.AvailableTime,
		&pick.ExpirationTime,
	)

	if err != nil {
		log.Warn(ctx, "No current pick found", "draftId", draftId, "error", err.Error())
		return Pick{}, err
	}

	return pick, nil
}

func skipPick(ctx context.Context, database *sql.DB, pickId int) error {
	query := `Update Picks Set Skipped = true Where Id = $1`

	assert := assert.CreateAssertWithContext("Skip Pick")
	assert.AddContext("pickId", pickId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "SkipPick: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, pickId)
	if err != nil {
		log.Error(ctx, "Failed to skip pick", "pickId", pickId, "error", err)
		return err
	}
	return nil
}

func updatePickExpirationTime(ctx context.Context, database *sql.DB, pickId int, expirationTime time.Time) error {
	query := `Update Picks Set ExpirationTime = $1 Where Id = $2;`

	assert := assert.CreateAssertWithContext("Update Pick Expiration Time")
	assert.AddContext("pickId", pickId)
	assert.AddContext("Expiration Time", expirationTime)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "UpdatePickExpirationTime: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, expirationTime, pickId)
	return err
}

func getPreviousPick(ctx context.Context, database *sql.DB, draftId int, currentPickId int) (Pick, error) {
	query := `Select
                p.Id,
                p.Player,
                COALESCE(p.Pick, '') As Pick,
                p.PickTime,
                p.Skipped,
                p.AvailableTime,
                p.ExpirationTime
            From Picks p
            Inner Join DraftPlayers dp On p.Player = dp.Id
            Where dp.DraftId = $1 And p.Id < $2
            Order By p.Id Desc
            Limit 1`

	assert := assert.CreateAssertWithContext("Get Previous Pick")
	assert.AddContext("draftId", draftId)
	assert.AddContext("currentPickId", currentPickId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetPreviousPick: Failed to close statement", "error", err)
		}
	}()
	var pick Pick
	err = stmt.QueryRowContext(ctx, draftId, currentPickId).Scan(
		&pick.Id,
		&pick.Player,
		&pick.Pick,
		&pick.PickTime,
		&pick.Skipped,
		&pick.AvailableTime,
		&pick.ExpirationTime,
	)

	if err != nil {
		return Pick{}, err
	}

	return pick, nil
}

func deletePick(ctx context.Context, database *sql.DB, pickId int) error {
	query := `Delete From Picks Where Id = $1`

	assert := assert.CreateAssertWithContext("Delete Pick")
	assert.AddContext("pickId", pickId)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "DeletePick: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, pickId)
	return err
}

func resetPick(ctx context.Context, database *sql.DB, pickId int, expirationTime time.Time) error {
	query := `Update Picks Set Pick = Null, PickTime = Null, Skipped = false, ExpirationTime = $1 Where Id = $2`

	assert := assert.CreateAssertWithContext("Reset Pick")
	assert.AddContext("pickId", pickId)
	assert.AddContext("Expiration Time", expirationTime)
	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "ResetPick: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, expirationTime, pickId)
	return err
}

func getDraftsInStatus(ctx context.Context, database *sql.DB, status DraftState) ([]int, error) {
	assert := assert.CreateAssertWithContext("Get Drafts In Status")
	assert.AddContext("statusCode", status)

	query := `Select
        Id
    From Drafts
    Where Status = $1;`

	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetDraftsInStatus: Failed to close statement", "error", err)
		}
	}()

	rows, err := stmt.QueryContext(ctx, status)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error(ctx, "GetDraftsInStatus: Failed to close rows", "error", err)
		}
	}()

	var errs []error
	var drafts []int
	for rows.Next() {
		var draftId int
		err = rows.Scan(&draftId)

		if err != nil {
			log.Error(ctx, "Failed to load draft in status", "statusCode", status, "error", err)
			errs = append(errs, err)
		}

		drafts = append(drafts, draftId)
	}

	return drafts, errors.Join(errs...)
}

func getDraftScore(ctx context.Context, database *sql.DB, draftId int) ([]DraftPlayer, error) {
	assert := assert.CreateAssertWithContext("Get Draft Score")
	assert.AddContext("draftId", draftId)
	assert.RunAssert(ctx, draftId != 0, "Draft Id Should Not Be 0")

	query := `Select
        dp.Id,
        u.Username,
        p.Pick
    From Picks p
    Inner Join DraftPlayers dp On p.Player = dp.Id
    Inner Join Users u On u.UserUuid = dp.UserUuid
    Where dp.DraftId = $1
	and p.Pick Is Not Null;`

	stmt, err := database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "GetDraftScore: Failed to close statement", "error", err)
		}
	}()

	rows, err := stmt.QueryContext(ctx, draftId)
	if err != nil {
		return nil, fmt.Errorf("failed to get picks for draft: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Error(ctx, "GetDraftScore: Failed to close rows", "error", err)
		}
	}()

	picks := make(map[int][]string)
	usernames := make(map[int]string)
	for rows.Next() {
		var playerId int
		var username string
		var pick string
		err = rows.Scan(&playerId, &username, &pick)

		if err != nil {
			return nil, fmt.Errorf("failed to scan draft scores: %w", err)
		}

		usernames[playerId] = username
		picks[playerId] = append(picks[playerId], pick)
	}

	var playerScores []DraftPlayer
	for player, playerPicks := range picks {
		draftPlayer := DraftPlayer{
			Id: player,
			User: User{
				Username: usernames[player],
			},
			Score: 0,
		}

		for _, pick := range playerPicks {
			score, err := getScore(ctx, database, pick)
			if err != nil {
				return nil, fmt.Errorf("failed to get score for pick %s: %w", pick, err)
			}
			draftPlayer.Score += score["Total Score"]

			team := Pick{
				Pick: sql.NullString{
					Valid:  true,
					String: pick,
				},
				Score: score["Total Score"],
			}
			draftPlayer.Picks = append(draftPlayer.Picks, team)
		}
		playerScores = append(playerScores, draftPlayer)
	}

	assert.RunAssert(ctx, len(picks) == len(usernames), "Picks and usernames maps have inconsistent lengths")
	return playerScores, nil
}
