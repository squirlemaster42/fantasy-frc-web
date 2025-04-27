package draft

import (
	"database/sql"
	"errors"
	"server/model"
	"server/picking"
	"sync"
)

type DraftManager struct {
    drafts map[int]*Draft
}

func (dm *DraftManager) GetDraft(draftId int, reload bool) *Draft {
    draft, ok := dm.drafts[draftId]
    if ok && !reload {
        return draft
    } else {
        //TODO Load draft, get pick manager, store in map, and return itA
        return nil
    }
}

type Draft struct {
    draftId int
    model *model.DraftModel
    pickManager *picking.PickManager
    stateLock sync.Mutex
}

func (d *Draft) ExecuteDraftStateTransition(requestedState model.DraftState, database *sql.DB) error {
    d.stateLock.Lock()
    defer d.stateLock.Unlock()

    switch requestedState {
    case model.FILLING:
        return nil
    case model.WAITING_TO_START:
        return nil
    case model.PICKING:
        return nil
    case model.TEAMS_PLAYING:
        return nil
    case model.COMPLETE:
        return nil
    default:
        return errors.New("Invalid requested draft state")
    }
}
