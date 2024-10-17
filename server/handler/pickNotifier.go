package handler

import (
	"database/sql"

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
    database *sql.DB
    watchers map[int][]watcher //map the draft id to the group watching that draft
}

type watcher struct {
    watcherId uuid.UUID
    notifierQueue chan string
}

func (pn *PickNotifier) RegisterWatcher(draftId int) *watcher {
    watcher := watcher {
        watcherId: uuid.New(),
        notifierQueue: make(chan string, 10), //This can probably be smaller?
    }

    pn.watchers[draftId] = append(pn.watchers[draftId], watcher)
    return &watcher
}

func (pn *PickNotifier) UnregiserWatcher(watcher *watcher) {
    for key, watchers := range pn.watchers {
        index := -1
        for i, w := range watchers {
            if w.watcherId == watcher.watcherId {
                index = i
            }
        }
        pn.watchers[key] = removeWatcher(watchers, index)
    }
}

func removeWatcher(w []watcher, i int) []watcher {
    w[i] = w[len(w) - 1]
    return w[:len(w) - 1]
}

func (pn *PickNotifier) NotifyWatchers(draftId int, replacementText string) {
    for _, watcher := range pn.watchers[draftId] {
        watcher.notifierQueue <- replacementText
    }
}
