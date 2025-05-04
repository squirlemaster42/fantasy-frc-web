package draft

import (
	"database/sql"
	"fmt"
	"log/slog"
	"server/assert"
	"server/background"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"sync"
)

func NewDraftManager(tbaHandler *tbaHandler.TbaHandler, database *sql.DB) *DraftManager {
    //Start the draft daemon and add all running drafts to it
    draftDaemon := background.NewDraftDaemon(database)
    draftDaemon.Start()
    slog.Info("Checking for drafts that need to be added to daemon")
    drafts := model.GetDraftsInStatus(database, model.PICKING)
    for _, draftId := range drafts {
        draftDaemon.AddDraft(draftId)
    }

    draftManager := &DraftManager{
        drafts: map[int]*Draft{},
        database: database,
        tbaHandler: tbaHandler,
        draftDaemon: draftDaemon,
        states: setupStates(database, draftDaemon),
    }

    return draftManager
}

type stateTransition interface {
    executeTransition(draft *Draft) error
}

type ToStartTransition struct {
    database *sql.DB
}

func (tst *ToStartTransition) executeTransition(draft *Draft) error {
    model.UpdateDraftStatus(tst.database, draft.draftId, model.WAITING_TO_START)
    return nil
}

type ToPickingTransition struct {
    database *sql.DB
    draftDaemon *background.DraftDaemon
}

func (tpt *ToPickingTransition) executeTransition(draft *Draft) error {
    model.UpdateDraftStatus(tpt.database, draft.draftId, model.PICKING)
    //Add the draft to the pick daemon
    tpt.draftDaemon.AddDraft(draft.draftId)
    return nil
}

type ToPlayingTransition struct {
    database *sql.DB
    draftDaemon *background.DraftDaemon
}

func (tpt *ToPlayingTransition) executeTransition(draft *Draft) error {
    model.UpdateDraftStatus(tpt.database, draft.draftId, model.TEAMS_PLAYING)
    //Remove the draft from the pick daemon
    tpt.draftDaemon.RemoveDraft(draft.draftId)
    return nil
}

type ToCompleteTransition struct {
    database *sql.DB
}

func (tct *ToCompleteTransition) executeTransition(draft *Draft) error {
    model.UpdateDraftStatus(tct.database, draft.draftId, model.COMPLETE)
    return nil
}

type state struct {
    state model.DraftState
    transitions map[model.DraftState]stateTransition
}

func setupStates(database *sql.DB, draftDaemon *background.DraftDaemon) map[model.DraftState]*state {
    states := make(map[model.DraftState]*state)
    states[model.FILLING] = &state {
        state: model.FILLING,
    }
    states[model.FILLING].transitions[model.WAITING_TO_START] = &ToStartTransition{
        database: database,
    }

    states[model.WAITING_TO_START] = &state {
        state: model.WAITING_TO_START,
    }
    states[model.WAITING_TO_START].transitions[model.PICKING] = &ToPickingTransition {
        database: database,
        draftDaemon: draftDaemon,
    }

    states[model.PICKING] = &state {
        state: model.PICKING,
    }
    states[model.PICKING].transitions[model.TEAMS_PLAYING] = &ToPlayingTransition{
        database: database,
        draftDaemon: draftDaemon,
    }

    states[model.TEAMS_PLAYING] = &state {
        state: model.TEAMS_PLAYING,
    }
    states[model.TEAMS_PLAYING].transitions[model.COMPLETE] = &ToCompleteTransition{
        database: database,
    }

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
    draftDaemon *background.DraftDaemon
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

func (dm *DraftManager) ExecuteDraftStateTransition(draft *Draft, requestedState model.DraftState) error {
    assert := assert.CreateAssertWithContext("Execute Draft State Transition")
    assert.AddContext("Draft Id", draft.draftId)
    assert.AddContext("Current State", string(draft.model.Status))
    assert.AddContext("Requested State", string(requestedState))

    draft.stateLock.Lock()
    defer draft.stateLock.Unlock()

    state, stateFound := dm.states[draft.model.Status]
    assert.RunAssert(stateFound, "Current draft state is not registed in state machine")
    transition, transitionFound := state.transitions[requestedState]
    if !transitionFound {
        return &invalidStateTransitionError{
            currentState: model.FILLING,
            requestedState: model.FILLING,
        }
    }

    return transition.executeTransition(draft)
}

func (dm *DraftManager) MakePick(draft *Draft, pick model.Pick) error {
    return nil
}
