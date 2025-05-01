package draft

import (
	"database/sql"
	"fmt"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"sync"
)

func NewDraftManager(tbaHandler *tbaHandler.TbaHandler, database *sql.DB) *DraftManager {
    return &DraftManager{
        drafts: map[int]*Draft{},
        database: database,
        tbaHandler: tbaHandler,
        states: setupStates(),
    }
}

type stateTransition interface {
    executeTransition(previousState model.DraftState, draft Draft)
}

type state struct {
    state model.DraftState
    transitions map[model.DraftState]stateTransition
}

func setupStates() map[model.DraftState]*state {
    states := make(map[model.DraftState]*state)
    states[model.FILLING] = &state {
        state: model.FILLING,
    }
    states[model.FILLING].transitions[model.WAITING_TO_START] = nil
    states[model.WAITING_TO_START] = &state {
        state: model.WAITING_TO_START,
    }
    states[model.WAITING_TO_START].transitions[model.PICKING] = nil
    states[model.PICKING] = &state {
        state: model.PICKING,
    }
    states[model.PICKING].transitions[model.TEAMS_PLAYING] = nil
    states[model.TEAMS_PLAYING] = &state {
        state: model.TEAMS_PLAYING,
    }
    states[model.TEAMS_PLAYING].transitions[model.COMPLETE] = nil
    states[model.COMPLETE] = &state {
        state: model.COMPLETE,
    }
    return states
}

type DraftManager struct {
    drafts map[int]*Draft
    database *sql.DB
    tbaHandler *tbaHandler.TbaHandler
    states map[model.DraftState]*state
}

func (dm *DraftManager) GetDraft(draftId int, reload bool) (*Draft, error) {
    draft, ok := dm.drafts[draftId]
    if ok && !reload {
        return draft, nil
    } else if reload {
        draftModel, err := model.GetDraft(dm.database, draftId)
        draft.model = &draftModel
        return draft, err
    } else {
        //Load draft model
        draftModel, err := model.GetDraft(dm.database, draftId)

        draft := Draft {
            draftId: draftId,
            pickManager: picking.NewPickManager(draftId, dm.database, dm.tbaHandler),
            model: &draftModel,
        }
        dm.drafts[draftId] = &draft
        return &draft, err
    }
}

type Draft struct {
    draftId int
    model *model.DraftModel
    pickManager *picking.PickManager
    stateLock sync.Mutex
}

type invalidStateTransitionError struct {
    currentState model.DraftState
    requestedState model.DraftState
}

func (e *invalidStateTransitionError) Error() string {
    return fmt.Sprintf("Invalid state tranition where current state was %s and requested state was %s", e.currentState, e.requestedState)
}

func (d *Draft) ExecuteDraftStateTransition(requestedState model.DraftState, database *sql.DB) error {
    d.stateLock.Lock()
    defer d.stateLock.Unlock()

    return &invalidStateTransitionError{
        currentState: model.FILLING,
        requestedState: model.FILLING,
    }
}
