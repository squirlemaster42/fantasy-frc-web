package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"server/assert"
	"server/database"
	"server/log"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type DraftState string
type TimingType string

const (
	FILLING          DraftState = "Filling"
	WAITING_TO_START DraftState = "Waiting to Start"
	PICKING          DraftState = "Picking"
	TEAMS_PLAYING    DraftState = "Teams Playing"
	COMPLETE         DraftState = "Complete"
    INCREMENT		 TimingType = "Increment"
    PER_PICK		 TimingType = "Per Pick"
)

const (
	DraftPlayerCount = 8
	PicksPerPlayer   = 8
	PicksPerDraft    = DraftPlayerCount * PicksPerPlayer // 64
)

type DraftSearchQuery struct {
	UserUuid uuid.UUID
	DraftNameSearch string
	PageNum int
	PageSize int
}

type DraftModel struct {
	Id                int
	DisplayName       string
	Description       string
	DiscordWebhook    string
	Owner             User
	Status            DraftState
	Players           []DraftPlayer
	NextPick          DraftPlayer
	CurrentPick       Pick
	Picks             []Pick
	TimingType 	      TimingType
	IncrementTimeSec  int16
	PerPickExpTimeSec int16
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

	return fmt.Sprintf("Draft: {\nId: %d\n Displayname: %s\n Description: %s\n Owner: %s\n Status: %s\n Players: %s\n NextPick: %s\n TimingType %s\n IncrementTimeSec %d\n PerPickExpTimeSec %d\n}",
		d.Id, d.DisplayName, d.Description, d.Owner.String(), d.Status, stringBuilder.String(), d.NextPick.String(), d.TimingType, d.IncrementTimeSec, d.PerPickExpTimeSec)
}

type DraftPlayer struct {
	Id          		 int
	User        		 User
	PlayerOrder 		 sql.NullInt16
	Pending     		 bool
	Score       		 int
	Picks       		 []Pick
	InviteId    		 int
	RemainingPickTimeSec int
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
	return fmt.Sprintf("DraftPlayer: {\nId: %d\n User: %s\n PlayerOrder: %s\n Pending: %t\n Picks: %s\n RemainingPickTimeSec %d\n}",
		d.Id, d.User.UserUuid.String(), playerOrderStr, d.Pending, stringBuilder.String(), d.RemainingPickTimeSec)
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
	Status             string
}

func (d *DraftInvite) String() string {
	return fmt.Sprintf("DraftInvite: {\nId: %d\n DraftId: %d\n InvitingUserUuid: %s\n InvitedUserUuid: %s\n SentTime: %s\n AcceptedTime: %s\n Status: %s\n DraftName: %s\n InvitingPlayerName: %s\n InvitedPlayerName: %s\n}",
		d.Id, d.DraftId, d.InvitingUserUuid.String(), d.InvitedUserUuid.String(), d.SentTime.String(), d.AcceptedTime.String(), d.Status, d.DraftName, d.InvitingPlayerName, d.InvitedPlayerName)
}

func getDraftsByName(ctx context.Context, db database.DBTX, searchString string) ([]DraftModel, error) {
	query := `SELECT
        Drafts.Id,
        DisplayName
    From Drafts
    Where DisplayName LIKE CONCAT('%', Cast($1 As varchar), '%');`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get drafts by name: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetDraftsByName")
	rows, err := stmt.QueryContext(ctx, searchString)

	if err != nil {
		return nil, fmt.Errorf("failed to get drafts by name: %w", err)
	}
	defer database.CloseRows(ctx, rows, "GetDraftsByName")
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

func searchDrafts(ctx context.Context, db database.DBTX, search DraftSearchQuery) ([]DraftModel, error) {
	query := `SELECT DISTINCT
        Drafts.Id,
        Drafts.displayName,
        owners.UserUuid As ownerId,
        owners.Username As OwnerUsername,
        COALESCE(Drafts.Status, '0') As Status
    From Drafts
    Left Join DraftPlayers On DraftPlayers.DraftId = Drafts.Id
    Left Join DraftInvites On DraftInvites.DraftId = Drafts.Id And Drafts.Status = $1
    Left Join Users dpUsers On DraftPlayers.UserUuid = dpUsers.UserUuid
    Left Join Users diUsers On DraftInvites.InvitedUserUuid = diUsers.UserUuid
    Left Join Users owners On Drafts.OwnerUserUuid = owners.UserUuid
	Left Join Users currUser On currUser.UserUuid = $2
    Where (DraftPlayers.UserUuid = $2
		Or DraftInvites.InvitedUserUuid = $2
		Or currUser.IsAdmin = true)
		And Drafts.DisplayName ILIKE CONCAT('%', CAST($3 As VARCHAR), '%')
	Order By Drafts.Id Asc
	Limit $4
	Offset $5;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search drafts: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "SearchDrafts")
	rows, err := stmt.QueryContext(ctx, FILLING, search.UserUuid, search.DraftNameSearch, search.PageSize, search.PageNum*search.PageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to search drafts: %w", err)
	}
	defer database.CloseRows(ctx, rows, "SearchDrafts")

	var drafts []DraftModel
	draftIndexById := make(map[int]int)
	var pickingDraftIds []int

	for rows.Next() {
		var draftId int
		var displayName string
		var ownerId uuid.UUID
		var ownerUsername string
		var status DraftState
		err = rows.Scan(&draftId, &displayName, &ownerId, &ownerUsername, &status)
		if err != nil {
			return nil, fmt.Errorf("failed to search drafts: %w", err)
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

		draftIndexById[draftId] = len(drafts)
		drafts = append(drafts, draftModel)

		if status == PICKING {
			pickingDraftIds = append(pickingDraftIds, draftId)
		}
	}

	if len(drafts) == 0 {
		return drafts, nil
	}

	draftIds := make([]int, 0, len(draftIndexById))
	for id := range draftIndexById {
		draftIds = append(draftIds, id)
	}

	playersByDraft, err := loadDraftPlayersBatch(ctx, db, draftIds)
	if err != nil {
		return nil, fmt.Errorf("failed to search drafts: %w", err)
	}
	for draftId, players := range playersByDraft {
		idx := draftIndexById[draftId]
		drafts[idx].Players = players
	}

	if len(pickingDraftIds) > 0 {
		nextPicks, err := loadCurrentPicksBatch(ctx, db, pickingDraftIds)
		if err != nil {
			return nil, fmt.Errorf("failed to search drafts: %w", err)
		}
		for draftId, nextPick := range nextPicks {
			idx := draftIndexById[draftId]
			drafts[idx].NextPick = nextPick
		}
	}

	return drafts, nil
}

func loadDraftPlayersBatch(ctx context.Context, db database.DBTX, draftIds []int) (map[int][]DraftPlayer, error) {
	playersByDraft := make(map[int][]DraftPlayer)
	if len(draftIds) == 0 {
		return playersByDraft, nil
	}

	query := `SELECT
		DraftId,
		UserUuid,
		USERNAME,
		BOOL_OR(Status = 'accepted') AS ACCEPTED,
		MAX(PLAYERORDER) AS PLAYERORDER
	FROM (
		SELECT
			dp.DraftId,
			u.UserUuid,
			u.USERNAME,
			'accepted' AS Status,
			dp.PLAYERORDER
		FROM USERS u
		INNER JOIN DRAFTPLAYERS dp ON dp.UserUuid = u.UserUuid
		WHERE dp.DraftId = ANY($1)
		UNION ALL
		SELECT
			di.DraftId,
			u.UserUuid,
			u.USERNAME,
			di.Status,
			-1 AS PLAYERORDER
		FROM USERS u
		INNER JOIN DRAFTINVITES di ON di.InvitedUserUuid = u.UserUuid
		WHERE di.DraftId = ANY($1) AND di.Status != 'canceled'
	) U
	GROUP BY DraftId, UserUuid, USERNAME
	ORDER BY DraftId, MAX(PLAYERORDER);`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load draft players batch: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "LoadDraftPlayersBatch")

	rows, err := stmt.QueryContext(ctx, draftIds)
	if err != nil {
		return nil, fmt.Errorf("failed to load draft players batch: %w", err)
	}
	defer database.CloseRows(ctx, rows, "LoadDraftPlayersBatch")

	for rows.Next() {
		var draftId int
		var userUuid uuid.UUID
		var username string
		var accepted bool
		var playerOrder sql.NullInt16
		err = rows.Scan(&draftId, &userUuid, &username, &accepted, &playerOrder)
		if err != nil {
			return nil, fmt.Errorf("failed to scan draft player batch: %w", err)
		}

		draftPlayer := DraftPlayer{
			User: User{
				UserUuid: userUuid,
				Username: username,
			},
			Pending:     !accepted,
			PlayerOrder: playerOrder,
		}
		playersByDraft[draftId] = append(playersByDraft[draftId], draftPlayer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating draft players batch: %w", err)
	}

	return playersByDraft, nil
}

func loadCurrentPicksBatch(ctx context.Context, db database.DBTX, draftIds []int) (map[int]DraftPlayer, error) {
	nextPicks := make(map[int]DraftPlayer)
	if len(draftIds) == 0 {
		return nextPicks, nil
	}

	query := `SELECT
		p.Id,
		p.Player,
		COALESCE(p.Pick, '') AS Pick,
		p.PickTime,
		p.Skipped,
		p.AvailableTime,
		p.ExpirationTime,
		dp.DraftId,
		u.UserUuid,
		u.Username
	FROM Picks p
	INNER JOIN DraftPlayers dp ON p.Player = dp.Id
	INNER JOIN Users u ON dp.UserUuid = u.UserUuid
	INNER JOIN (
		SELECT dp.DraftId, MAX(p.Id) AS Id
		FROM Picks p
		INNER JOIN DraftPlayers dp ON p.Player = dp.Id
		WHERE dp.DraftId = ANY($1)
		GROUP BY dp.DraftId
	) m ON m.Id = p.Id;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("failed to load current picks batch: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "LoadCurrentPicksBatch")

	rows, err := stmt.QueryContext(ctx, draftIds)
	if err != nil {
		return nil, fmt.Errorf("failed to load current picks batch: %w", err)
	}
	defer database.CloseRows(ctx, rows, "LoadCurrentPicksBatch")

	for rows.Next() {
		var pick Pick
		var draftId int
		var user User
		err = rows.Scan(
			&pick.Id,
			&pick.Player,
			&pick.Pick,
			&pick.PickTime,
			&pick.Skipped,
			&pick.AvailableTime,
			&pick.ExpirationTime,
			&draftId,
			&user.UserUuid,
			&user.Username,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan current pick batch: %w", err)
		}

		nextPicks[draftId] = DraftPlayer{
			Id:   pick.Player,
			User: user,
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating current picks batch: %w", err)
	}

	return nextPicks, nil
}

func createDraft(ctx context.Context, db database.DBTX, draft *DraftModel) (int, error) {
	if draft.Owner.UserUuid == uuid.Nil {
		return 0, errors.New("draft owner uuid is nil")
	}

	query := `INSERT INTO Drafts (DisplayName, OwnerUserUuid, Description, Status) Values ($1, $2, $3, $4) RETURNING Id;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return -1, fmt.Errorf("failed to prepare create draft statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "CreateDraft")
	var draftId int
	err = stmt.QueryRowContext(ctx, draft.DisplayName, draft.Owner.UserUuid, draft.Description, draft.Status).Scan(&draftId)
	if err != nil {
		return -1, fmt.Errorf("failed to create draft: %w", err)
	}
	playerQuery := `INSERT INTO DraftPlayers (draftId, useruuid) Values ($1, $2);`
	stmt, err = database.Prepare(ctx, db, playerQuery)
	if err != nil {
		return -1, fmt.Errorf("failed to prepare create draft player statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "CreateDraft")
	_, err = stmt.ExecContext(ctx, draftId, draft.Owner.UserUuid)
	if err != nil {
		return -1, fmt.Errorf("failed to add owner to draft: %w", err)
	}
	log.Info(ctx, "Created draft", "draftId", draftId, "ownerUuid", draft.Owner.UserUuid)
	return draftId, nil
}

func updateDraftStatus(ctx context.Context, db database.DBTX, draftId int, status DraftState) error {
	query := `Update Drafts Set Status = $1 Where Id = $2;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare update draft status statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "UpdateDraftStatus")

	_, err = stmt.ExecContext(ctx, status, draftId)
	if err != nil {
		log.Error(ctx, "Failed to update draft status", "draftId", draftId, "statusCode", status, "error", err)
		return fmt.Errorf("failed to update draft status: %w", err)
	}
	return nil
}

func queryDraftRow(ctx context.Context, db database.DBTX, draftId int) (DraftModel, uuid.UUID, error) {
	log.Debug(ctx, "model.GetDraft: starting", "draftId", draftId)
	query := `Select
        DisplayName,
        COALESCE(Description, '') As Description,
        COALESCE(Status, '') As Status,
        OwnerUserUuid,
		COALESCE(DiscordWebhook, '')
    From Drafts Where Id = $1;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return DraftModel{}, uuid.UUID{}, fmt.Errorf("failed to query draft row: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetDraft")
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
		return DraftModel{}, uuid.UUID{}, fmt.Errorf("failed to load draft: %w", err)
	}

	return draftModel, ownerId, nil
}

func loadCurrentPickIfPicking(ctx context.Context, db database.DBTX, draftModel *DraftModel) error {
	log.Debug(ctx, "Checking if we need to get the current pick for the draft", "statusCode", draftModel.Status, "picking", PICKING)
	if draftModel.Status == PICKING {
		log.Debug(ctx, "Getting the current pick for the draft")
		currPick, err := getCurrentPick(ctx, db, draftModel.Id)
		if err != nil {
			return fmt.Errorf("failed to load current pick: %w", err)
		}
		draftModel.NextPick = DraftPlayer{
			Id: currPick.Player,
		}
	}
	return nil
}

func loadDraftPlayers(ctx context.Context, db database.DBTX, draftId int, draftModel *DraftModel, ownerId uuid.UUID) error {
	playerQuery := `SELECT
                        UserUuid,
	                    USERNAME,
	                    BOOL_OR(Status = 'accepted') AS ACCEPTED,
	                    MAX(PLAYERORDER) AS PLAYERORDER,
	                    Max(PlayerId) As PlayerId,
	                    MAX(InviteId) As InviteId
                    FROM (
		                    SELECT
			                    USERS.UserUuid AS UserUuid,
			                    USERS.USERNAME,
			                    'accepted' AS Status,
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
			                    DRAFTINVITES.Status AS Status,
			                    -1 AS PLAYERORDER,
			                    -1 As PlayerId,
			                    DraftInvites.Id As InviteId
		                    FROM USERS
		                    INNER JOIN DRAFTINVITES ON DRAFTINVITES.InvitedUserUuid = USERS.UserUuid
		                    WHERE DRAFTINVITES.DRAFTID = $1 AND DRAFTINVITES.Status != 'canceled'
	                    ) U
                    GROUP BY UserUuid, USERNAME
                    ORDER BY PLAYERORDER;`

	playerStmt, err := database.Prepare(ctx, db, playerQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare load players statement: %w", err)
	}
	defer database.CloseStatement(ctx, playerStmt, "GetDraft")
	playerRows, err := playerStmt.QueryContext(ctx, draftId)
	if err != nil {
		log.Error(ctx, "Failed to load players for draft", "draftId", draftId, "error", err)
		return fmt.Errorf("failed to load players for draft: %w", err)
	}
	defer database.CloseRows(ctx, playerRows, "GetDraft")

	picksByPlayer := make(map[int][]Pick)
	for _, pick := range draftModel.Picks {
		picksByPlayer[pick.Player] = append(picksByPlayer[pick.Player], pick)
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
			return fmt.Errorf("failed to scan draft player: %w", err)
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
			Picks:       picksByPlayer[playerId],
		}

		draftModel.Players = append(draftModel.Players, draftPlayer)
	}

	return nil
}

func getDraft(ctx context.Context, db database.DBTX, draftId int) (DraftModel, error) {
	draftModel, ownerId, err := queryDraftRow(ctx, db, draftId)
	if err != nil {
		return DraftModel{}, fmt.Errorf("failed to get draft: %w", err)
	}

	currentPick, err := getCurrentPick(ctx, db, draftId)
	if err != nil {
		log.Error(ctx, "Failed to get current pick for draft", "draftId", draftId, "error", err)
	} else {
		draftModel.CurrentPick = currentPick
	}

	picks, err := getPicks(ctx, db, draftId)
	if err != nil {
		log.Error(ctx, "Failed to get picks for draft", "draftId", draftId, "error", err)
		return DraftModel{}, fmt.Errorf("failed to get picks for draft: %w", err)
	}
	draftModel.Picks = picks

	err = loadCurrentPickIfPicking(ctx, db, &draftModel)
	if err != nil {
		return DraftModel{}, fmt.Errorf("failed to load current pick: %w", err)
	}

	err = loadDraftPlayers(ctx, db, draftId, &draftModel, ownerId)
	if err != nil {
		return DraftModel{}, fmt.Errorf("failed to load draft players: %w", err)
	}

	return draftModel, nil
}

func updateDraft(ctx context.Context, db database.DBTX, draft *DraftModel) error {
	log.Debug(ctx, "model.UpdateDraft: starting", "draftId", draft.Id)
	query := `Update Drafts Set DisplayName = $1, Description = $2, DiscordWebhook = $3 Where Id = $4;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare update draft statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "UpdateDraft")
	log.Debug(ctx, "model.UpdateDraft: executing query", "draftId", draft.Id)
	_, err = stmt.ExecContext(ctx, draft.DisplayName, draft.Description, draft.DiscordWebhook, draft.Id)
	log.Debug(ctx, "model.UpdateDraft: query completed", "draftId", draft.Id)
	if err != nil {
		return fmt.Errorf("failed to update draft: %w", err)
	}
	log.Info(ctx, "Updated draft", "draftId", draft.Id)
	return nil
}

func invitePlayer(ctx context.Context, db database.DBTX, draft int, invitingUserUuid uuid.UUID, invitedUserUuid uuid.UUID) (int, error) {
	query := `INSERT INTO DraftInvites (draftId, invitingUserUuid, invitedUserUuid,
    sentTime, Status) Values ($1, $2, $3, $4, $5) RETURNING Id;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return -1, fmt.Errorf("failed to prepare invite player statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "InvitePlayer")

	var inviteId int
	err = stmt.QueryRowContext(ctx, draft, invitingUserUuid, invitedUserUuid, time.Now().UTC(), "pending").Scan(&inviteId)
	if err != nil {
		return -1, fmt.Errorf("failed to invite player: %w", err)
	}
	log.Info(ctx, "Invited player to draft", "draftId", draft, "invitedUserUuid", invitedUserUuid, "inviteId", inviteId)
	return inviteId, nil
}

// Returns draftId, UserUuid, error
func acceptInvite(ctx context.Context, db database.DBTX, inviteId int) (int, uuid.UUID, error) {
	query := `UPDATE DraftInvites Set Status = 'accepted', acceptedTime = $1 where id = $2;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return 0, uuid.UUID{}, fmt.Errorf("failed to prepare accept invite statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "AcceptInvite")
	_, err = stmt.ExecContext(ctx, time.Now().UTC(), inviteId)
	if err != nil {
		return 0, uuid.UUID{}, fmt.Errorf("failed to accept invite: %w", err)
	}

	query = `Select DraftId, InvitedUserUuid From DraftInvites Where Id = $1;`
	stmt, err = database.Prepare(ctx, db, query)
	if err != nil {
		return 0, uuid.UUID{}, fmt.Errorf("failed to prepare select invite statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "AcceptInvite")
	var draftId int
	var userUuid uuid.UUID
	err = stmt.QueryRowContext(ctx, inviteId).Scan(&draftId, &userUuid)
	if err != nil {
		return 0, uuid.UUID{}, fmt.Errorf("failed to get invite details: %w", err)
	}

	log.Info(ctx, "Accepted invite", "inviteId", inviteId, "draftId", draftId, "userUuid", userUuid)
	return draftId, userUuid, nil
}

func addPlayerToDraft(ctx context.Context, db database.DBTX, draft int, player uuid.UUID) error {
	query := `INSERT INTO DraftPlayers (draftId, UserUuid) Values ($1, $2);`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare add player statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "AddPlayerToDraft")
	_, err = stmt.ExecContext(ctx, draft, player)
	if err != nil {
		return fmt.Errorf("failed to add player to draft: %w", err)
	}
	return nil
}

func cancelOutstandingInvites(ctx context.Context, db database.DBTX, draftId int) error {
	query := `Update DraftInvites Set Status = 'canceled' Where DraftId = $1 and Status = 'pending';`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare cancel outstanding invites statement: %w", err)
	}

	defer database.CloseStatement(ctx, stmt, "CancelOutstandingInvites")

	_, err = stmt.ExecContext(ctx, draftId)
	if err != nil {
		return fmt.Errorf("failed to cancel invites for draft %d: %w", draftId, err)
	}

	return nil
}

func getInvite(ctx context.Context, db database.DBTX, inviteId int) (DraftInvite, error) {
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
        And di.Status != 'canceled';`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return DraftInvite{}, fmt.Errorf("failed to get invite: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetInvite")
	invite := DraftInvite{}
	err = stmt.QueryRowContext(ctx, inviteId).Scan(
		&invite.Id,
		&invite.InvitingPlayerName,
		&invite.InvitedUserUuid,
		&invite.DraftName,
		&invite.DraftId)
	if err != nil {
		log.Error(ctx, "GetInvite: Failed to query invite", "error", err, "inviteId", inviteId)
		return DraftInvite{}, fmt.Errorf("failed to get invite: %w", err)
	}
	return invite, nil
}

func getInvites(ctx context.Context, db database.DBTX, userUuid uuid.UUID) ([]DraftInvite, error) {
	query := `SELECT
            di.Id,
            u.username,
            d.DisplayName
        From DraftInvites di
        Inner Join Drafts d On di.DraftId = d.Id
        Inner Join Users u On di.InvitingUserUuid = u.UserUuid
        Where di.InvitedUserUuid = $1
        And di.Status = 'pending';`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare get invites statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetInvites")
	rows, err := stmt.QueryContext(ctx, userUuid)

	if err != nil {
		return nil, fmt.Errorf("failed to get invites: %w", err)
	}
	defer database.CloseRows(ctx, rows, "GetInvites")

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

func cancelInvite(ctx context.Context, db database.DBTX, inviteId int) error {
	query := `Update DraftInvites Set Status = 'canceled' Where Id = $1;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare cancel invite statement: %w", err)
	}

	defer database.CloseStatement(ctx, stmt, "CancelInvite")

	_, err = stmt.ExecContext(ctx, inviteId)
	if err != nil {
		return fmt.Errorf("failed to cancel invite %d: %w", inviteId, err)
	}

	return nil
}

func uninvitePlayer(ctx context.Context, db database.DBTX, draftId int, ownerUuid uuid.UUID, inviteId int) error {
	ownerQuery := `Select OwnerUserUuid From Drafts Where Id = $1;`
	ownerStmt, err := database.Prepare(ctx, db, ownerQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare get draft owner statement: %w", err)
	}

	var dbOwnerUuid string
	err = ownerStmt.QueryRowContext(ctx, draftId).Scan(&dbOwnerUuid)
	database.CloseStatement(ctx, ownerStmt, "UninvitePlayer")
	if err != nil {
		return fmt.Errorf("failed to get draft owner: %w", err)
	}

	if dbOwnerUuid != ownerUuid.String() {
		return fmt.Errorf("user %s is not the owner of draft %d", ownerUuid, draftId)
	}

	query := `Update DraftInvites Set Status = 'canceled' Where Id = $1 And DraftId = $2;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare uninvite player statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "UninvitePlayer")

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

func getOutstandingInvitesForDraft(ctx context.Context, db database.DBTX, draftId int) ([]DraftInvite, error) {
	query := `SELECT
		di.Id,
		u.username,
		di.InvitedUserUuid
	From DraftInvites di
	Inner Join Users u On di.InvitedUserUuid = u.UserUuid
	Where di.DraftId = $1
	And di.Status = 'pending';`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get outstanding invites for draft: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetOutstandingInvitesForDraft")

	rows, err := stmt.QueryContext(ctx, draftId)
	if err != nil {
		return nil, fmt.Errorf("failed to get outstanding invites: %w", err)
	}
	defer database.CloseRows(ctx, rows, "GetOutstandingInvitesForDraft")

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

func getPicks(ctx context.Context, db database.DBTX, draftId int) ([]Pick, error) {
	query := `SELECT
        Picks.id, Picks.player, Picks.pick, Picks.pickTime, Picks.ExpirationTime, Picks.Skipped
    From Picks
    Inner Join DraftPlayers On DraftPlayers.id = Picks.player
    Where DraftPlayers.draftId = $1
    Order By Picks.AvailableTime Asc;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get picks: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetPicks")
	rows, err := stmt.QueryContext(ctx, draftId)
	if err != nil {
		return nil, fmt.Errorf("failed to get picks: %w", err)
	}
	defer database.CloseRows(ctx, rows, "GetPicks")

	var picks []Pick
	for rows.Next() {
		pick := Pick{}
		err = rows.Scan(&pick.Id, &pick.Player, &pick.Pick, &pick.PickTime, &pick.ExpirationTime, &pick.Skipped)

		if err != nil {
			return nil, fmt.Errorf("failed to get picks: %w", err)
		}

		picks = append(picks, pick)
	}

	return picks, nil
}

func getDraftPlayerId(ctx context.Context, db database.DBTX, draftId int, userUuid uuid.UUID) (int, error) {
	query := `Select Id From DraftPlayers Where draftId = $1 And userUuid = $2`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return -1, fmt.Errorf("failed to get draft player id: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetDraftPlayerId")

	var draftPlayerId int
	err = stmt.QueryRowContext(ctx, draftId, userUuid).Scan(&draftPlayerId)

	if err != nil {
		return -1, errors.Join(fmt.Errorf("failed to get draft player for user %s in draft %d", userUuid.String(), draftId), err)
	}

	return draftPlayerId, nil
}

func getDraftPlayerUser(ctx context.Context, db database.DBTX, draftPlayerId int) (User, error) {
	query := `Select
        u.UserUuid,
        u.Username
    From DraftPlayers dp
    Inner Join Users u On dp.UserUuid = u.UserUuid
    Where dp.Id = $1;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return User{}, fmt.Errorf("failed to get draft player user: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetDraftPlayerUser")

	var user User
	err = stmt.QueryRowContext(ctx, draftPlayerId).Scan(&user.UserUuid, &user.Username)
	if err != nil {
		return User{}, fmt.Errorf("failed to get draft player user: %w", err)
	}

	return user, nil
}

func makePickAvailable(ctx context.Context, db database.DBTX, draftPlayerId int, availableTime time.Time, expirationTime time.Time) (int, error) {
	query := `Insert Into Picks (Player, AvailableTime, ExpirationTime) Values ($1, $2, $3) Returning Id;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare make pick available statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "MakePickAvailable")

	var pickId int
	err = stmt.QueryRowContext(ctx, draftPlayerId, availableTime, expirationTime).Scan(&pickId)

	if err != nil {
		log.Error(ctx, "Failed to make pick available", "draftPlayerId", draftPlayerId, "error", err)
		return 0, fmt.Errorf("failed to make pick available: %w", err)
	}

	return pickId, nil
}

func makePick(ctx context.Context, db database.DBTX, pick Pick) error {
	query := `Update Picks Set pick = $1, pickTime = $2 Where Id = $3 Returning Id;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare make pick statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "MakePick")
	var updatedId int
	err = stmt.QueryRowContext(ctx, pick.Pick, pick.PickTime, pick.Id).Scan(&updatedId)
	if err != nil {
		log.Error(ctx, "Failed to make pick", "error", err)
		return fmt.Errorf("failed to make pick: %w", err)
	}
	if updatedId != pick.Id {
		log.Error(ctx, "Pick id returned from database does not match expected id", "expected", pick.Id, "actual", updatedId)
		return fmt.Errorf("pick id returned from database does not match expected id")
	}
	log.Info(ctx, "Made pick", "pickId", pick.Id, "team", pick.Pick.String, "player", pick.Player)
	return nil
}

func hasBeenPicked(ctx context.Context, db database.DBTX, draftId int, team string) (bool, error) {
	query := `SELECT
    Count(*) As num
    From Picks
    Inner Join DraftPlayers On DraftPlayers.id = Picks.player
    Where DraftPlayers.draftId = $1
    And Picks.pick = $2;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return false, fmt.Errorf("failed to has been picked: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "HasBeenPicked")
	var numPicked int
	err = stmt.QueryRowContext(ctx, draftId, team).Scan(&numPicked)
	if err != nil {
		log.Error(ctx, "Failed to query for picks", "draftId", draftId, "team", team, "error", err)
		return false, fmt.Errorf("failed to has been picked: %w", err)
	}
	return numPicked != 0, nil
}

func randomizePickOrder(ctx context.Context, db database.DBTX, draftId int) error {
	draftModel, err := getDraft(ctx, db, draftId)
	if err != nil {
		log.Warn(ctx, "Attempting to randomize pick order for invalid draft", "draftId", draftId)
		return fmt.Errorf("could not load draft %d: %w", draftId, err)
	}
	var awaitingAssignment []DraftPlayer
	// We only want to randomize the pick order of players who accepted the draft
	for _, player := range draftModel.Players {
		if !player.Pending {
			awaitingAssignment = append(awaitingAssignment, player)
		}
	}

	for i := range awaitingAssignment {
		j := rand.Intn(i + 1)
		awaitingAssignment[i], awaitingAssignment[j] = awaitingAssignment[j], awaitingAssignment[i]
	}

	query := `Update DraftPlayers Set PlayerOrder = $1 Where Id = $2`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare randomize pick order statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "RandomizePickOrder")

	for i, player := range awaitingAssignment {
		draftPlayerId, err := getDraftPlayerId(ctx, db, draftId, player.User.UserUuid)
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

func nextPick(ctx context.Context, db database.DBTX, draftId int) (DraftPlayer, error) {
	picks, err := getPicks(ctx, db, draftId)
	if err != nil {
		log.Error(ctx, "Failed to get picks", "draftId", draftId, "error", err)
		return DraftPlayer{}, fmt.Errorf("failed to next pick: %w", err)
	}

	draft, err := getDraft(ctx, db, draftId)
	if err != nil {
		log.Error(ctx, "Attempting to find next pick for invalid draft", "draftId", draftId, "error", err)
		return DraftPlayer{}, fmt.Errorf("failed to next pick: %w", err)
	}

	return DetermineNextPick(draft.Players, picks)
}

func getNumPlayersInInvitedDraft(ctx context.Context, db database.DBTX, inviteId int) (int, error) {
	query := `Select
                Count(*)
            From DraftInvites ci
            Inner Join Drafts d On d.Id = ci.DraftId
            Inner Join DraftPlayers dp On dp.DraftId = d.Id
            Where ci.Id = $1;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare get num players statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetNumPlayersInInvitedDraft")
	var numPlayers int
	err = stmt.QueryRowContext(ctx, inviteId).Scan(&numPlayers)
	if err != nil {
		return 0, fmt.Errorf("failed to query for num players: %w", err)
	}
	return numPlayers, nil
}

func lockDraft(ctx context.Context, db database.DBTX, draftId int) error {
	query := `SELECT Id FROM Drafts WHERE Id = $1 FOR UPDATE;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare lock draft statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "LockDraft")
	_, err = stmt.ExecContext(ctx, draftId)
	if err != nil {
		return fmt.Errorf("failed to lock draft: %w", err)
	}
	return nil
}

func getNumPlayersInDraft(ctx context.Context, db database.DBTX, draftId int) (int, error) {
	query := `Select Count(*) From DraftPlayers Where draftId = $1;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare get num players in draft statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetNumPlayersInDraft")
	var numPlayers int
	err = stmt.QueryRowContext(ctx, draftId).Scan(&numPlayers)
	if err != nil {
		return 0, fmt.Errorf("failed to query for num players in draft: %w", err)
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

func DetermineNextPick(players []DraftPlayer, picks []Pick) (DraftPlayer, error) {
	if len(players) == 0 {
		return DraftPlayer{}, fmt.Errorf("no players in draft")
	}

	findPlayer := func(playerId int) (DraftPlayer, error) {
		for _, p := range players {
			if p.Id == playerId {
				return p, nil
			}
		}
		return DraftPlayer{}, fmt.Errorf("player %d not found in draft", playerId)
	}

	var nextPlayer DraftPlayer

	if len(picks) < 2 {
		for _, player := range players {
			if !player.PlayerOrder.Valid {
				return DraftPlayer{}, fmt.Errorf("player order not set when finding next pick")
			}
			if int(player.PlayerOrder.Int16) == len(picks) {
				nextPlayer = player
			}
		}
		if nextPlayer.Id == 0 {
			return DraftPlayer{}, fmt.Errorf("next player has invalid id")
		}
		return nextPlayer, nil
	}

	lastPlayer, err := findPlayer(picks[len(picks)-1].Player)
	if err != nil {
		return DraftPlayer{}, fmt.Errorf("failed to determine next pick: %w", err)
	}
	secondLastPick, err := findPlayer(picks[len(picks)-2].Player)
	if err != nil {
		return DraftPlayer{}, fmt.Errorf("failed to determine next pick: %w", err)
	}
	if !lastPlayer.PlayerOrder.Valid {
		return DraftPlayer{}, fmt.Errorf("player order not set when finding next pick")
	}
	direction := lastPlayer.PlayerOrder.Int16 - secondLastPick.PlayerOrder.Int16
	if lastPlayer.User.UserUuid == secondLastPick.User.UserUuid {
		if int(lastPlayer.PlayerOrder.Int16) == len(players)-1 {
			direction = -1
		} else {
			direction = 1
		}
	}
	if len(picks)%len(players) == 0 {
		direction = 0
	}

	nextIndex := lastPlayer.PlayerOrder.Int16 + direction
	if nextIndex < 0 || int(nextIndex) >= len(players) {
		return DraftPlayer{}, fmt.Errorf("next pick is out of bounds")
	}
	nextPlayer = players[nextIndex]
	if nextPlayer.Id == 0 {
		return DraftPlayer{}, fmt.Errorf("next player has invalid id")
	}
	return nextPlayer, nil
}

func shouldSkipPick(ctx context.Context, db database.DBTX, draftPlayer int) (bool, error) {
	query := `SELECT
        COALESCE(skipPicks, false) As skipPicks
    From DraftPlayers dp
    Where dp.Id = $1;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare should skip pick statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "ShouldSkipPick")
	var shouldSkip bool
	err = stmt.QueryRowContext(ctx, draftPlayer).Scan(&shouldSkip)

	if err != nil {
		return false, fmt.Errorf("failed to query if player should be skipped: %w", err)
	}

	return shouldSkip, nil
}

func markShouldSkipPick(ctx context.Context, db database.DBTX, draftPlayer int, shouldSkip bool) error {
	query := `Update DraftPlayers Set skipPicks = $2 Where Id = $1;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare mark should skip pick statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "MarkShouldSkipPick")
	_, err = stmt.ExecContext(ctx, draftPlayer, shouldSkip)
	if err != nil {
		return fmt.Errorf("failed to mark should skip pick: %w", err)
	}
	return nil
}

func getCurrentPick(ctx context.Context, db database.DBTX, draftId int) (Pick, error) {
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

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return Pick{}, fmt.Errorf("failed to get current pick: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetCurrentPick")
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
		return Pick{}, fmt.Errorf("failed to get current pick: %w", err)
	}

	return pick, nil
}

func skipPick(ctx context.Context, db database.DBTX, pickId int) error {
	query := `Update Picks Set Skipped = true Where Id = $1`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare skip pick statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "SkipPick")
	_, err = stmt.ExecContext(ctx, pickId)
	if err != nil {
		log.Error(ctx, "Failed to skip pick", "pickId", pickId, "error", err)
		return fmt.Errorf("failed to skip pick: %w", err)
	}
	return nil
}

func updatePickExpirationTime(ctx context.Context, db database.DBTX, pickId int, expirationTime time.Time) error {
	query := `Update Picks Set ExpirationTime = $1 Where Id = $2;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare update pick expiration time statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "UpdatePickExpirationTime")
	_, err = stmt.ExecContext(ctx, expirationTime, pickId)
	if err != nil {
		return fmt.Errorf("failed to update pick expiration time: %w", err)
	}
	return nil
}

func getPreviousPick(ctx context.Context, db database.DBTX, draftId int, currentPickId int) (Pick, error) {
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

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return Pick{}, fmt.Errorf("failed to get previous pick: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetPreviousPick")
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
		return Pick{}, fmt.Errorf("failed to get previous pick: %w", err)
	}

	return pick, nil
}

func deletePick(ctx context.Context, db database.DBTX, pickId int) error {
	query := `Delete From Picks Where Id = $1`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare delete pick statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "DeletePick")
	_, err = stmt.ExecContext(ctx, pickId)
	if err != nil {
		return fmt.Errorf("failed to delete pick: %w", err)
	}
	return nil
}

func resetPick(ctx context.Context, db database.DBTX, pickId int, expirationTime time.Time) error {
	query := `Update Picks Set Pick = Null, PickTime = Null, Skipped = false, ExpirationTime = $1 Where Id = $2`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare reset pick statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "ResetPick")
	_, err = stmt.ExecContext(ctx, expirationTime, pickId)
	if err != nil {
		return fmt.Errorf("failed to reset pick: %w", err)
	}
	return nil
}

func getDraftsInStatus(ctx context.Context, db database.DBTX, status DraftState) ([]int, error) {
	query := `Select
        Id
    From Drafts
    Where Status = $1;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get drafts in status: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetDraftsInStatus")

	rows, err := stmt.QueryContext(ctx, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get drafts in status: %w", err)
	}
	defer database.CloseRows(ctx, rows, "GetDraftsInStatus")

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

func getDraftScore(ctx context.Context, db database.DBTX, draftId int) ([]DraftPlayer, error) {
	if draftId == 0 {
		return nil, fmt.Errorf("draft id must be greater than zero")
	}

	query := `Select
        dp.Id,
        u.Username,
        p.Pick
    From Picks p
    Inner Join DraftPlayers dp On p.Player = dp.Id
    Inner Join Users u On u.UserUuid = dp.UserUuid
    Where dp.DraftId = $1
	and p.Pick Is Not Null;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft score: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "GetDraftScore")

	rows, err := stmt.QueryContext(ctx, draftId)
	if err != nil {
		return nil, fmt.Errorf("failed to get picks for draft: %w", err)
	}
	defer database.CloseRows(ctx, rows, "GetDraftScore")

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

	uniqueTeams := make(map[string]struct{})
	for _, playerPicks := range picks {
		for _, pick := range playerPicks {
			uniqueTeams[pick] = struct{}{}
		}
	}

	teamTbaIds := make([]string, 0, len(uniqueTeams))
	for tbaId := range uniqueTeams {
		teamTbaIds = append(teamTbaIds, tbaId)
	}

	teamScores, err := getScoresBatch(ctx, db, teamTbaIds)
	if err != nil {
		return nil, fmt.Errorf("failed to get scores for draft: %w", err)
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
			scores, ok := teamScores[pick]
			if !ok {
				return nil, fmt.Errorf("failed to get score for pick %s: %w", pick, sql.ErrNoRows)
			}
			totalScore := scores["Total Score"]
			draftPlayer.Score += totalScore

			team := Pick{
				Pick: sql.NullString{
					Valid:  true,
					String: pick,
				},
				Score: totalScore,
			}
			draftPlayer.Picks = append(draftPlayer.Picks, team)
		}
		playerScores = append(playerScores, draftPlayer)
	}

	// Both maps are populated from the same result set rows, so their key sets must match
	assert.AssertCF(ctx, len(picks) == len(usernames), "Picks and usernames maps have inconsistent lengths")
	return playerScores, nil
}

type LeaderboardEntry struct {
	User      User
	Score     int
	Picks     []Pick
	DraftId   int
	DraftName string
}

type LeaderboardPage struct {
	Entries     []LeaderboardEntry
	CurrentPage int
	TotalPages  int
	PerPage     int
	Total       int
}

func getOverallLeaderboard(ctx context.Context, db database.DBTX, page int, perPage int) (LeaderboardPage, error) {
	query := `Select
		dp.Id,
		u.UserUuid,
		u.Username,
		p.Pick,
		d.Id,
		d.DisplayName
	From Picks p
	Inner Join DraftPlayers dp On p.Player = dp.Id
	Inner Join Users u On u.UserUuid = dp.UserUuid
	Inner Join Drafts d On d.Id = dp.DraftId
	Where p.Pick Is Not Null;`

	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return LeaderboardPage{}, fmt.Errorf("failed to get overall leaderboard: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "getOverallLeaderboard")

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return LeaderboardPage{}, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer database.CloseRows(ctx, rows, "getOverallLeaderboard")

	type rawPick struct {
		dpId      int
		userUuid  uuid.UUID
		username  string
		pick      string
		draftId   int
		draftName string
	}

	var rawPicks []rawPick
	for rows.Next() {
		var r rawPick
		err = rows.Scan(&r.dpId, &r.userUuid, &r.username, &r.pick, &r.draftId, &r.draftName)
		if err != nil {
			return LeaderboardPage{}, fmt.Errorf("failed to scan leaderboard row: %w", err)
		}
		rawPicks = append(rawPicks, r)
	}

	type entryKey struct {
		userUuid uuid.UUID
		draftId  int
	}

	uniqueTeams := make(map[string]struct{})
	for _, r := range rawPicks {
		uniqueTeams[r.pick] = struct{}{}
	}

	teamTbaIds := make([]string, 0, len(uniqueTeams))
	for tbaId := range uniqueTeams {
		teamTbaIds = append(teamTbaIds, tbaId)
	}

	teamScores, err := getScoresBatch(ctx, db, teamTbaIds)
	if err != nil {
		return LeaderboardPage{}, fmt.Errorf("failed to get scores for leaderboard: %w", err)
	}

	entryMap := make(map[entryKey]*LeaderboardEntry)
	for _, r := range rawPicks {
		key := entryKey{userUuid: r.userUuid, draftId: r.draftId}
		entry, ok := entryMap[key]
		if !ok {
			entry = &LeaderboardEntry{
				User:      User{UserUuid: r.userUuid, Username: r.username},
				DraftId:   r.draftId,
				DraftName: r.draftName,
			}
			entryMap[key] = entry
		}

		scores, ok := teamScores[r.pick]
		if !ok {
			return LeaderboardPage{}, fmt.Errorf("failed to get score for pick %s: %w", r.pick, sql.ErrNoRows)
		}
		totalScore := scores["Total Score"]
		entry.Score += totalScore
		entry.Picks = append(entry.Picks, Pick{
			Pick:  sql.NullString{Valid: true, String: r.pick},
			Score: totalScore,
		})
	}

	entries := make([]LeaderboardEntry, 0, len(entryMap))
	for _, entry := range entryMap {
		entries = append(entries, *entry)
	}

	slices.SortFunc(entries, func(a, b LeaderboardEntry) int {
		return b.Score - a.Score
	})

	total := len(entries)
	totalPages := (total + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}

	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * perPage
	end := start + perPage
	if end > total {
		end = total
	}

	pagedEntries := entries[start:end]

	return LeaderboardPage{
		Entries:     pagedEntries,
		CurrentPage: page,
		TotalPages:  totalPages,
		PerPage:     perPage,
		Total:       total,
	}, nil
}

func transferOwnership(ctx context.Context, db database.DBTX, draftId int, newOwnerUuid uuid.UUID) error {
	query := `Update Drafts Set OwnerUserUuid = $1 Where Id = $2;`
	stmt, err := database.Prepare(ctx, db, query)
	if err != nil {
		return fmt.Errorf("failed to prepare transfer ownership statement: %w", err)
	}
	defer database.CloseStatement(ctx, stmt, "TransferOwnership")
	_, err = stmt.ExecContext(ctx, newOwnerUuid, draftId)
	if err != nil {
		log.Error(ctx, "Failed to transfer draft ownership", "draftId", draftId, "newOwnerUuid", newOwnerUuid, "error", err)
		return fmt.Errorf("failed to transfer draft ownership: %w", err)
	}
	log.Info(ctx, "Transferred draft ownership", "draftId", draftId, "newOwnerUuid", newOwnerUuid)
	return nil
}

func CanStartDraft(draftModel DraftModel) bool {
	// Check that eight players have accepted the draft
	numAccepted := 0
	for _, p := range draftModel.Players {
		if !p.Pending {
			numAccepted++
		}
	}
	return numAccepted == DraftPlayerCount
}
