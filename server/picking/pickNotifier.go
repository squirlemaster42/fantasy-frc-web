package picking

import (
	"context"
	"server/log"
	"server/metrics"
	"server/model"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PickEvent struct {
	Success bool
	Err     error
	Pick    model.Pick
	DraftId int
}

type PickListener interface {
	ReceivePickEvent(ctx context.Context, pickEvent PickEvent) error
}

//We need to store a set of connected clients
//and what draft they are looking at
//When a pick is made on a given draft we need
//to notify all of the clients watching
//that draft

//We are going to provide a block of html that will be the current state of the
//draft. We will need to disable the text input for anyone who is not making
//the current pick.

type PickNotifier struct {
	mu       sync.RWMutex
	Watchers map[int][]Watcher //map the draft id to the group watching that draft
}

type Watcher struct {
	WatcherId     uuid.UUID
	NotifierQueue chan bool
}

func (pn *PickNotifier) RegisterWatcher(draftId int) *Watcher {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	watcher := Watcher{
		WatcherId:     uuid.New(),
		NotifierQueue: make(chan bool, 10),
	}

	pn.Watchers[draftId] = append(pn.Watchers[draftId], watcher)
	metrics.IncrementWebSocketListener()
	return &watcher
}

func (pn *PickNotifier) UnregisterWatcher(watcher *Watcher) {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	for key, watchers := range pn.Watchers {
		index := -1
		for i, w := range watchers {
			if w.WatcherId == watcher.WatcherId {
				index = i
				break
			}
		}
		if index >= 0 {
			log.Info(context.TODO(), "Unregistered watcher", "Index", index, "Key", key, "Watcher Id", watcher.WatcherId)
			pn.Watchers[key] = removeWatcher(watchers, index)
			metrics.DecrementWebSocketListener()
			// Clean up empty draft entries to prevent memory leaks
			if len(pn.Watchers[key]) == 0 {
				delete(pn.Watchers, key)
			}
		}
	}
}

func removeWatcher(w []Watcher, i int) []Watcher {
	w[i] = w[len(w)-1]
	return w[:len(w)-1]
}

func (pn *PickNotifier) ReceivePickEvent(ctx context.Context, pickEvent PickEvent) error {
	pn.mu.RLock()
	watchers := make([]Watcher, len(pn.Watchers[pickEvent.DraftId]))
	copy(watchers, pn.Watchers[pickEvent.DraftId])
	pn.mu.RUnlock()

	log.Info(context.TODO(), "Received Pick Event", "Pick Id", pickEvent.Pick.Id, "Draft Id", pickEvent.DraftId, "Num Watchers", len(watchers))
	for _, watcher := range watchers {
		select {
		case watcher.NotifierQueue <- true:
			// Sent successfully
		case <-time.After(5 * time.Second):
			log.Warn(ctx, "Timeout sending to watcher, skipping", "Watcher Id", watcher.WatcherId)
			// Continue notifying remaining watchers; do not return error
		}
	}
	return nil
}

func (pn *PickNotifier) NotifyWatchers(ctx context.Context, draftId int) {
	pn.mu.RLock()
	watchers := make([]Watcher, len(pn.Watchers[draftId]))
	copy(watchers, pn.Watchers[draftId])
	pn.mu.RUnlock()

	log.Info(ctx, "Notifying Watchers", "Draft Id", draftId, "Num Watchers", len(watchers))
	for _, watcher := range watchers {
		select {
		case watcher.NotifierQueue <- true:
		case <-time.After(5 * time.Second):
			log.Warn(context.TODO(), "Timeout notifying watcher", "Watcher Id", watcher.WatcherId)
		}
	}
}
