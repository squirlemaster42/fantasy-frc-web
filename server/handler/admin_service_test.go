package handler

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

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
