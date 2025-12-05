package draft

import (
	"database/sql"
	"fmt"
	"log/slog"
	"server/assert"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"sync"
)

func NewDraftManager(tbaHandler *tbaHandler.TbaHandler, database *sql.DB) *DraftManager {
    draftManager := &DraftManager{
        drafts: map[int]*Draft{},
        database: database,
        tbaHandler: tbaHandler,
        states: setupStates(database),
    }

	slog.Info("Draft Manager Started")

	return draftManager
}

type stateTransition interface {
	executeTransition(draft Draft) error
}

type ToStartTransition struct {
	database *sql.DB
}

func (tst *ToStartTransition) executeTransition(draft Draft) error {
	return model.UpdateDraftStatus(tst.database, draft.draftId, model.WAITING_TO_START)
}

type ToPickingTransition struct {
    database *sql.DB
}

func (tpt *ToPickingTransition) executeTransition(draft Draft) error {
    model.RandomizePickOrder(tpt.database, draft.draftId)
    model.UpdateDraftStatus(tpt.database, draft.draftId, model.PICKING)
    return nil
}

type ToPlayingTransition struct {
    database *sql.DB
}

func (tpt *ToPlayingTransition) executeTransition(draft Draft) error {
    model.UpdateDraftStatus(tpt.database, draft.draftId, model.TEAMS_PLAYING)
    //Remove the draft from the pick daemon
    return nil
}

type ToCompleteTransition struct {
	database *sql.DB
}

func (tct *ToCompleteTransition) executeTransition(draft Draft) error {
	return model.UpdateDraftStatus(tct.database, draft.draftId, model.COMPLETE)
}

type state struct {
	state       model.DraftState
	transitions map[model.DraftState]stateTransition
}

func setupStates(database *sql.DB) map[model.DraftState]*state {
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
    }

    states[model.PICKING] = &state {
        state: model.PICKING,
        transitions: make(map[model.DraftState]stateTransition),
    }
    states[model.PICKING].transitions[model.TEAMS_PLAYING] = &ToPlayingTransition {
        database: database,
    }

	states[model.TEAMS_PLAYING] = &state{
		state:       model.TEAMS_PLAYING,
		transitions: make(map[model.DraftState]stateTransition),
	}
	states[model.TEAMS_PLAYING].transitions[model.COMPLETE] = &ToCompleteTransition{
		database: database,
	}

	states[model.COMPLETE] = &state{
		state:       model.COMPLETE,
		transitions: make(map[model.DraftState]stateTransition),
	}
	return states
}

type DraftManager struct {
    //TODO this map should be thread safe
    drafts map[int]*Draft
    locks sync.Map
    database *sql.DB
    tbaHandler *tbaHandler.TbaHandler
    states map[model.DraftState]*state
}

// TODO we need to do something better with the locks here because there are some
// times where we are already locked from the parent and others where we need to aquire the lock here.
func (dm *DraftManager) GetDraft(draftId int, reload bool) (Draft, error) {
	slog.Info("Get Draft", "Draft Id", draftId, "Reload", reload)
	//We just need to be careful that only one call can reload the draft at one time
	lock := dm.getLock(draftId)
	draft, ok := dm.drafts[draftId]
	if ok && !reload {
		slog.Info("Returning cached draft", "Draft Id", draftId)
		return *draft, nil
	} else if reload {
		slog.Info("Reloading Draft", "Draft Id", draftId)
		lock.Lock()
		draftModel, err := model.GetDraft(dm.database, draftId)
		draft.model = &draftModel
		lock.Unlock()
		slog.Info("Reloaded Draft", "Draft Id", draftId)
		return *draft, err
	} else {
		slog.Info("Loading Draft For First Time", "Draft Id", draftId)
		//Load draft model
		lock.Lock()
		draftModel, err := model.GetDraft(dm.database, draftId)

		//TODO we should probably check if we need to do this in the reload path
		draft := Draft{
			draftId:     draftId,
			pickManager: picking.NewPickManager(draftId, dm.database, dm.tbaHandler),
			model:       &draftModel,
		}
		dm.drafts[draftId] = &draft
		lock.Unlock()
		slog.Info("Loaded Draft For First Time", "Draft Id", draftId)
		return draft, err
	}
}

type Draft struct {
	draftId     int
	model       *model.DraftModel
	pickManager *picking.PickManager
}

func (d *Draft) GetOwner() model.User {
	return d.model.Owner
}

func (d *Draft) GetStatus() model.DraftState {
	return d.model.Status
}

type invalidStateTransitionError struct {
	currentState   model.DraftState
	requestedState model.DraftState
}

func (e *invalidStateTransitionError) Error() string {
	return fmt.Sprintf("Invalid state tranition where current state was %s and requested state was %s", e.currentState, e.requestedState)
}

func (dm *DraftManager) ExecuteDraftStateTransition(draftId int, requestedState model.DraftState) error {
    slog.Info("Got request to execute draft state transition", "Draft Id", draftId, "Requested State", requestedState)
    assert := assert.CreateAssertWithContext("Execute Draft State Transition")
    draft, err := dm.GetDraft(draftId, false)
	if err != nil {
		slog.Warn("Failed get draft when trying to execute state transition", "Draft Id", draftId, "Error", err)
		return err
	}
    assert.AddContext("Draft Id", draft.draftId)
    assert.AddContext("Current State", string(draft.model.Status))
    assert.AddContext("Requested State", string(requestedState))

	lock := dm.getLock(draft.draftId)
	lock.Lock()

    state, stateFound := dm.states[draft.model.Status]
    assert.AddContext("Current Draft State", state)
    assert.RunAssert(stateFound, "Current draft state is not registed in state machine")
    transition, transitionFound := state.transitions[requestedState]
    if !transitionFound {
        slog.Error("Did not find draft state transition", "Current State", draft.model.Status, "Requested State", requestedState)
        return &invalidStateTransitionError{
            currentState: draft.model.Status,
            requestedState: requestedState,
        }
    }

    slog.Info("Executing Draft State Transition", "Draft Id", draftId, "Requested State", requestedState)
	err = transition.executeTransition(draft)
	if err != nil {
		slog.Warn("Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		return err
	}
	slog.Info("Executed draft state transition", "Draft Id", draftId)
	lock.Unlock()

	draft, err = dm.GetDraft(draftId, true)
	slog.Info("Reloaded draft after state transition", "End State", draft.GetStatus(), "Error", err)

	return err
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
		err = dm.ExecuteDraftStateTransition(draft.draftId, model.TEAMS_PLAYING)

        if err != nil {
            slog.Warn("Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
        }
    }
    return err
}

func (dm *DraftManager) getLock(draftId int) *sync.Mutex {
	//Get the lock if it exists for the draft, if not register it
	lock, ok := dm.locks.Load(draftId)
	if !ok {
		mtx := &sync.Mutex{}
		dm.locks.Store(draftId, mtx)
		return mtx
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
