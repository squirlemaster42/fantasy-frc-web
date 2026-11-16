package picking

import (
	"context"
	"server/log"
	"server/metrics"
	"server/model"
	"sync"
	"time"

	"uuid"
)

type PickEvent struct {
	Pick    model.Pick
	DraftId int
}

// NotificationType identifies the kind of update a watcher should react to.
type NotificationType string

const (
	PickNotificationType              NotificationType = "pick"
	ScoreNotificationType             NotificationType = "score"
	DraftStateChangedNotificationType NotificationType = "draft_state_changed"
)

// Notification is the message sent to a watcher's queue. It replaces the
// previous untyped bool signal so the same notifier can fan out different
// kinds of draft updates (picks, scores, etc.).
type Notification struct {
	Type    NotificationType
	DraftId int
	Status  model.DraftState
}

// PickNotifier manages per-draft WebSocket watchers. Despite the name, it
// handles notifications for all live draft updates, including pick events and
// score events. Each connected client receives a typed Notification and the
// handler decides which HTML fragment to render and push.
type PickNotifier struct {
	mu       sync.RWMutex
	Watchers map[int][]Watcher // map the draft id to the group watching that draft
}

type Watcher struct {
	WatcherId     uuid.UUID
	NotifierQueue chan Notification
}

func (pn *PickNotifier) RegisterWatcher(draftId int) *Watcher {
	pn.mu.Lock()
	defer pn.mu.Unlock()

	watcher := Watcher{
		WatcherId:     uuid.New(),
		NotifierQueue: make(chan Notification, PickNotifierQueueBuffer()),
	}

	pn.Watchers[draftId] = append(pn.Watchers[draftId], watcher)
	metrics.IncrementWebSocketListener()
	return &watcher
}

func (pn *PickNotifier) UnregisterWatcher(ctx context.Context, watcher *Watcher) {
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
			log.Debug(ctx, "Unregistered watcher", "index", index, "draftId", key, "watcherId", watcher.WatcherId)
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
	pn.notifyWatchers(ctx, Notification{Type: PickNotificationType, DraftId: pickEvent.DraftId}, "pickId", pickEvent.Pick.Id)
	return nil
}

// ReceiveScoreEvent notifies all watchers for the given draft that scores have
// changed. The handler will re-render the draft score fragment and push it.
func (pn *PickNotifier) ReceiveScoreEvent(ctx context.Context, draftId int) {
	pn.notifyWatchers(ctx, Notification{Type: ScoreNotificationType, DraftId: draftId})
}

// NotifyStateChange notifies all watchers that the draft's state has changed.
// Handlers use this to close connections that are no longer valid for the
// current draft state (e.g. score notifiers when the draft leaves TEAMS_PLAYING).
func (pn *PickNotifier) NotifyStateChange(ctx context.Context, draftId int, status model.DraftState) {
	pn.notifyWatchers(ctx, Notification{Type: DraftStateChangedNotificationType, DraftId: draftId, Status: status}, "status", status)
}

func (pn *PickNotifier) notifyWatchers(ctx context.Context, notification Notification, logKeys ...any) {
	pn.mu.RLock()
	watchers := make([]Watcher, len(pn.Watchers[notification.DraftId]))
	copy(watchers, pn.Watchers[notification.DraftId])
	pn.mu.RUnlock()

	logArgs := append([]any{"type", notification.Type, "draftId", notification.DraftId, "numWatchers", len(watchers)}, logKeys...)
	log.Debug(ctx, "Notifying watchers", logArgs...)

	for _, watcher := range watchers {
		select {
		case watcher.NotifierQueue <- notification:
			// Sent successfully
		case <-time.After(PickNotifierSendTimeout()):
			log.Warn(ctx, "Timeout sending to watcher, skipping", "type", notification.Type, "watcherId", watcher.WatcherId)
			// Continue notifying remaining watchers; do not return error
		}
	}
}

// NotifyWatchers sends a pick notification to all watchers for the draft. It is
// primarily used by tests; production code should use ReceivePickEvent or
// ReceiveScoreEvent.
func (pn *PickNotifier) NotifyWatchers(ctx context.Context, draftId int) {
	pn.notifyWatchers(ctx, Notification{Type: PickNotificationType, DraftId: draftId})
}

// GetWatcherCount returns the number of active watchers for a draft. It is safe
// to call concurrently with RegisterWatcher and UnregisterWatcher.
func (pn *PickNotifier) GetWatcherCount(draftId int) int {
	pn.mu.RLock()
	defer pn.mu.RUnlock()
	return len(pn.Watchers[draftId])
}

// GetWatchers returns a snapshot of the watchers for a draft. It is safe to
// call concurrently with RegisterWatcher and UnregisterWatcher.
func (pn *PickNotifier) GetWatchers(draftId int) []Watcher {
	pn.mu.RLock()
	defer pn.mu.RUnlock()
	watchers := make([]Watcher, len(pn.Watchers[draftId]))
	copy(watchers, pn.Watchers[draftId])
	return watchers
}

// HasDraftWatchers reports whether any watchers are registered for a draft. It
// is safe to call concurrently with RegisterWatcher and UnregisterWatcher.
func (pn *PickNotifier) HasDraftWatchers(draftId int) bool {
	return pn.GetWatcherCount(draftId) > 0
}
