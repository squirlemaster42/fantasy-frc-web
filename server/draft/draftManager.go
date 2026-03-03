package draft

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"server/assert"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"server/utils"
	"sync"
	"time"
)

func NewDraftManager(tbaHandler *tbaHandler.TbaHandler, database *sql.DB) *DraftManager {
	draftManager := &DraftManager{
		drafts:     map[int]*Draft{},
		database:   database,
		tbaHandler: tbaHandler,
		states:     setupStates(database),
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
	nextPickPlayer := model.NextPick(tpt.database, draft.draftId)
	model.MakePickAvailable(tpt.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now()))
	err := model.UpdateDraftStatus(tpt.database, draft.draftId, model.PICKING)
	if err != nil {
		slog.Error("Failed to update draft status", "Draft Id", draft.draftId, "Error", err)
		return err
	}
	return nil
}

type ToPlayingTransition struct {
	database *sql.DB
}

func (tpt *ToPlayingTransition) executeTransition(draft Draft) error {
	slog.Info("Executing TEAMS_PLAYING playing transition", "Draft Id", draft.draftId)
	err := model.UpdateDraftStatus(tpt.database, draft.draftId, model.TEAMS_PLAYING)
	if err != nil {
		slog.Error("Failed to update draft status", "Draft Id", draft.draftId, "Error", err)
	}
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
	states[model.FILLING] = &state{
		state:       model.FILLING,
		transitions: make(map[model.DraftState]stateTransition),
	}
	states[model.FILLING].transitions[model.WAITING_TO_START] = &ToStartTransition{
		database: database,
	}

	states[model.WAITING_TO_START] = &state{
		state:       model.WAITING_TO_START,
		transitions: make(map[model.DraftState]stateTransition),
	}
	states[model.WAITING_TO_START].transitions[model.PICKING] = &ToPickingTransition{
		database: database,
	}

	states[model.PICKING] = &state{
		state:       model.PICKING,
		transitions: make(map[model.DraftState]stateTransition),
	}
	states[model.PICKING].transitions[model.TEAMS_PLAYING] = &ToPlayingTransition{
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
	drafts          map[int]*Draft
	loadLocks       sync.Map
	transitionLocks sync.Map
	pickLocks       sync.Map
	database        *sql.DB
	tbaHandler      *tbaHandler.TbaHandler
	states          map[model.DraftState]*state
}

func (dm *DraftManager) GetDraft(draftId int, reload bool) (Draft, error) {
	slog.Info("Get Draft", "Draft Id", draftId, "Reload", reload)
	//We just need to be careful that only one call can reload the draft at one time
	lock := dm.getLoadLock(draftId)
	draft, ok := dm.drafts[draftId]
	if ok && !reload {
		slog.Debug("Returning cached draft", "Draft Id", draftId)
		return *draft, nil
	} else if reload {
		slog.Debug("Reloading Draft", "Draft Id", draftId)
		lock.Lock()
		draftModel, err := model.GetDraft(dm.database, draftId)
		draft.Model = &draftModel
		lock.Unlock()
		slog.Debug("Reloaded Draft", "Draft Id", draftId)
		return *draft, err
	} else {
		slog.Debug("Loading Draft For First Time", "Draft Id", draftId)
		//Load draft model
		lock.Lock()
		draftModel, err := model.GetDraft(dm.database, draftId)

		//TODO we should probably check if we need to do this in the reload path
		draft := Draft{
			draftId:     draftId,
			pickManager: picking.NewPickManager(draftId, dm.database, dm.tbaHandler),
			Model:       &draftModel,
		}
		dm.drafts[draftId] = &draft
		lock.Unlock()
		slog.Debug("Loaded Draft For First Time", "Draft Id", draftId)
		return draft, err
	}
}

type Draft struct {
	draftId     int
	Model       *model.DraftModel
	pickManager *picking.PickManager
}

func (d *Draft) GetOwner() model.User {
	return d.Model.Owner
}

func (d *Draft) GetStatus() model.DraftState {
	return d.Model.Status
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

	lock := dm.getTransitionLock(draftId)
	lock.Lock()
	defer lock.Unlock()
	slog.Debug("Aquired transition lock", "Draft Id", draftId)

	draft, err := dm.GetDraft(draftId, false)
	slog.Debug("Loaded draft to execute transition", "Draft Id", draftId)
	if err != nil {
		slog.Warn("Failed get draft when trying to execute state transition", "Draft Id", draftId, "Error", err)
		return err
	}
	assert.AddContext("Draft Id", draft.draftId)
	assert.AddContext("Current State", string(draft.Model.Status))
	assert.AddContext("Requested State", string(requestedState))

	state, stateFound := dm.states[draft.Model.Status]
	assert.AddContext("Current Draft State", state)
	assert.RunAssert(stateFound, "Current draft state is not registed in state machine")
	slog.Debug("Found draft state", "Draft Id", draft.draftId, "State", state.state)
	transition, transitionFound := state.transitions[requestedState]
	if !transitionFound {
		slog.Error("Did not find draft state transition", "Current State", draft.Model.Status, "Requested State", requestedState)
		return &invalidStateTransitionError{
			currentState:   draft.Model.Status,
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

	draft, err = dm.GetDraft(draftId, true)
	slog.Debug("Reloaded draft after state transition", "End State", draft.GetStatus(), "Error", err)

	return err
}

func (dm *DraftManager) UndoLastPick(draftId int) error {
	pickLock := dm.getPickLock(draftId)
	pickLock.Lock()
	defer pickLock.Unlock()

	// Get the current pick
	currentPick, err := model.GetCurrentPick(dm.database, draftId)
	if currentPick.Id == 0 || err != nil {
		return errors.New("no current pick found for this draft")
	}

	// Get the previous pick
	previousPick, err := model.GetPreviousPick(dm.database, draftId, currentPick.Id)
	if err != nil {
		return errors.New("cannot undo pick: this is the first pick of the draft")
	}

	// Delete the current pick
	err = model.DeletePick(dm.database, currentPick.Id)
	if err != nil {
		slog.Error("Failed to delete current pick", "Pick Id", currentPick.Id, "Error", err)
		return errors.New("failed to delete current pick")
	}

	// Set the expiration time to 3 hours from now
	newExpirationTime := time.Now().Add(3 * time.Hour)

	// Reset the previous pick (null out pick and pickTime, and set new expiration)
	err = model.ResetPick(dm.database, previousPick.Id, newExpirationTime)
	if err != nil {
		slog.Error("Failed to reset previous pick", "Pick Id", previousPick.Id, "Error", err)
		return errors.New("failed to reset previous pick")
	}
	return nil
}

// TODO Make sure that all GetCurrentPick calls are going through this
func (dm *DraftManager) GetCurrentPick(draftId int) (model.Pick, error) {
	pickLock := dm.getPickLock(draftId)
	pickLock.Lock()
	defer pickLock.Unlock()
	return model.GetCurrentPick(dm.database, draftId)
}

func (dm *DraftManager) MakePick(draftId int, pick model.Pick) error {
	draft, err := dm.GetDraft(draftId, false)
	if err != nil {
		return err
	}

	pickLock := dm.getPickLock(draftId)
	pickLock.Lock()
	defer pickLock.Unlock()
	pickingComplete, err := draft.pickManager.MakePick(pick)
	if pickingComplete {
		slog.Info("Update status to TEAMS_PLAYING", "Draft Id", draftId)
		// TODO This transition does not execute because we have the lock above
		// I should probably just make a pick lock
		err = dm.ExecuteDraftStateTransition(draft.draftId, model.TEAMS_PLAYING)

		if err != nil {
			slog.Warn("Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		}
	}
	return err
}

// TODO Can we do something nicer with these two lock functions?
func (dm *DraftManager) getLoadLock(draftId int) *sync.Mutex {
	//Get the lock if it exists for the draft, if not register it
	lock, ok := dm.loadLocks.Load(draftId)
	if !ok {
		mtx := &sync.Mutex{}
		dm.loadLocks.Store(draftId, mtx)
		return mtx
	}
	return lock.(*sync.Mutex)
}

func (dm *DraftManager) getPickLock(draftId int) *sync.Mutex {
	//Get the lock if it exists for the draft, if not register it
	lock, ok := dm.pickLocks.Load(draftId)
	if !ok {
		mtx := &sync.Mutex{}
		dm.pickLocks.Store(draftId, mtx)
		return mtx
	}
	return lock.(*sync.Mutex)
}

func (dm *DraftManager) getTransitionLock(draftId int) *sync.Mutex {
	//Get the lock if it exists for the draft, if not register it
	lock, ok := dm.transitionLocks.Load(draftId)
	if !ok {
		mtx := &sync.Mutex{}
		dm.transitionLocks.Store(draftId, mtx)
		return mtx
	}
	return lock.(*sync.Mutex)
}

func (dm *DraftManager) SkipCurrentPick(draftId int) error {
	lock := dm.getTransitionLock(draftId)
	lock.Lock()
	defer lock.Unlock()
	draft, err := dm.GetDraft(draftId, false)

	if err != nil {
		slog.Warn("Skip Current Pick Error", "Draft Id", draftId, "Error", err)
		return err
	}

	err = draft.pickManager.SkipCurrentPick()
	if err != nil {
		return err
	}
	return nil
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
	loadLock := dm.getLoadLock(draftModel.Id)
	transitionLock := dm.getTransitionLock(draftModel.Id)
	loadLock.Lock()
	transitionLock.Lock()
	defer loadLock.Unlock()
	defer transitionLock.Unlock()
	err := model.UpdateDraft(dm.database, &draftModel)
	if err != nil {
		slog.Warn("Failed to update draft", "Error", err)
		return err
	}
	_, err = dm.GetDraft(draftModel.Id, true)
	return err
}

func (dm *DraftManager) ModifyCurrentPickExpirationTime(draftId int, expirationTime time.Duration) error {
	loadLock := dm.getLoadLock(draftId)
	transitionLock := dm.getTransitionLock(draftId)
	loadLock.Lock()
	transitionLock.Lock()
	defer loadLock.Unlock()
	defer transitionLock.Unlock()

	currentPick, err := model.GetCurrentPick(dm.database, draftId)
	if currentPick.Id == 0 || err != nil {
		return errors.New("no current pick found for this draft")
	}

	newExpirationTime := time.Now().Add(expirationTime)

	err = model.UpdatePickExpirationTime(dm.database, currentPick.Id, newExpirationTime)
	if err != nil {
		slog.Error("Failed to update pick expiration time", "Pick Id", currentPick.Id, "Error", err)
		return errors.New("failed to update pick expiration time")
	}

	return nil
}
