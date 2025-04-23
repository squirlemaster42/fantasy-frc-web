package picking

import (
    "database/sql"
    "errors"
    "server/model"
    "server/tbaHandler"
    "sync"
)

type PickManager struct {
    draftId int
    lock sync.Mutex
    listeners []*PickListener
    database *sql.DB
    tbaHandler *tbaHandler.TbaHandler
}

type PickEvent struct {
    pick string
}

type PickListener interface {
    recievePickEvent(pickEvent PickEvent)
}

//TODO Do we want to wrap this in another layer so you
//cannot create an more than one pick manager per draft
func NewPickManager(draftId int, database *sql.DB, tbaHandler *tbaHandler.TbaHandler) *PickManager {
    return &PickManager{
        draftId: draftId,
        database: database,
    }
}

//Return error if pick is not able to be made
func (p *PickManager) makePick(pick string) error {
    p.lock.Lock()
    defer p.lock.Unlock()
    //validate pick
    if model.ValidPick(p.database, p.tbaHandler, pick, p.draftId) {
        for _, listener := range p.listeners {
            (*listener).recievePickEvent(PickEvent{
                pick: pick,
            })
        }
        return nil
    } else {
        return errors.New("Invalid Pick")
    }
}

func (p *PickManager) AddListener(listener PickListener) {
    p.listeners = append(p.listeners, &listener)
}

//TODO What should this take in?
func (p *PickManager) RemoveListener() { }
