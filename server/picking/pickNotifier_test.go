package picking

import (
	"context"
	"testing"
	"time"

	"uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"server/model"
)

func TestPickNotifier_RegisterWatcher(t *testing.T) {
	pn := &PickNotifier{Watchers: make(map[int][]Watcher)}

	watcher := pn.RegisterWatcher(1)

	assert.NotNil(t, watcher)
	assert.NotEqual(t, uuid.Nil(), watcher.WatcherId)
	assert.NotNil(t, watcher.NotifierQueue)
	assert.Len(t, pn.Watchers[1], 1)
}

func TestPickNotifier_UnregisterWatcher(t *testing.T) {
	pn := &PickNotifier{Watchers: make(map[int][]Watcher)}

	watcher1 := pn.RegisterWatcher(1)
	watcher2 := pn.RegisterWatcher(1)
	_ = pn.RegisterWatcher(2)

	pn.UnregisterWatcher(context.Background(), watcher1)

	assert.Len(t, pn.Watchers[1], 1)
	assert.Equal(t, watcher2.WatcherId, pn.Watchers[1][0].WatcherId)
	assert.Len(t, pn.Watchers[2], 1)
}

func TestPickNotifier_UnregisterWatcher_RemovesEmptyDraftEntry(t *testing.T) {
	pn := &PickNotifier{Watchers: make(map[int][]Watcher)}

	watcher := pn.RegisterWatcher(1)
	pn.UnregisterWatcher(context.Background(), watcher)

	_, exists := pn.Watchers[1]
	assert.False(t, exists)
}

func TestPickNotifier_UnregisterWatcher_WatcherNotFound(t *testing.T) {
	pn := &PickNotifier{Watchers: make(map[int][]Watcher)}

	watcher := Watcher{
		NotifierQueue: make(chan bool, 1),
	}

	pn.UnregisterWatcher(context.Background(), &watcher)

	assert.Empty(t, pn.Watchers)
}

func TestPickNotifier_ReceivePickEvent(t *testing.T) {
	pn := &PickNotifier{Watchers: make(map[int][]Watcher)}

	watcher := pn.RegisterWatcher(1)

	err := pn.ReceivePickEvent(context.Background(), PickEvent{DraftId: 1, Pick: model.Pick{Id: 42}})

	require.NoError(t, err)
	select {
	case notified := <-watcher.NotifierQueue:
		assert.True(t, notified)
	case <-time.After(time.Second):
		t.Fatal("expected watcher to be notified")
	}
}

func TestPickNotifier_ReceivePickEvent_NoWatchers(t *testing.T) {
	pn := &PickNotifier{Watchers: make(map[int][]Watcher)}

	err := pn.ReceivePickEvent(context.Background(), PickEvent{DraftId: 1, Pick: model.Pick{Id: 42}})

	require.NoError(t, err)
}

func TestPickNotifier_ReceivePickEvent_SlowWatcherTimeout(t *testing.T) {
	pn := &PickNotifier{Watchers: make(map[int][]Watcher)}

	watcher := pn.RegisterWatcher(1)

	// Fill the channel so the notification blocks
	for range cap(watcher.NotifierQueue) {
		watcher.NotifierQueue <- true
	}

	done := make(chan struct{})
	var sendErr error
	go func() {
		defer close(done)
		sendErr = pn.ReceivePickEvent(context.Background(), PickEvent{DraftId: 1, Pick: model.Pick{Id: 42}})
	}()

	select {
	case <-done:
		require.NoError(t, sendErr)
	case <-time.After(10 * time.Second):
		t.Fatal("expected ReceivePickEvent to return after watcher timeout")
	}
}

func TestPickNotifier_NotifyWatchers(t *testing.T) {
	pn := &PickNotifier{Watchers: make(map[int][]Watcher)}

	watcher := pn.RegisterWatcher(1)

	pn.NotifyWatchers(context.Background(), 1)

	select {
	case notified := <-watcher.NotifierQueue:
		assert.True(t, notified)
	case <-time.After(time.Second):
		t.Fatal("expected watcher to be notified")
	}
}

func TestPickNotifier_NotifyWatchers_NoWatchers(t *testing.T) {
	pn := &PickNotifier{Watchers: make(map[int][]Watcher)}

	// Should not panic or block when no watchers are registered
	pn.NotifyWatchers(context.Background(), 1)
}
