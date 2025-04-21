package picking

import (
	"database/sql"
	"log/slog"

	"github.com/google/uuid"
)

//We need to store a set of connected clients
//and what draft they are looking at
//When a pick is made on a given draft we need
//to notify all of the clients watching
//that draft

//We are going to provide a block of html that will be the current state of the
//draft. We will need to disable the text input for anyone who is not making
//the current pick.

type PickNotifier struct {
    Database *sql.DB
    Watchers map[int][]Watcher //map the draft id to the group watching that draft
}

type Watcher struct {
    WatcherId uuid.UUID
    NotifierQueue chan bool
}

func (pn *PickNotifier) RegisterWatcher(draftId int) *Watcher {
    watcher := Watcher {
        WatcherId: uuid.New(),
        NotifierQueue: make(chan bool, 10), //This can probably be smaller?
    }

    pn.Watchers[draftId] = append(pn.Watchers[draftId], watcher)
    return &watcher
}

func (pn *PickNotifier) UnregiserWatcher(watcher *Watcher) {
    for key, watchers := range pn.Watchers {
        index := -1
        for i, w := range watchers {
            if w.WatcherId == watcher.WatcherId {
                index = i
            }
        }
        if index >= 0 {
            slog.Info("Unregistered watcher", "Index", index, "Key", key, "Watcher Id", watcher.WatcherId)
            pn.Watchers[key] = removeWatcher(watchers, index)
        } else {
            slog.Warn("Failed to unregister watcher", "Index", index, "Key", key, "Watcher Id", watcher.WatcherId)
        }
    }
}

func removeWatcher(w []Watcher, i int) []Watcher {
    w[i] = w[len(w) - 1]
    return w[:len(w) - 1]
}

func (pn *PickNotifier) NotifyWatchers(draftId int) {
    for _, watcher := range pn.Watchers[draftId] {
        watcher.NotifierQueue <- true
    }
}
