package draft

import (
	"database/sql"
	"errors"
	"fmt"
	"server/assert"
	"server/discord"
	"server/log"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"server/utils"
	"sync"
	"time"
)

func NewDraftManager(tbaHandler *tbaHandler.TbaHandler, database *sql.DB, discordBus *discord.DiscordWebhookBus) *DraftManager {
	draftManager := &DraftManager{
		drafts:     map[int]*Draft{},
		database:   database,
		tbaHandler: tbaHandler,
		states:     setupStates(database),
		discordBus: discordBus,
	}

	log.InfoNoContext("Draft Manager Started")

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
	err := model.RandomizePickOrder(tpt.database, draft.draftId)
	if err != nil {
		return err
	}
	nextPickPlayer := model.NextPick(tpt.database, draft.draftId)
	model.MakePickAvailable(tpt.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now(), utils.PICK_TIME))
	err = model.UpdateDraftStatus(tpt.database, draft.draftId, model.PICKING)
	if err != nil {
		log.ErrorNoContext("Failed to update draft status", "Draft Id", draft.draftId, "Error", err)
		return err
	}
	return nil
}

type ToPlayingTransition struct {
	database *sql.DB
}

func (tpt *ToPlayingTransition) executeTransition(draft Draft) error {
	log.InfoNoContext("Executing TEAMS_PLAYING playing transition", "Draft Id", draft.draftId)
	err := model.UpdateDraftStatus(tpt.database, draft.draftId, model.TEAMS_PLAYING)
	if err != nil {
		log.ErrorNoContext("Failed to update draft status", "Draft Id", draft.draftId, "Error", err)
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
	drafts          map[int]*Draft
	loadLocks       sync.Map
	transitionLocks sync.Map
	database        *sql.DB
	tbaHandler      *tbaHandler.TbaHandler
	states          map[model.DraftState]*state
	discordBus      *discord.DiscordWebhookBus
}

func (dm *DraftManager) GetDraft(draftId int, reload bool) (Draft, error) {
	log.InfoNoContext("Get Draft", "Draft Id", draftId, "Reload", reload)
	//We just need to be careful that only one call can reload the draft at one time
	lock := dm.getLoadLock(draftId)
	draft, ok := dm.drafts[draftId]
	if ok && !reload {
		log.DebugNoContext("Returning cached draft", "Draft Id", draftId)
		return *draft, nil
	} else if reload {
		log.DebugNoContext("Reloading Draft", "Draft Id", draftId)
		lock.Lock()
		log.DebugNoContext("GetDraft reload: acquired loadLock", "Draft Id", draftId)
		draftModel, err := model.GetDraft(dm.database, draftId)
		log.DebugNoContext("GetDraft reload: model.GetDraft returned", "Requested Draft Id", draftId, "Returned", draftModel.Id, "Error", err)
		if draft != nil {
			draft.Model = &draftModel
		} else {
			draft := Draft{
				draftId:     draftId,
				pickManager: picking.NewPickManager(draftId, dm.database, dm.tbaHandler, dm.discordBus),
				Model:       &draftModel,
			}
			dm.drafts[draftId] = &draft
		}
		lock.Unlock()
		log.DebugNoContext("Reloaded Draft", "Draft Id", draftId)
		return *draft, err
	} else {
		log.DebugNoContext("Loading Draft For First Time", "Draft Id", draftId)
		//Load draft model
		lock.Lock()
		draftModel, err := model.GetDraft(dm.database, draftId)

		draft := Draft{
			draftId:     draftId,
			pickManager: picking.NewPickManager(draftId, dm.database, dm.tbaHandler, dm.discordBus),
			Model:       &draftModel,
		}
		dm.drafts[draftId] = &draft
		lock.Unlock()
		log.DebugNoContext("Loaded Draft For First Time", "Draft Id", draftId)
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
	log.InfoNoContext("Got request to execute draft state transition", "Draft Id", draftId, "Requested State", requestedState)
	assert := assert.CreateAssertWithContext("Execute Draft State Transition")

	loadLock := dm.getLoadLock(draftId)
	transitionLock := dm.getTransitionLock(draftId)
	loadLock.Lock()
	log.DebugNoContext("ExecuteDraftStateTransition: acquired loadLock", "Draft Id", draftId)
	transitionLock.Lock()
	log.DebugNoContext("ExecuteDraftStateTransition: acquired transitionLock", "Draft Id", draftId)
	defer transitionLock.Unlock()

	draft, err := dm.GetDraft(draftId, false)
	log.DebugNoContext("Loaded draft to execute transition", "Draft Id", draftId)
	if err != nil {
		log.WarnNoContext("Failed get draft when trying to execute state transition", "Draft Id", draftId, "Error", err)
		loadLock.Unlock()
		return err
	}
	assert.AddContext("Draft Id", draft.draftId)
	assert.AddContext("Current State", string(draft.Model.Status))
	assert.AddContext("Requested State", string(requestedState))

	state, stateFound := dm.states[draft.Model.Status]
	assert.AddContext("Current Draft State", state)
	assert.RunAssert(stateFound, "Current draft state is not registed in state machine")
	log.DebugNoContext("Found draft state", "Draft Id", draft.draftId, "State", state.state)
	transition, transitionFound := state.transitions[requestedState]
	if !transitionFound {
		log.ErrorNoContext("Did not find draft state transition", "Current State", draft.Model.Status, "Requested State", requestedState)
		loadLock.Unlock()
		return &invalidStateTransitionError{
			currentState:   draft.Model.Status,
			requestedState: requestedState,
		}
	}

	log.InfoNoContext("Executing Draft State Transition", "Draft Id", draftId, "Requested State", requestedState)
	err = transition.executeTransition(draft)
	if err != nil {
		log.WarnNoContext("Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		loadLock.Unlock()
		return err
	}
	log.InfoNoContext("Executed draft state transition", "Draft Id", draftId)

	loadLock.Unlock()
	draft, err = dm.GetDraft(draftId, true)
	log.DebugNoContext("Reloaded draft after state transition", "End State", draft.GetStatus(), "Error", err)

	return err
}

func (dm *DraftManager) UndoLastPick(draftId int) error {
	draft, err := dm.GetDraft(draftId, false)
	if err != nil {
		return err
	}
	return draft.pickManager.UndoLastPick()
}

func (dm *DraftManager) GetCurrentPick(draftId int) (model.Pick, error) {
	draft, err := dm.GetDraft(draftId, false)
	if err != nil {
		return model.Pick{}, err
	}

	return draft.pickManager.GetCurrentPick(draftId)
}

func (dm *DraftManager) MakePick(draftId int, pick model.Pick) error {
	draft, err := dm.GetDraft(draftId, false)
	if err != nil {
		return err
	}

	pickingComplete, err := draft.pickManager.MakePick(pick)

	if err != nil {
		log.InfoNoContext("Failed to make pick", "Pick", pick.Pick.String, "Pick Id", pick.Id, "Player", pick.Player, "Error", err)
		return err
	}

	if pickingComplete {
		log.InfoNoContext("Update status to TEAMS_PLAYING", "Draft Id", draftId)
		err = dm.ExecuteDraftStateTransition(draft.draftId, model.TEAMS_PLAYING)

		if err != nil {
			log.WarnNoContext("Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		}
	}

	go draft.pickManager.NotifyListeners(picking.PickEvent{
		Pick:    pick,
		Success: err == nil,
		Err:     err,
		DraftId: draftId,
	})

	return err
}

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
	draft, err := dm.GetDraft(draftId, false)

	if err != nil {
		log.WarnNoContext("Skip Current Pick Error", "Draft Id", draftId, "Error", err)
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
		log.ErrorNoContext("Failed to load draft when adding pick listener", "Error", err)
		return
	}
	draft.pickManager.AddListener(listener)
}

func (dm *DraftManager) UpdateDraft(draftModel model.DraftModel) error {
	log.InfoNoContext("UpdateDraft: acquiring locks", "Draft Id", draftModel.Id)
	loadLock := dm.getLoadLock(draftModel.Id)
	transitionLock := dm.getTransitionLock(draftModel.Id)
	loadLock.Lock()
	log.InfoNoContext("UpdateDraft: acquired loadLock", "Draft Id", draftModel.Id)
	transitionLock.Lock()
	log.InfoNoContext("UpdateDraft: acquired transitionLock", "Draft Id", draftModel.Id)
	defer transitionLock.Unlock()
	log.InfoNoContext("UpdateDraft: calling model.UpdateDraft", "Draft Id", draftModel.Id)
	err := model.UpdateDraft(dm.database, &draftModel)
	log.InfoNoContext("UpdateDraft: model.UpdateDraft returned", "Draft Id", draftModel.Id, "Error", err)
	loadLock.Unlock()
	if err != nil {
		log.WarnNoContext("Failed to update draft", "Error", err)
		return err
	}
	log.InfoNoContext("UpdateDraft: calling GetDraft with reload", "Draft Id", draftModel.Id)
	_, err = dm.GetDraft(draftModel.Id, true)
	log.InfoNoContext("UpdateDraft: GetDraft returned", "Draft Id", draftModel.Id, "Error", err)
	return err
}

func (dm *DraftManager) ModifyCurrentPickExpirationTime(draftId int, extention time.Duration) error {
	loadLock := dm.getLoadLock(draftId)
	transitionLock := dm.getTransitionLock(draftId)
	loadLock.Lock()
	transitionLock.Lock()
	defer loadLock.Unlock()
	defer transitionLock.Unlock()

	currentPick, err := dm.GetCurrentPick(draftId)
	if currentPick.Id == 0 || err != nil {
		return errors.New("no current pick found for this draft")
	}

	newExpirationTime := utils.GetPickExpirationTime(currentPick.ExpirationTime, extention)
	log.InfoNoContext("Setting new pick expiration time", "Expiration Time", newExpirationTime, "Pick Id", currentPick.Id)

	err = model.UpdatePickExpirationTime(dm.database, currentPick.Id, newExpirationTime)
	if err != nil {
		log.ErrorNoContext("Failed to update pick expiration time", "Pick Id", currentPick.Id, "Error", err)
		return errors.New("failed to update pick expiration time")
	}

	return nil
}

func (dm *DraftManager) GetTbaHandler() *tbaHandler.TbaHandler {
	return dm.tbaHandler
}
