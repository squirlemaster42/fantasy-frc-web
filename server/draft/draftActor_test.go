package draft

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"server/database"
	"server/discord"
	"server/model"
	"server/model/mocks"
	"server/picking"
	"server/swagger"
	"server/tbaHandler"
	"server/utils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type testDiscordStore struct {
	playerDiscordIds map[int]sql.NullString
	webhooks         map[int]string
}

func (t *testDiscordStore) GetPlayerDiscordId(ctx context.Context, draftPlayerId int) (sql.NullString, error) {
	return t.playerDiscordIds[draftPlayerId], nil
}

func (t *testDiscordStore) GetDraftWebhook(ctx context.Context, draftId int) (string, error) {
	return t.webhooks[draftId], nil
}

func newTestActorMap(t *testing.T, draftStore model.DraftStore, handler tbaHandler.TBAInterface, discordStore model.DiscordStore, discordBus discord.DiscordNotifier, pickNotifier *picking.PickNotifier) *DraftActorMap {
	return NewDraftActorMap(draftStore, handler, discordStore, discordBus, pickNotifier, utils.DefaultPickWindowConfig(), 16)
}

type testTBAHandler struct {
	events map[string][]string
	err    error
}

func (t *testTBAHandler) MakeEventListReq(ctx context.Context, teamId string) ([]string, error) {
	if t.err != nil {
		return nil, t.err
	}
	return t.events[teamId], nil
}

func (t *testTBAHandler) MakeMatchReq(ctx context.Context, matchId string) (swagger.Match, error) {
	return swagger.Match{}, nil
}

func (t *testTBAHandler) MakeEventMatchKeysRequest(ctx context.Context, eventId string) ([]string, error) {
	return nil, nil
}

func (t *testTBAHandler) MakeTeamsAtEventRequest(ctx context.Context, eventId string) ([]swagger.Team, error) {
	return nil, nil
}

func (t *testTBAHandler) MakeEliminationAllianceRequest(ctx context.Context, eventId string) ([]swagger.EliminationAlliance, error) {
	return nil, nil
}

func (t *testTBAHandler) MakeTeamAvatarRequest(ctx context.Context, teamId string) (string, error) {
	return "", nil
}

func mockRunInTransaction(mockStore *mocks.MockDraftStore) {
	mockStore.On("WithTx", mock.Anything).Return(mockStore).Maybe()
	mockStore.On("RunInTransaction", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(database.DBTX) error)
		_ = fn(nil)
	}).Return(nil).Once()
}

func TestDraftActorMap_GetActor_CachesActor(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{Id: draftId}, nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)

	// First call creates the actor
	actor1, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)
	assert.NotNil(t, actor1)

	// Second call returns cached actor
	actor2, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)
	assert.Equal(t, actor1, actor2)

	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_GetActor_ReturnsError(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{}, errors.New("db error")).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)

	actor, err := actorMap.GetActor(t.Context(), draftId)
	assert.Error(t, err)
	assert.Nil(t, actor)
	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_SkipCurrentPick(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	pickId := 42
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id: draftId,
		CurrentPick: model.Pick{Id: pickId},
		Players: []model.DraftPlayer{
			{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
		},
	}, nil).Once()
	mockRunInTransaction(mockStore)
	mockStore.On("SkipPick", mock.Anything, pickId).Return(nil).Once()
	mockStore.On("MakePickAvailable", mock.Anything, 1, mock.Anything, mock.Anything).Return(0, nil).Once()
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id: draftId,
		CurrentPick: model.Pick{Id: pickId},
		Players: []model.DraftPlayer{
			{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
		},
	}, nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)

	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	skipped := SkipCurrentPick(t.Context(), draftActor, draftId, draftActor.GetDraftState().CurrentPick.Id)
	assert.True(t, skipped)
	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_SkipCurrentPick_At64DoesNotCreate65th(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	pickId := 42

	// Build PicksPerDraft picks so the draft is at the final pick
	picks := make([]model.Pick, model.PicksPerDraft)
	for i := range picks {
		picks[i] = model.Pick{Id: i + 1}
	}

	players := []model.DraftPlayer{
		{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
	}

	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: pickId},
		Picks:       picks,
		Players:     players,
	}, nil).Once()
	mockRunInTransaction(mockStore)
	mockStore.On("SkipPick", mock.Anything, pickId).Return(nil).Once()
	// State transition to TEAMS_PLAYING after the last pick
	mockStore.On("UpdateDraftStatus", mock.Anything, draftId, model.TEAMS_PLAYING).Return(nil).Once()
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.TEAMS_PLAYING,
		CurrentPick: model.Pick{Id: pickId},
		Picks:       picks,
		Players:     players,
	}, nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)

	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	skipped := SkipCurrentPick(t.Context(), draftActor, draftId, draftActor.GetDraftState().CurrentPick.Id)
	assert.True(t, skipped)

	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_AcceptInvite(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	inviteId := 123
	userUuid := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id: draftId,
	}, nil).Once()
	mockStore.On("GetInvite", mock.Anything, inviteId).Return(model.DraftInvite{
		Id:              inviteId,
		DraftId:         draftId,
		InvitedUserUuid: userUuid,
	}, nil).Once()
	mockRunInTransaction(mockStore)
	mockStore.On("LockDraft", mock.Anything, draftId).Return(nil).Once()
	mockStore.On("GetNumPlayersInDraft", mock.Anything, draftId).Return(3, nil).Once()
	mockStore.On("AcceptInvite", mock.Anything, inviteId).Return(draftId, userUuid, nil).Once()
	mockStore.On("AddPlayerToDraft", mock.Anything, draftId, userUuid).Return(nil).Once()
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id: draftId,
	}, nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)
	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	err = AcceptInvite(t.Context(), draftActor, inviteId, userUuid)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_SkipCurrentPick_SendsDiscordNotification(t *testing.T) {
	received := make(chan discord.DiscordWebhook, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var webhook discord.DiscordWebhook
		err = json.Unmarshal(body, &webhook)
		assert.NoError(t, err)

		received <- webhook
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	mockStore := mocks.NewMockDraftStore(t)
	discordStore := &testDiscordStore{
		playerDiscordIds: map[int]sql.NullString{
			3: {String: "12345678901234567", Valid: true},
			4: {String: "98765432109876543", Valid: true},
		},
		webhooks: map[int]string{
			1: server.URL,
		},
	}

	draftId := 1
	pickId := 42
	players := []model.DraftPlayer{
		{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}, User: model.User{Username: "Alice"}},
		{Id: 2, PlayerOrder: sql.NullInt16{Int16: 1, Valid: true}, User: model.User{Username: "Bob"}},
		{Id: 3, PlayerOrder: sql.NullInt16{Int16: 2, Valid: true}, User: model.User{Username: "Charlie"}},
		{Id: 4, PlayerOrder: sql.NullInt16{Int16: 3, Valid: true}, User: model.User{Username: "David"}},
	}

	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: pickId, Player: 3},
		Picks: []model.Pick{
			{Id: 1, Player: 1},
			{Id: 2, Player: 2},
			{Id: pickId, Player: 3},
		},
		Players: players,
	}, nil).Once()
	mockRunInTransaction(mockStore)
	mockStore.On("SkipPick", mock.Anything, pickId).Return(nil).Once()
	mockStore.On("MakePickAvailable", mock.Anything, 4, mock.Anything, mock.Anything).Return(43, nil).Once()
	mockStore.On("GetDraftPlayerUser", mock.Anything, 3).Return(model.User{Username: "Charlie"}, nil).Once()
	mockStore.On("GetDraftPlayerUser", mock.Anything, 4).Return(model.User{Username: "David"}, nil).Once()
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: 43, Player: 4},
		Picks: []model.Pick{
			{Id: 1, Player: 1},
			{Id: 2, Player: 2},
			{Id: pickId, Player: 3, Skipped: true},
		},
		Players: players,
	}, nil).Once()

	bus := discord.NewBus()
	defer bus.Stop()

	actorMap := newTestActorMap(t, mockStore, nil, discordStore, bus, nil)

	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	skipped := SkipCurrentPick(t.Context(), draftActor, draftId, draftActor.GetDraftState().CurrentPick.Id)
	assert.True(t, skipped)

	select {
	case webhook := <-received:
		assert.Equal(t, "Pick Notifier", webhook.Username)
		assert.Contains(t, webhook.Content, "<@12345678901234567>")
		assert.Contains(t, webhook.Content, "<@98765432109876543>")
		assert.Contains(t, webhook.Content, "your pick was skipped")
		assert.Contains(t, webhook.Content, "it is now your pick")
	case <-time.After(2 * time.Second):
		t.Fatal("expected discord webhook to be received")
	}

	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_ModifyCurrentPickExpirationTime(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	pickId := 42
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id: draftId,
		CurrentPick: model.Pick{Id: pickId, ExpirationTime: time.Now()},
	}, nil).Once()
	mockStore.On("UpdatePickExpirationTime", mock.Anything, pickId, mock.Anything).Return(nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)
	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	err = ModifyCurrentPickExpirationTime(t.Context(), draftActor, 30*time.Minute)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_GetCurrentPick(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	expectedPick := model.Pick{Id: 42}
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		CurrentPick: expectedPick,
	}, nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)
	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	pick := GetCurrentPick(draftActor)
	assert.Equal(t, expectedPick.Id, pick.Id)
	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_UndoLastPick(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	pickId := 42
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id: draftId,
		CurrentPick: model.Pick{Id: pickId},
	}, nil).Once()
	mockStore.On("GetPreviousPick", mock.Anything, draftId, pickId).Return(model.Pick{Id: 41}, nil).Once()
	mockRunInTransaction(mockStore)
	mockStore.On("DeletePick", mock.Anything, pickId).Return(nil).Once()
	mockStore.On("ResetPick", mock.Anything, 41, mock.Anything).Return(nil).Once()
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id: draftId,
		CurrentPick: model.Pick{Id: 41},
	}, nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)
	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	err = UndoLastPick(t.Context(), draftActor)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_GetDraft(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	expectedDraft := model.DraftModel{Id: draftId, DisplayName: "Test Draft"}
	mockStore.On("GetDraft", mock.Anything, draftId).Return(expectedDraft, nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)
	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	draft := GetDraft(draftActor)
	assert.Equal(t, expectedDraft.DisplayName, draft.DisplayName)
	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_UpdateDraft(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{Id: draftId}, nil).Once()
	mockStore.On("UpdateDraft", mock.Anything, mock.Anything).Return(nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)
	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	err = UpdateDraft(t.Context(), draftActor, model.DraftModel{
		Id:          draftId,
		DisplayName: "Updated",
	})
	assert.NoError(t, err)

	// Verify cached state was updated directly without re-querying
	draft := GetDraft(draftActor)
	assert.Equal(t, "Updated", draft.DisplayName)
	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_ExecuteDraftStateTransition(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	nextPickPlayer := model.DraftPlayer{Id: 10}

	// Initial load when the actor is created
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:     draftId,
		Status: model.FILLING,
	}, nil).Once()

	// FILLING -> PICKING transition now randomizes order and sets up the first pick
	mockRunInTransaction(mockStore)
	mockStore.On("RandomizePickOrder", mock.Anything, draftId).Return(nil).Once()
	mockStore.On("NextPick", mock.Anything, draftId).Return(nextPickPlayer, nil).Once()
	mockStore.On("MakePickAvailable", mock.Anything, nextPickPlayer.Id, mock.Anything, mock.Anything).Return(1, nil).Once()
	mockStore.On("UpdateDraftStatus", mock.Anything, draftId, model.PICKING).Return(nil).Once()

	// Reload draft state so cached model is not stale
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:     draftId,
		Status: model.PICKING,
	}, nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)
	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	err = ExecuteDraftStateTransition(t.Context(), draftActor, model.PICKING)
	assert.NoError(t, err)

	// Verify cached state was reloaded after transition
	draft := GetDraft(draftActor)
	assert.Equal(t, model.PICKING, draft.Status)
	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_RegisterAndUnregisterWatcher(t *testing.T) {
	notifier := &picking.PickNotifier{
		Watchers: make(map[int][]picking.Watcher),
	}
	actorMap := newTestActorMap(t, nil, nil, nil, nil, notifier)

	draftId := 1
	watcher := RegisterWatcher(t.Context(), actorMap, draftId)
	assert.NotNil(t, watcher)
	assert.NotNil(t, watcher.NotifierQueue)

	// Verify watcher receives events before unregister
	event := picking.PickEvent{DraftId: draftId, Pick: model.Pick{Id: 1}}
	err := notifier.ReceivePickEvent(t.Context(), event)
	assert.NoError(t, err)

	select {
	case <-watcher.NotifierQueue:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("watcher should have received event")
	}

	UnregisterWatcher(t.Context(), actorMap, watcher)

	// After unregister, watcher should not receive new events
	select {
	case <-watcher.NotifierQueue:
		t.Fatal("watcher should NOT have received event after unregister")
	case <-time.After(100 * time.Millisecond):
		// Success
	}
}

func TestDraftActor_handleMessage_UnknownType(t *testing.T) {
	actor := &DraftActor{
		inbox: make(chan Message, 1),
	}

	result := actor.handleMessage(Message{
		Content: "unknown string type",
	})

	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "unknown message type")
}

func TestDraftActor_handleTransferDraftOwnership_Success(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	newOwnerUuid := uuid.New()
	actor := &DraftActor{
		draftStore: mockStore,
		draftState: model.DraftModel{
			Id:    1,
			Owner: model.User{UserUuid: uuid.New()},
		},
	}
	mockStore.On("TransferOwnership", mock.Anything, 1, newOwnerUuid).Return(nil).Once()

	result := actor.handleTransferDraftOwnership(t.Context(), TransferDraftOwnershipMessage{
		UpdatedOwnerId: newOwnerUuid,
	})

	assert.NoError(t, result.Error)
	assert.Equal(t, newOwnerUuid, actor.draftState.Owner.UserUuid)
	mockStore.AssertExpectations(t)
}

func TestPickNotifier_ReceivePickEvent_SkipsSlowWatchers(t *testing.T) {
	notifier := &picking.PickNotifier{
		Watchers: make(map[int][]picking.Watcher),
	}

	draftId := 1
	watcher1 := notifier.RegisterWatcher(draftId)
	watcher2 := notifier.RegisterWatcher(draftId)

	// Block watcher1 so it times out
	go func() {
		<-watcher1.NotifierQueue
	}()

	// Fill watcher2's buffer so it also times out
	for i := 0; i < 10; i++ {
		select {
		case watcher2.NotifierQueue <- true:
		default:
		}
	}

	event := picking.PickEvent{
		DraftId: draftId,
		Pick:    model.Pick{Id: 1},
	}

	// Should not return error even if watchers are slow
	err := notifier.ReceivePickEvent(t.Context(), event)
	assert.NoError(t, err)

	// Clean up
	notifier.UnregisterWatcher(t.Context(), watcher1)
	notifier.UnregisterWatcher(t.Context(), watcher2)
}

func TestPickNotifier_UnregisterWatcher_CleansUpEmptyEntries(t *testing.T) {
	notifier := &picking.PickNotifier{
		Watchers: make(map[int][]picking.Watcher),
	}

	draftId := 1
	watcher := notifier.RegisterWatcher(draftId)

	// Verify watcher receives events before unregister
	event := picking.PickEvent{DraftId: draftId, Pick: model.Pick{Id: 1}}
	err := notifier.ReceivePickEvent(t.Context(), event)
	assert.NoError(t, err)

	select {
	case <-watcher.NotifierQueue:
		// Success
	case <-time.After(100 * time.Millisecond):
		t.Fatal("watcher should have received event")
	}

	notifier.UnregisterWatcher(t.Context(), watcher)

	// After unregister, watcher should not receive new events
	// (the event will be sent to zero watchers, which is fine)
	err = notifier.ReceivePickEvent(t.Context(), event)
	assert.NoError(t, err)

	select {
	case <-watcher.NotifierQueue:
		t.Fatal("watcher should NOT have received event after unregister")
	case <-time.After(100 * time.Millisecond):
		// Success - no event received
	}
}

func TestDraftActorMap_ModifyCurrentPickExpirationTime_StalePickId(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	currentPickId := 42
	stalePickId := 99
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		CurrentPick: model.Pick{Id: currentPickId, ExpirationTime: time.Now()},
	}, nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)
	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	// First, test successful modification
	mockStore.On("UpdatePickExpirationTime", mock.Anything, currentPickId, mock.Anything).Return(nil).Once()
	err = ModifyCurrentPickExpirationTime(t.Context(), draftActor, 30*time.Minute)
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)

	// Now try with a stale pick ID by creating a new actor map and faking the pick ID mismatch
	// We can't easily test this through the actor map since it reads current pick internally,
	// but we can test the actor directly
	actor := &DraftActor{
		draftState: model.DraftModel{
			Id:          draftId,
			CurrentPick: model.Pick{Id: currentPickId},
		},
	}
	result := actor.handleModifyExpirationTime(t.Context(), ModifyExpirationTimeMessage{
		PickId:    stalePickId,
		Extension: 30 * time.Minute,
	})
	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "pick id does not match current pick")
}

func TestDraftActor_Shutdown(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{Id: draftId}, nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)
	actor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)
	assert.NotNil(t, actor)

	// Shutdown the actor
	err = ShutdownActor(actorMap, t.Context(), draftId)
	assert.NoError(t, err)

	// Verify actor is removed from map
	assert.False(t, actorMap.actorCache.Contains(draftId), "actor should be removed from map after shutdown")

	// Posting a message to a shutdown actor should return an error
	msg := Message{Content: StateTransitionMessage{RequestedState: model.FILLING}}
	err = actor.PostMessage(t.Context(), msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shutting down")
}

func TestDraftActorMap_ConcurrentGetActor(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{Id: draftId}, nil).Once()

	actorMap := newTestActorMap(t, mockStore, nil, nil, nil, nil)

	var actors []*DraftActor
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			actor, err := actorMap.GetActor(t.Context(), draftId)
			assert.NoError(t, err)
			assert.NotNil(t, actor)
			mu.Lock()
			actors = append(actors, actor)
			mu.Unlock()
		}()
	}

	wg.Wait()

	// All goroutines should have received the same actor instance
	for i := 1; i < len(actors); i++ {
		assert.Equal(t, actors[0], actors[i], "all concurrent GetActor calls should return the same instance")
	}

	mockStore.AssertExpectations(t)
}

func TestPickNotifier_ConcurrentOperations(t *testing.T) {
	notifier := &picking.PickNotifier{
		Watchers: make(map[int][]picking.Watcher),
	}

	draftId := 1
	var wg sync.WaitGroup

	// Concurrent register/unregister
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			watcher := notifier.RegisterWatcher(draftId)
			time.Sleep(10 * time.Millisecond)
			notifier.UnregisterWatcher(t.Context(), watcher)
		}()
	}

	// Concurrent events
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			event := picking.PickEvent{
				DraftId: draftId,
				Pick:    model.Pick{Id: i},
			}
			err := notifier.ReceivePickEvent(t.Context(), event)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()

	// After all operations, there should be no watchers left
	assert.Empty(t, notifier.Watchers[draftId], "all watchers should be unregistered")
}

func TestDraftActor_ConcurrentMessages(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:     draftId,
		Status: model.FILLING,
		CurrentPick: model.Pick{Id: 42},
	}, nil).Once()
	mockStore.On("TransferOwnership", mock.Anything, draftId, uuid.Nil).Return(nil).Maybe()

	actor, err := NewDraftActor(t.Context(), draftId, mockStore, nil, nil, nil, nil, utils.DefaultPickWindowConfig())
	assert.NoError(t, err)
	assert.NotNil(t, actor)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			replyChan := make(chan Result)
			var msg Message
			switch idx % 3 {
			case 0:
				// ModifyExpirationTimeMessage: fails early with "pick id does not match" (no DB call)
				msg = Message{
					Content: ModifyExpirationTimeMessage{PickId: 999, Extension: time.Minute},
					Reply:   replyChan,
				}
			case 1, 2:
				// TransferDraftOwnershipMessage: calls TransferOwnership on the store
				msg = Message{
					Content: TransferDraftOwnershipMessage{Initiator: idx},
					Reply:   replyChan,
				}
			}
			err := actor.PostMessage(t.Context(), msg)
			if err != nil {
				return
			}
			select {
			case <-replyChan:
			case <-time.After(time.Second):
				t.Log("timeout waiting for reply")
			}
		}(i)
	}

	wg.Wait()
}

func TestDraftActor_getPreviousPick_Errors(t *testing.T) {
	actor := &DraftActor{
		draftState: model.DraftModel{
			Picks: []model.Pick{},
		},
	}

	// No picks
	pick, err := actor.getPreviousPick(t.Context())
	assert.Error(t, err)
	assert.Equal(t, model.Pick{}, pick)

	// Only one pick
	actor.draftState.Picks = []model.Pick{{Id: 1}}
	pick, err = actor.getPreviousPick(t.Context())
	assert.Error(t, err)
	assert.Equal(t, model.Pick{}, pick)

	// Two picks - should return the first
	actor.draftState.Picks = []model.Pick{{Id: 1}, {Id: 2}}
	pick, err = actor.getPreviousPick(t.Context())
	assert.NoError(t, err)
	assert.Equal(t, 1, pick.Id)
}

func TestDraftActorMap_MakePick(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	pickId := 42
	teamId := "frc254"

	players := []model.DraftPlayer{
		{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
		{Id: 2, PlayerOrder: sql.NullInt16{Int16: 1, Valid: true}},
	}

	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: pickId, Player: 1},
		Picks: []model.Pick{
			{Id: pickId, Player: 1},
		},
		Players: players,
	}, nil).Once()
	mockStore.On("HasBeenPicked", mock.Anything, draftId, teamId).Return(false, nil).Once()
	mockRunInTransaction(mockStore)
	mockStore.On("MakePick", mock.Anything, mock.Anything).Return(nil).Once()
	mockStore.On("MakePickAvailable", mock.Anything, 2, mock.Anything, mock.Anything).Return(43, nil).Once()
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: 43, Player: 2},
		Picks: []model.Pick{
			{Id: pickId, Player: 1, Pick: sql.NullString{Valid: true, String: teamId}},
			{Id: 43, Player: 2},
		},
		Players: players,
	}, nil).Once()

	handler := &testTBAHandler{
		events: map[string][]string{
			teamId: {utils.Events()[0]},
		},
	}

	actorMap := newTestActorMap(t, mockStore, handler, nil, nil, nil)
	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	err = MakePick(t.Context(), draftActor, model.Pick{
		Id:     pickId,
		Player: 1,
		Pick:   sql.NullString{Valid: true, String: teamId},
	})
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}

func TestDraftActorMap_MakePick_FinalPickTransitionsToTeamsPlaying(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	pickId := 64
	teamId := "frc254"

	picks := make([]model.Pick, model.PicksPerDraft)
	for i := range picks {
		picks[i] = model.Pick{Id: i + 1, Player: 1}
	}
	picks[model.PicksPerDraft-1] = model.Pick{Id: pickId, Player: 1}

	players := []model.DraftPlayer{
		{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
	}

	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: pickId, Player: 1},
		Picks:       picks,
		Players:     players,
	}, nil).Once()
	mockStore.On("HasBeenPicked", mock.Anything, draftId, teamId).Return(false, nil).Once()
	mockRunInTransaction(mockStore)
	mockStore.On("MakePick", mock.Anything, mock.Anything).Return(nil).Once()
	mockStore.On("UpdateDraftStatus", mock.Anything, draftId, model.TEAMS_PLAYING).Return(nil).Once()
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.TEAMS_PLAYING,
		CurrentPick: model.Pick{Id: pickId, Player: 1},
		Picks:       picks,
		Players:     players,
	}, nil).Once()

	handler := &testTBAHandler{
		events: map[string][]string{
			teamId: {utils.Events()[0]},
		},
	}

	actorMap := newTestActorMap(t, mockStore, handler, nil, nil, nil)
	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	err = MakePick(t.Context(), draftActor, model.Pick{
		Id:     pickId,
		Player: 1,
		Pick:   sql.NullString{Valid: true, String: teamId},
	})
	assert.NoError(t, err)
	mockStore.AssertExpectations(t)
}
