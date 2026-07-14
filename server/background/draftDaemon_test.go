package background

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"server/draft"
	"server/model"
	"server/model/mocks"
)

func TestNewDraftDaemon(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	actorMap := draft.NewDraftActorMap(mockStore, nil, nil, nil, nil)

	daemon := NewDraftDaemon(mockStore, actorMap)

	assert.NotNil(t, daemon)
	assert.False(t, daemon.IsRunning())
}

func TestDraftDaemon_AddDraft(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	actorMap := draft.NewDraftActorMap(mockStore, nil, nil, nil, nil)
	daemon := NewDraftDaemon(mockStore, actorMap)

	err := daemon.AddDraft(context.Background(), 1)
	assert.NoError(t, err)

	err = daemon.AddDraft(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "draft already added")
}

func TestDraftDaemon_RemoveDraft(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	actorMap := draft.NewDraftActorMap(mockStore, nil, nil, nil, nil)
	daemon := NewDraftDaemon(mockStore, actorMap)

	err := daemon.RemoveDraft(context.Background(), 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "draft not in daemon")

	err = daemon.AddDraft(context.Background(), 1)
	assert.NoError(t, err)

	err = daemon.RemoveDraft(context.Background(), 1)
	assert.NoError(t, err)
}

func TestDraftDaemon_StartStop(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	actorMap := draft.NewDraftActorMap(mockStore, nil, nil, nil, nil)
	daemon := NewDraftDaemon(mockStore, actorMap)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := daemon.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, daemon.IsRunning())

	err = daemon.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already started")

	err = daemon.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, daemon.IsRunning())

	err = daemon.Stop(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestDraftDaemon_StartStop_WithDraft(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	// The daemon will try to load the draft when checking for picks to skip.
	// Returning an error lets it log and continue without creating a real actor.
	mockStore.On("GetDraft", mock.Anything, 1).Return(model.DraftModel{}, assert.AnError).Maybe()

	actorMap := draft.NewDraftActorMap(mockStore, nil, nil, nil, nil)
	daemon := NewDraftDaemon(mockStore, actorMap)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := daemon.AddDraft(ctx, 1)
	assert.NoError(t, err)

	err = daemon.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, daemon.IsRunning())

	// Give the daemon a moment to enter its loop
	time.Sleep(50 * time.Millisecond)

	err = daemon.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, daemon.IsRunning())
}

func TestDraftDaemon_Run_RespectsStop(t *testing.T) {
	mockStore := mocks.NewMockDraftStore(t)
	actorMap := draft.NewDraftActorMap(mockStore, nil, nil, nil, nil)
	daemon := NewDraftDaemon(mockStore, actorMap)

	ctx := context.Background()

	err := daemon.Start(ctx)
	assert.NoError(t, err)

	err = daemon.Stop(ctx)
	assert.NoError(t, err)

	// Wait for the goroutine to observe the stop
	assert.Eventually(t, func() bool {
		return !daemon.IsRunning()
	}, time.Second, 10*time.Millisecond)
}
