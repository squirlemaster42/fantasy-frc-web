package draft

import (
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"server/model"
	"server/model/mocks"
	"server/picking"
)

func TestDraftActorMap_GetActor_CachesActor(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	draftId := 1
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{Id: draftId}, nil).Once()

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)

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

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)

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
	mockStore.On("SkipPick", mock.Anything, pickId).Return(nil).Once()
	mockStore.On("MakePickAvailable", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil).Once()
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id: draftId,
		CurrentPick: model.Pick{Id: pickId},
		Players: []model.DraftPlayer{
			{Id: 1, PlayerOrder: sql.NullInt16{Int16: 0, Valid: true}},
		},
	}, nil).Once()

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)

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

	// Build 64 picks so the draft is at the final pick
	picks := make([]model.Pick, 64)
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
	mockStore.On("SkipPick", mock.Anything, pickId).Return(nil).Once()
	// MakePickAvailable should NOT be called when the draft is already at 64 picks
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.PICKING,
		CurrentPick: model.Pick{Id: pickId},
		Picks:       picks,
		Players:     players,
	}, nil).Once()
	// State transition to TEAMS_PLAYING after the last pick
	mockStore.On("UpdateDraftStatus", mock.Anything, draftId, model.TEAMS_PLAYING).Return(nil).Once()
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:          draftId,
		Status:      model.TEAMS_PLAYING,
		CurrentPick: model.Pick{Id: pickId},
		Picks:       picks,
		Players:     players,
	}, nil).Once()

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)

	draftActor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)

	skipped := SkipCurrentPick(t.Context(), draftActor, draftId, draftActor.GetDraftState().CurrentPick.Id)
	assert.True(t, skipped)

	// Give the actor a moment to process the state transition message
	// it posts internally after the skip.
	time.Sleep(100 * time.Millisecond)

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

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)
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

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)
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
	mockStore.On("DeletePick", mock.Anything, pickId).Return(nil).Once()
	mockStore.On("ResetPick", mock.Anything, 41, mock.Anything).Return(nil).Once()
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id: draftId,
		CurrentPick: model.Pick{Id: 41},
	}, nil).Once()

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)
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

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)
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

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)
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
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:     draftId,
		Status: model.FILLING,
	}, nil).Once()
	mockStore.On("UpdateDraftStatus", mock.Anything, draftId, model.PICKING).Return(nil).Once()
	mockStore.On("GetDraft", mock.Anything, draftId).Return(model.DraftModel{
		Id:     draftId,
		Status: model.PICKING,
	}, nil).Once()

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)
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
	actorMap := NewDraftActorMap(nil, nil, nil, nil, notifier)

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

func TestDraftActor_handleDeclineInvite_NotSupported(t *testing.T) {
	actor := &DraftActor{}

	result := actor.handleDeclineInvite(t.Context(), DeclineInviteMessage{})

	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "not yet supported")
}

func TestDraftActor_handleTransferDraftOwnership_NotSupported(t *testing.T) {
	actor := &DraftActor{}

	result := actor.handleTransferDraftOwnership(t.Context(), TransferDraftOwnershipMessage{})

	assert.Error(t, result.Error)
	assert.Contains(t, result.Error.Error(), "not yet supported")
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

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)
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

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)
	actor, err := actorMap.GetActor(t.Context(), draftId)
	assert.NoError(t, err)
	assert.NotNil(t, actor)

	// Shutdown the actor
	err = ShutdownActor(actorMap, t.Context(), draftId)
	assert.NoError(t, err)

	// Verify actor is removed from map
	_, ok := actorMap.actorMap.Load(draftId)
	assert.False(t, ok, "actor should be removed from map after shutdown")

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

	actorMap := NewDraftActorMap(mockStore, nil, nil, nil, nil)

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

	actor, err := NewDraftActor(t.Context(), draftId, mockStore, nil, nil, nil, nil)
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
				// This will fail early with "pick id does not match" (no DB call)
				msg = Message{
					Content: ModifyExpirationTimeMessage{PickId: 999, Extension: time.Minute},
					Reply:   replyChan,
				}
			case 1:
				// This returns an error without DB calls
				msg = Message{
					Content: DeclineInviteMessage{InviteId: idx},
					Reply:   replyChan,
				}
			case 2:
				// This also returns an error without DB calls
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
