package handler

import (
	"server/model"

	"github.com/gorilla/websocket"
)

//We need to store a set of connected clients
//and what draft they are looking at
//When a pick is made on a given draft we need
//to notify all of the clients watching
//that draft

//We are going to provide a block of html that will be the current state of the
//draft. We will need to disable the text input for anyone who is not making
//the current pick.

type pickNotifier struct {
    watchers map[int][]watcher //map the draft id to the group watching that draft
}

type watcher struct {
    ws *websocket.Conn
}

func (pn *pickNotifier) registerWatcher(draft model.Draft, ws *websocket.Conn) {
    watcher := watcher {
        ws: ws,
    }

    //TODO We need to make sure that we dont need to do some initialization here
    pn.watchers[draft.Id] = append(pn.watchers[draft.Id], watcher)
}

func (pn *pickNotifier) unregiserWatcher(watcher watcher) {
    //TODO need to remove from watchers
    watcher.ws.Close();
}

func (pn *pickNotifier) notifyWatchers(draft model.Draft) {
    for _, watcher := range pn.watchers[draft.Id] {
        watcher.ws.WriteMessage(websocket.TextMessage, []byte("Test"))
    }
}
