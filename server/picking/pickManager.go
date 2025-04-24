package picking

import (
    "database/sql"
    "errors"
    "server/model"
    "server/tbaHandler"
    "sync"
)

type DraftPickManager struct {
    pickManagers map[int]*PickManager
}

func (d *DraftPickManager) GetPickManagerForDraft(draftId int) *PickManager {
    return d.pickManagers[draftId]
}

type PickManager struct {
    draftId int
    lock sync.Mutex
    listeners []*PickListener
    database *sql.DB
    tbaHandler *tbaHandler.TbaHandler
}

type PickEvent struct {
    success bool
    pick string
}

type PickListener interface {
    recievePickEvent(pickEvent PickEvent)
}

func NewPickManager(draftId int, database *sql.DB, tbaHandler *tbaHandler.TbaHandler) *PickManager {
    return &PickManager{
        draftId: draftId,
        database: database,
        tbaHandler: tbaHandler,
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
