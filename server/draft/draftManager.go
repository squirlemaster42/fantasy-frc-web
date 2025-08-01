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
    err := draftDaemon.Start()
    if err != nil {
        slog.Warn("Failed to start draft daemon", "Error", err)
        return nil
    }

    slog.Info("Checking for drafts that need to be added to daemon")
    drafts := model.GetDraftsInStatus(database, model.PICKING)
    for _, draftId := range drafts {
        err = draftDaemon.AddDraft(draftId)
        if err != nil {
            slog.Warn("Failed to add draft to manager in init", "Error", err)
        }
    }

    draftManager := &DraftManager{
        drafts: map[int]Draft{},
        database: database,
        tbaHandler: tbaHandler,
        draftDaemon: draftDaemon,
        states: setupStates(database, draftDaemon),
    }

    slog.Info("Draft Manager Started")

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
    return tpt.draftDaemon.AddDraft(draft.draftId)
}

type ToPlayingTransition struct {
    database *sql.DB
    draftDaemon *background.DraftDaemon
}

func (tpt *ToPlayingTransition) executeTransition(draft *Draft) error {
    model.UpdateDraftStatus(tpt.database, draft.draftId, model.TEAMS_PLAYING)
    //Remove the draft from the pick daemon
    return tpt.draftDaemon.RemoveDraft(draft.draftId)
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
        transitions: make(map[model.DraftState]stateTransition),
    }
    states[model.FILLING].transitions[model.WAITING_TO_START] = &ToStartTransition {
        database: database,
    }

    states[model.WAITING_TO_START] = &state {
        state: model.WAITING_TO_START,
        transitions: make(map[model.DraftState]stateTransition),
    }
    states[model.WAITING_TO_START].transitions[model.PICKING] = &ToPickingTransition {
        database: database,
        draftDaemon: draftDaemon,
    }

    states[model.PICKING] = &state {
        state: model.PICKING,
        transitions: make(map[model.DraftState]stateTransition),
    }
    states[model.PICKING].transitions[model.TEAMS_PLAYING] = &ToPlayingTransition {
        database: database,
        draftDaemon: draftDaemon,
    }

    states[model.TEAMS_PLAYING] = &state {
        state: model.TEAMS_PLAYING,
        transitions: make(map[model.DraftState]stateTransition),
    }
    states[model.TEAMS_PLAYING].transitions[model.COMPLETE] = &ToCompleteTransition {
        database: database,
    }

    states[model.COMPLETE] = &state {
        state: model.COMPLETE,
        transitions: make(map[model.DraftState]stateTransition),
    }
    return states
}

type DraftManager struct {
    drafts map[int]Draft
    locks sync.Map
    database *sql.DB
    tbaHandler *tbaHandler.TbaHandler
    states map[model.DraftState]*state
    draftDaemon *background.DraftDaemon
}

func (dm *DraftManager) GetDraft(draftId int, reload bool) (Draft, error) {
    //We just need to be careful that only one call can reload the draft at one time
    lock := dm.getLock(draftId)
    draft, ok := dm.drafts[draftId]
    if ok && !reload {
        return draft, nil
    } else if reload {
        lock.Lock()
        defer lock.Unlock()
        draftModel, err := model.GetDraft(dm.database, draftId)
        draft.model = &draftModel
        return draft, err
    } else {
        //Load draft model
        lock.Lock()
        defer lock.Unlock()
        draftModel, err := model.GetDraft(dm.database, draftId)

        //TODO we should probably check if we need to do this in the reload path
        draft := Draft {
            draftId: draftId,
            pickManager: picking.NewPickManager(draftId, dm.database, dm.tbaHandler),
            model: &draftModel,
        }
        dm.drafts[draftId] = draft
        return draft, err
    }
}

type Draft struct {
    draftId int
    model *model.DraftModel
    pickManager *picking.PickManager
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

    lock := dm.getLock(draft.draftId)
    lock.Lock()
    defer lock.Unlock()

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

func (dm *DraftManager) MakePick(draftId int, pick model.Pick) error {
    lock := dm.getLock(draftId)
    lock.Lock()
    defer lock.Unlock()
    draft, err := dm.GetDraft(draftId, false)
    if err != nil {
        return err
    }

    pickingComplete, err := draft.pickManager.MakePick(pick)
    if pickingComplete {
        slog.Info("Update status to TEAMS_PLAYING", "Draft Id", draftId)
        err = dm.ExecuteDraftStateTransition(&draft, model.TEAMS_PLAYING)

        if err != nil {
            slog.Warn("Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
        }

        err = dm.draftDaemon.RemoveDraft(draftId)
        if err != nil {
            slog.Warn("Failed to remove draft from daemon", "Draft Id", draftId, "Error", err)
        }
    }
    return err
}

func (dm *DraftManager) getLock(draftId int) *sync.Mutex {
    //Get the lock if it exists for the draft, if not register it
    lock, ok := dm.locks.Load(draftId)
    if !ok {
        dm.locks.Store(draftId, &sync.Mutex{})
    }
    return lock.(*sync.Mutex)
}

func (dm *DraftManager) SkipCurrentPick(draftId int) {
    lock := dm.getLock(draftId)
    lock.Lock()
    defer lock.Unlock()
    dm.drafts[draftId].pickManager.SkipCurrentPick()
}

func (dm *DraftManager) AddPickListener(draftId int, listener picking.PickListener) {
    draft, err := dm.GetDraft(draftId, false)
    if err != nil {
        slog.Error("Failed to load draft when adding pick listener", "Error", err)
        return
    }
    draft.pickManager.AddListener(listener)
}

func (dm *DraftManager) UpdateDraft(draftModel model.DraftModel) error {
    lock := dm.getLock(draftModel.Id)
    lock.Lock()
    defer lock.Unlock()
    return model.UpdateDraft(dm.database, &draftModel)
}
