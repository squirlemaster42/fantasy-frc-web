package draft

import (
	"context"
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

func NewDraftManager(tbaHandler *tbaHandler.TbaHandler, draftStore model.DraftStore, teamStore model.TeamStore, discordStore model.DiscordStore, discordBus *discord.DiscordWebhookBus) *DraftManager {
	draftManager := &DraftManager{
		drafts:       map[int]*Draft{},
		draftStore:   draftStore,
		teamStore:    teamStore,
		discordStore: discordStore,
		tbaHandler:   tbaHandler,
		states:       setupStates(context.TODO(), draftStore),
		discordBus:   discordBus,
	}

	log.Info(context.TODO(), "Draft Manager Started")

	return draftManager
}

type DraftManager struct {
	drafts          map[int]*Draft
	loadLocks       sync.Map
	transitionLocks sync.Map
	draftStore      model.DraftStore
	teamStore       model.TeamStore
	discordStore    model.DiscordStore
	tbaHandler      *tbaHandler.TbaHandler
	states          map[model.DraftState]*state
	discordBus      *discord.DiscordWebhookBus
}

func (dm *DraftManager) GetDraft(draftId int, reload bool) (Draft, error) {
	log.Info(context.TODO(), "Get Draft", "Draft Id", draftId, "Reload", reload)
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
		draftModel, err := dm.draftStore.GetDraft(context.TODO(), draftId)
		log.DebugNoContext("GetDraft reload: model.GetDraft returned", "Requested Draft Id", draftId, "Returned", draftModel.Id, "Error", err)
		if draft != nil {
			draft.Model = &draftModel
		} else {
			draft := Draft{
				draftId:     draftId,
			pickManager: picking.NewPickManager(draftId, dm.draftStore, dm.teamStore, dm.discordStore, dm.tbaHandler, dm.discordBus),
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
		draftModel, err := dm.draftStore.GetDraft(context.TODO(), draftId)

		draft := Draft{
			draftId:     draftId,
			pickManager: picking.NewPickManager(draftId, dm.draftStore, dm.teamStore, dm.discordStore, dm.tbaHandler, dm.discordBus),
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

func (dm *DraftManager) ExecuteDraftStateTransition(ctx context.Context, draftId int, requestedState model.DraftState) error {
	log.Info(ctx, "Got request to execute draft state transition", "Draft Id", draftId, "Requested State", requestedState)
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
		log.Warn(ctx, "Failed get draft when trying to execute state transition", "Draft Id", draftId, "Error", err)
		loadLock.Unlock()
		return err
	}
	assert.AddContext("Draft Id", draft.draftId)
	assert.AddContext("Current State", string(draft.Model.Status))
	assert.AddContext("Requested State", string(requestedState))

	state, stateFound := dm.states[draft.Model.Status]
	assert.AddContext("Current Draft State", state)
	assert.RunAssert(ctx, stateFound, "Current draft state is not registed in state machine")
	log.Debug(ctx, "Found draft state", "Draft Id", draft.draftId, "State", state.state)
	transition, transitionFound := state.transitions[requestedState]
	if !transitionFound {
		log.Error(ctx, "Did not find draft state transition", "Current State", draft.Model.Status, "Requested State", requestedState)
		loadLock.Unlock()
		return &invalidStateTransitionError{
			currentState:   draft.Model.Status,
			requestedState: requestedState,
		}
	}

	log.Info(ctx, "Executing Draft State Transition", "Draft Id", draftId, "Requested State", requestedState)
	err = transition.executeTransition(ctx, model.DraftModel{})
	if err != nil {
		log.Warn(ctx, "Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		loadLock.Unlock()
		return err
	}
	log.Info(ctx, "Executed draft state transition", "Draft Id", draftId)

	loadLock.Unlock()
	draft, err = dm.GetDraft(draftId, true)
	log.Debug(ctx, "Reloaded draft after state transition", "End State", draft.GetStatus(), "Error", err)

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

	pickingComplete, err := draft.pickManager.MakePick(context.TODO(), pick)

	if err != nil {
		log.Info(context.TODO(), "Failed to make pick", "Pick", pick.Pick.String, "Pick Id", pick.Id, "Player", pick.Player, "Error", err)
		return err
	}

	if pickingComplete {
		log.Info(context.TODO(), "Update status to TEAMS_PLAYING", "Draft Id", draftId)
		err = dm.ExecuteDraftStateTransition(context.TODO(), draft.draftId, model.TEAMS_PLAYING)

		if err != nil {
			log.Warn(context.TODO(), "Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		}
	} else {
		// Reload the cached draft model so PickNotifier gets fresh data
		_, _ = dm.GetDraft(draftId, true)
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
		log.Warn(context.TODO(), "Skip Current Pick Error", "Draft Id", draftId, "Error", err)
		return err
	}

	err = draft.pickManager.SkipCurrentPick()
	if err != nil {
		return err
	}

	// Reload the cached draft model so PickNotifier gets fresh data
	_, _ = dm.GetDraft(draftId, true)
	return nil
}

func (dm *DraftManager) AddPickListener(draftId int, listener picking.PickListener) {
	draft, err := dm.GetDraft(draftId, false)
	if err != nil {
		log.Error(context.TODO(), "Failed to load draft when adding pick listener", "Error", err)
		return
	}
	draft.pickManager.AddListener(listener)
}

func (dm *DraftManager) RemovePickListener(draftId int, listener picking.PickListener) {
	draft, err := dm.GetDraft(draftId, false)
	if err != nil {
		log.Error(context.TODO(), "Failed to load draft when removing pick listener, searching all drafts", "Draft Id", draftId, "Error", err)
		for _, d := range dm.drafts {
			d.pickManager.RemoveListener(listener)
		}
		return
	}
	draft.pickManager.RemoveListener(listener)
}

// TODO decompose this into messages
func (dm *DraftManager) UpdateDraft(draftModel model.DraftModel) error {
	log.Info(context.TODO(), "UpdateDraft: acquiring locks", "Draft Id", draftModel.Id)
	loadLock := dm.getLoadLock(draftModel.Id)
	transitionLock := dm.getTransitionLock(draftModel.Id)
	loadLock.Lock()
	log.Info(context.TODO(), "UpdateDraft: acquired loadLock", "Draft Id", draftModel.Id)
	transitionLock.Lock()
	log.Info(context.TODO(), "UpdateDraft: acquired transitionLock", "Draft Id", draftModel.Id)
	defer transitionLock.Unlock()
	log.Info(context.TODO(), "UpdateDraft: calling model.UpdateDraft", "Draft Id", draftModel.Id)
	err := dm.draftStore.UpdateDraft(context.TODO(), &draftModel)
	log.Info(context.TODO(), "UpdateDraft: model.UpdateDraft returned", "Draft Id", draftModel.Id, "Error", err)
	loadLock.Unlock()
	if err != nil {
		log.Warn(context.TODO(), "Failed to update draft", "Error", err)
		return err
	}
	log.Info(context.TODO(), "UpdateDraft: calling GetDraft with reload", "Draft Id", draftModel.Id)
	_, err = dm.GetDraft(draftModel.Id, true)
	log.Info(context.TODO(), "UpdateDraft: GetDraft returned", "Draft Id", draftModel.Id, "Error", err)
	return err
}

func (dm *DraftManager) ModifyCurrentPickExpirationTime(draftId int, extention time.Duration) error {
	loadLock := dm.getLoadLock(draftId)
	transitionLock := dm.getTransitionLock(draftId)
	loadLock.Lock()
	transitionLock.Lock()
	defer transitionLock.Unlock()

	currentPick, err := dm.GetCurrentPick(draftId)
	if currentPick.Id == 0 || err != nil {
		loadLock.Unlock()
		return errors.New("no current pick found for this draft")
	}

	newExpirationTime := utils.GetPickExpirationTime(context.TODO(), currentPick.ExpirationTime, extention)
	log.Info(context.TODO(), "Setting new pick expiration time", "Current Pick Time", currentPick.ExpirationTime, "New Expiration Time", newExpirationTime, "Pick Id", currentPick.Id)

	err = dm.draftStore.UpdatePickExpirationTime(context.TODO(), currentPick.Id, newExpirationTime)
	if err != nil {
		log.Error(context.TODO(), "Failed to update pick expiration time", "Pick Id", currentPick.Id, "Error", err)
		loadLock.Unlock()
		return errors.New("failed to update pick expiration time")
	}

	loadLock.Unlock()
	_, err = dm.GetDraft(draftId, true)

	return err
}

func (dm *DraftManager) GetTbaHandler() *tbaHandler.TbaHandler {
	return dm.tbaHandler
}
