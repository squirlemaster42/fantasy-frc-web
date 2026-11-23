package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"server/background"
	"server/draft"
	"server/model"
	"server/model/mocks"
	"server/swagger"
	"server/utils"
)

func TestAdminServicePing(t *testing.T) {
	svc := &AdminService{}
	msg, err := svc.Ping(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Pong", msg)
}

func TestAdminServicePopulateTeams(t *testing.T) {
	ctx := context.Background()

	tbaHandler := &mockAdminTBAHandler{
		teams: map[string][]swagger.Team{
			"2026arc": {
				{Key: "frc254"},
				{Key: "frc1114"},
			},
		},
	}

	mockTeamStore := mocks.NewMockTeamStore(t)
	mockTeamStore.On("GetTeam", ctx, "frc254").Return(nil, nil).Once()
	mockTeamStore.On("CreateTeam", ctx, "frc254", "").Return(nil).Once()
	mockTeamStore.On("GetTeam", ctx, "frc1114").Return(&model.Team{TbaId: "frc1114"}, nil).Once()

	svc := &AdminService{TBAHandler: tbaHandler, TeamStore: mockTeamStore}
	msg, err := svc.PopulateTeams(ctx)

	require.NoError(t, err)
	assert.Equal(t, "Successfully populated 1 teams", msg)
	mockTeamStore.AssertExpectations(t)
}

func TestAdminServiceListDrafts(t *testing.T) {
	ctx := context.Background()

	t.Run("returns formatted draft list", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraftsByName", ctx, "").Return([]model.DraftModel{
			{Id: 1, DisplayName: "Test Draft One"},
			{Id: 42, DisplayName: "Test Draft Two"},
		}, nil)

		svc := &AdminService{DraftStore: mockDraftStore}
		result, err := svc.ListDrafts(ctx, "")

		require.NoError(t, err)
		assert.Contains(t, result, "Id    |  Name")
		assert.Contains(t, result, "   1  | Test Draft One")
		assert.Contains(t, result, "  42  | Test Draft Two")
	})

	t.Run("empty result shows headers only", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraftsByName", ctx, "").Return([]model.DraftModel{}, nil)

		svc := &AdminService{DraftStore: mockDraftStore}
		result, err := svc.ListDrafts(ctx, "")

		require.NoError(t, err)
		assert.Contains(t, result, "Id    |  Name")
		assert.NotContains(t, result, "Test Draft")
	})

	t.Run("search filter passed to store", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraftsByName", ctx, "playoffs").Return([]model.DraftModel{
			{Id: 7, DisplayName: "Playoffs Draft"},
		}, nil)

		svc := &AdminService{DraftStore: mockDraftStore}
		result, err := svc.ListDrafts(ctx, "playoffs")

		require.NoError(t, err)
		assert.Contains(t, result, "   7  | Playoffs Draft")
	})
}

func TestAdminServiceStartDraft(t *testing.T) {
	ctx := context.Background()

	t.Run("draft not found", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", ctx, 999).Return(model.DraftModel{}, sql.ErrNoRows)
		draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

		svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
		_, err := svc.StartDraft(ctx, 999)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "draft ID does not match a valid draft")
	})

	t.Run("not enough accepted players", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", ctx, 5).Return(model.DraftModel{
			Id: 5,
			Players: []model.DraftPlayer{
				{Pending: false},
				{Pending: true},
			},
		}, nil)
		draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

		svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
		_, err := svc.StartDraft(ctx, 5)

		require.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("draft must have between %d and %d accepted players to start", model.MinDraftPlayers, model.MaxDraftPlayers))
	})

	t.Run("starts and watches draft", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		draftId := 1

		players := make([]model.DraftPlayer, 8)
		for i := range players {
			players[i] = model.DraftPlayer{Pending: false}
		}

		mockDraftStore.On("GetDraft", ctx, draftId).Return(model.DraftModel{
			Id:      draftId,
			Status:  model.FILLING,
			Players: players,
		}, nil).Once()
		mockDraftStore.On("CancelOutstandingInvites", ctx, draftId).Return(nil).Once()
		mockRunInTransaction(mockDraftStore)
		mockDraftStore.On("RandomizePickOrder", ctx, draftId).Return(nil).Once()
		mockDraftStore.On("NextPick", ctx, draftId).Return(model.DraftPlayer{Id: 1}, nil).Once()
		mockDraftStore.On("MakePickAvailable", ctx, 1, mock.Anything, mock.Anything).Return(1, nil).Once()
		mockDraftStore.On("UpdateDraftStatus", ctx, draftId, model.PICKING).Return(nil).Once()
		mockDraftStore.On("GetDraft", ctx, draftId).Return(model.DraftModel{
			Id:      draftId,
			Status:  model.PICKING,
			Players: players,
		}, nil).Once()

		draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)
		draftDaemon := background.NewDraftDaemon(mockDraftStore, draftActorMap)

		svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap, DraftDaemon: draftDaemon}
		msg, err := svc.StartDraft(ctx, draftId)

		require.NoError(t, err)
		assert.Equal(t, "Draft Started", msg)
		err = draftDaemon.AddDraft(ctx, draftId)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already added")
		mockDraftStore.AssertExpectations(t)
	})
}

func TestAdminServiceSkipPick(t *testing.T) {
	ctx := context.Background()

	t.Run("draft not found", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", ctx, 999).Return(model.DraftModel{}, assert.AnError)
		draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

		svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
		_, err := svc.SkipPick(ctx, 999)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "draft ID does not match a valid draft")
	})
}

func TestAdminServiceModifyPickTime(t *testing.T) {
	ctx := context.Background()
	draftId := 1
	pickId := 42

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", ctx, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: pickId},
		Players: []model.DraftPlayer{
			{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
		},
	}, nil).Once()
	mockDraftStore.On("UpdatePickExpirationTime", ctx, pickId, mock.Anything).Return(nil).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
	msg, err := svc.ModifyPickTime(ctx, draftId, 45*time.Minute)

	require.NoError(t, err)
	assert.Equal(t, "Successfully updated pick expiration time", msg)
	mockDraftStore.AssertExpectations(t)
}

func TestAdminServiceAdminPick(t *testing.T) {
	ctx := context.Background()
	draftId := 1

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", ctx, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: 0},
		Players: []model.DraftPlayer{
			{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
		},
	}, nil).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
	_, err := svc.AdminPick(ctx, draftId, "254")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no current pick found for this draft")
}

func TestAdminServiceRenameDraft(t *testing.T) {
	ctx := context.Background()
	draftId := 1

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", ctx, draftId).Return(model.DraftModel{
		Id:          draftId,
		DisplayName: "Old Name",
		Players: []model.DraftPlayer{
			{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
		},
	}, nil).Once()
	mockDraftStore.On("UpdateDraft", ctx, mock.Anything).Return(nil).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
	msg, err := svc.RenameDraft(ctx, draftId, "NewName")

	require.NoError(t, err)
	assert.Equal(t, "Successfully renamed draft from 'Old Name' to 'NewName'", msg)
	mockDraftStore.AssertExpectations(t)
}

func TestAdminServiceUndoPick(t *testing.T) {
	ctx := context.Background()

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", ctx, 999).Return(model.DraftModel{}, assert.AnError)
	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
	_, err := svc.UndoPick(ctx, 999)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "draft ID does not match a valid draft")
}

func TestAdminServiceViewWebhookKey(t *testing.T) {
	ctx := context.Background()
	secretPath := "./webhookSecret.txt" //nolint:gosec // test fixture path, not a credential

	t.Run("reads existing key file", func(t *testing.T) {
		require.NoError(t, os.WriteFile(secretPath, []byte("super-secret-key"), 0600))
		defer func() { _ = os.Remove(secretPath) }()

		svc := &AdminService{}
		msg, err := svc.ViewWebhookKey(ctx)

		require.NoError(t, err)
		assert.Equal(t, "super-secret-key", msg)
	})

	t.Run("returns error when file missing", func(t *testing.T) {
		_ = os.Remove(secretPath)

		svc := &AdminService{}
		_, err := svc.ViewWebhookKey(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open file")
	})
}

func TestAdminServiceAdminPick_Success(t *testing.T) {
	ctx := context.Background()
	draftId := 1

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	initialModel := makeAdminDraftModel(userUuid)
	reloadedModel := initialModel
	reloadedModel.CurrentPick.Id = 11

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", ctx, draftId).Return(initialModel, nil).Once()
	mockDraftStore.On("HasBeenPicked", ctx, draftId, "frc254").Return(false, nil).Once()
	mockDraftStore.On("GetDraft", ctx, draftId).Return(reloadedModel, nil).Once()
	mockRunInTransaction(mockDraftStore)
	mockDraftStore.On("MakePick", ctx, mock.Anything).Return(nil).Once()
	mockDraftStore.On("MakePickAvailable", ctx, 5, mock.Anything, mock.Anything).Return(11, nil).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, &stubEventTeamCache{})

	svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
	msg, err := svc.AdminPick(ctx, draftId, "254")

	require.NoError(t, err)
	assert.Equal(t, "Successfully picked team 254", msg)
	mockDraftStore.AssertExpectations(t)
}

func TestAdminServiceUndoPick_Success(t *testing.T) {
	ctx := context.Background()
	draftId := 1

	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	draftModel := makeAdminDraftModel(userUuid)
	draftModel.CurrentPick.Id = 11

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", ctx, draftId).Return(draftModel, nil).Twice()
	mockDraftStore.On("GetPreviousPick", ctx, draftId, 11).Return(model.Pick{Id: 10, Player: 5}, nil).Once()
	mockRunInTransaction(mockDraftStore)
	mockDraftStore.On("DeletePick", ctx, 11).Return(nil).Once()
	mockDraftStore.On("ResetPick", ctx, 10, mock.Anything).Return(nil).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
	msg, err := svc.UndoPick(ctx, draftId)

	require.NoError(t, err)
	assert.Equal(t, "Successfully undid pick", msg)
	mockDraftStore.AssertExpectations(t)
}

func TestAdminServiceModifyPickTime_UpdateError(t *testing.T) {
	ctx := context.Background()
	draftId := 1
	pickId := 42

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", ctx, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: pickId},
		Players: []model.DraftPlayer{
			{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
		},
	}, nil).Once()
	mockDraftStore.On("UpdatePickExpirationTime", ctx, pickId, mock.Anything).Return(errors.New("db down")).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
	_, err := svc.ModifyPickTime(ctx, draftId, 45*time.Minute)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update pick expiration time")
}

func TestAdminServiceModifyPickTime_ActorNotFound(t *testing.T) {
	ctx := context.Background()
	draftId := 999

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", ctx, draftId).Return(model.DraftModel{}, sql.ErrNoRows).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
	_, err := svc.ModifyPickTime(ctx, draftId, 45*time.Minute)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "draft ID does not match a valid draft")
}

func TestAdminServiceInvalidatePickCache(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mockCache := &mockEventTeamCache{lastRefresh: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

		svc := &AdminService{EventTeamCache: mockCache}
		msg, err := svc.InvalidatePickCache(ctx)

		require.NoError(t, err)
		assert.True(t, mockCache.invalidateCalled)
		assert.Contains(t, msg, "Pick validation cache invalidated and refreshed")
		assert.Contains(t, msg, "2026-01-01T00:00:00Z")
	})

	t.Run("nil cache", func(t *testing.T) {
		svc := &AdminService{}
		_, err := svc.InvalidatePickCache(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "pick validation cache is not configured")
	})

	t.Run("refresh error", func(t *testing.T) {
		mockCache := &mockEventTeamCache{refreshErr: fmt.Errorf("tba down")}

		svc := &AdminService{EventTeamCache: mockCache}
		_, err := svc.InvalidatePickCache(ctx)

		require.Error(t, err)
		assert.True(t, mockCache.invalidateCalled)
		assert.Contains(t, err.Error(), "failed to invalidate pick validation cache")
		assert.Contains(t, err.Error(), "tba down")
	})
}

func TestAdminServiceStartDraft_CancelInvitesError(t *testing.T) {
	ctx := context.Background()
	draftId := 1

	players := make([]model.DraftPlayer, 8)
	for i := range players {
		players[i] = model.DraftPlayer{Pending: false}
	}

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", ctx, draftId).Return(model.DraftModel{
		Id:      draftId,
		Status:  model.FILLING,
		Players: players,
	}, nil).Once()
	mockDraftStore.On("CancelOutstandingInvites", ctx, draftId).Return(errors.New("db down")).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
	_, err := svc.StartDraft(ctx, draftId)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to cancel outstanding invites")
}

func TestAdminServiceAdminPick_PickError(t *testing.T) {
	ctx := context.Background()
	draftId := 1
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", ctx, draftId).Return(makeAdminDraftModel(userUuid), nil).Once()
	mockDraftStore.On("HasBeenPicked", ctx, draftId, "frc254").Return(false, nil).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
	_, err := svc.AdminPick(ctx, draftId, "254")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "pick validation cache is not available")
}

func TestAdminServiceUndoPick_Error(t *testing.T) {
	ctx := context.Background()
	draftId := 1
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	draftModel := makeAdminDraftModel(userUuid)
	draftModel.CurrentPick.Id = 11

	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraft", ctx, draftId).Return(draftModel, nil).Once()
	mockDraftStore.On("GetPreviousPick", ctx, draftId, 11).Return(model.Pick{}, sql.ErrNoRows).Once()

	draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

	svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
	_, err := svc.UndoPick(ctx, draftId)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get previous pick")
}

func TestAdminServiceRenameDraft_Errors(t *testing.T) {
	ctx := context.Background()
	draftId := 1

	t.Run("draft not found", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", ctx, draftId).Return(model.DraftModel{}, sql.ErrNoRows).Once()
		draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

		svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
		_, err := svc.RenameDraft(ctx, draftId, "NewName")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "draft ID does not match a valid draft")
	})

	t.Run("update error", func(t *testing.T) {
		mockDraftStore := mocks.NewMockDraftStore(t)
		mockDraftStore.On("GetDraft", ctx, draftId).Return(model.DraftModel{
			Id:          draftId,
			DisplayName: "Old Name",
			Players: []model.DraftPlayer{
				{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
			},
		}, nil).Once()
		mockDraftStore.On("UpdateDraft", ctx, mock.Anything).Return(errors.New("db down")).Once()
		draftActorMap := draft.NewDraftActorMap(mockDraftStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig(), 16, nil)

		svc := &AdminService{DraftStore: mockDraftStore, DraftActorMap: draftActorMap}
		_, err := svc.RenameDraft(ctx, draftId, "NewName")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update draft name")
	})
}

func TestAdminServicePopulateTeams_TBAError(t *testing.T) {
	ctx := context.Background()

	tbaHandler := &mockAdminTBAHandler{err: errors.New("tba down")}

	svc := &AdminService{TBAHandler: tbaHandler}
	msg, err := svc.PopulateTeams(ctx)

	require.NoError(t, err)
	assert.Equal(t, "Successfully populated 0 teams", msg)
}
