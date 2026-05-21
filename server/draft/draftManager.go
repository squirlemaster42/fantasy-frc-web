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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var draftManagerTracer = otel.Tracer("draft-manager")

func NewDraftManager(ctx context.Context, tbaHandler *tbaHandler.TbaHandler, draftStore model.DraftStore, teamStore model.TeamStore, discordStore model.DiscordStore, discordBus *discord.DiscordWebhookBus) *DraftManager {
	draftManager := &DraftManager{
		drafts:        map[int]*Draft{},
		draftContexts: map[int]context.CancelFunc{},
		draftStore:    draftStore,
		teamStore:     teamStore,
		discordStore:  discordStore,
		tbaHandler:    tbaHandler,
		states:        setupStates(ctx, draftStore),
		discordBus:    discordBus,
		serverCtx:     ctx,
	}

	log.Info(ctx, "Draft Manager Started")

	return draftManager
}

type DraftManager struct {
	drafts          map[int]*Draft
	draftContexts   map[int]context.CancelFunc
	loadLocks       sync.Map
	transitionLocks sync.Map
	draftStore      model.DraftStore
	teamStore       model.TeamStore
	discordStore    model.DiscordStore
	tbaHandler      *tbaHandler.TbaHandler
	states          map[model.DraftState]*state
	discordBus      *discord.DiscordWebhookBus
	serverCtx       context.Context
}

func (dm *DraftManager) GetDraft(ctx context.Context, draftId int, reload bool) (Draft, error) {
	log.Info(ctx, "Get Draft", "Draft Id", draftId, "Reload", reload)
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
		draftModel, err := dm.draftStore.GetDraft(ctx, draftId)
		log.DebugNoContext("GetDraft reload: model.GetDraft returned", "Requested Draft Id", draftId, "Returned", draftModel.Id, "Error", err)
		if draft != nil {
			draft.Model = &draftModel
		} else {
			draftCtx, cancel := context.WithCancel(dm.serverCtx)
			dm.draftContexts[draftId] = cancel
			draft := Draft{
				draftId:     draftId,
				ctx:         draftCtx,
				pickManager: picking.NewPickManager(draftId, dm.draftStore, dm.teamStore, dm.discordStore, dm.tbaHandler, dm.discordBus, draftCtx),
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
		draftModel, err := dm.draftStore.GetDraft(ctx, draftId)

		draftCtx, cancel := context.WithCancel(dm.serverCtx)
		dm.draftContexts[draftId] = cancel
		draft := Draft{
			draftId:     draftId,
			ctx:         draftCtx,
			pickManager: picking.NewPickManager(draftId, dm.draftStore, dm.teamStore, dm.discordStore, dm.tbaHandler, dm.discordBus, draftCtx),
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
	ctx         context.Context
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
	ctx, span := draftManagerTracer.Start(ctx, "draft.state-transition",
		trace.WithAttributes(
			attribute.Int("draft.id", draftId),
			attribute.String("draft.requested_state", string(requestedState)),
		),
	)
	defer span.End()

	log.Info(ctx, "Got request to execute draft state transition", "Draft Id", draftId, "Requested State", requestedState)
	assert := assert.CreateAssertWithContext("Execute Draft State Transition")

	loadLock := dm.getLoadLock(draftId)
	transitionLock := dm.getTransitionLock(draftId)
	loadLock.Lock()
	log.DebugNoContext("ExecuteDraftStateTransition: acquired loadLock", "Draft Id", draftId)
	transitionLock.Lock()
	log.DebugNoContext("ExecuteDraftStateTransition: acquired transitionLock", "Draft Id", draftId)
	defer transitionLock.Unlock()

	draft, err := dm.GetDraft(ctx, draftId, false)
	log.DebugNoContext("Loaded draft to execute transition", "Draft Id", draftId)
	if err != nil {
		log.Warn(ctx, "Failed get draft when trying to execute state transition", "Draft Id", draftId, "Error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		loadLock.Unlock()
		return err
	}
	span.SetAttributes(attribute.String("draft.current_state", string(draft.Model.Status)))
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
		err := &invalidStateTransitionError{
			currentState:   draft.Model.Status,
			requestedState: requestedState,
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		loadLock.Unlock()
		return err
	}

	log.Info(ctx, "Executing Draft State Transition", "Draft Id", draftId, "Requested State", requestedState)
	err = transition.executeTransition(ctx, draft)
	if err != nil {
		log.Warn(ctx, "Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		loadLock.Unlock()
		return err
	}
	log.Info(ctx, "Executed draft state transition", "Draft Id", draftId)

	loadLock.Unlock()
	draft, err = dm.GetDraft(ctx, draftId, true)
	log.Debug(ctx, "Reloaded draft after state transition", "End State", draft.GetStatus(), "Error", err)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (dm *DraftManager) UndoLastPick(draftId int) error {
	draft, err := dm.GetDraft(dm.serverCtx, draftId, false)
	if err != nil {
		return err
	}
	return draft.pickManager.UndoLastPick()
}

func (dm *DraftManager) GetCurrentPick(draftId int) (model.Pick, error) {
	draft, err := dm.GetDraft(dm.serverCtx, draftId, false)
	if err != nil {
		return model.Pick{}, err
	}

	return draft.pickManager.GetCurrentPick(draftId)
}

func (dm *DraftManager) MakePick(draftId int, pick model.Pick) error {
	draft, err := dm.GetDraft(dm.serverCtx, draftId, false)
	if err != nil {
		return err
	}

	ctx, span := draftManagerTracer.Start(draft.ctx, "draft.make-pick",
		trace.WithAttributes(
			attribute.Int("draft.id", draftId),
			attribute.Int("pick.id", pick.Id),
			attribute.String("pick.team", pick.Pick.String),
		),
	)
	defer span.End()

	pickingComplete, err := draft.pickManager.MakePick(ctx, pick)

	if err != nil {
		log.Info(ctx, "Failed to make pick", "Pick", pick.Pick.String, "Pick Id", pick.Id, "Player", pick.Player, "Error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if pickingComplete {
		log.Info(ctx, "Update status to TEAMS_PLAYING", "Draft Id", draftId)
		err = dm.ExecuteDraftStateTransition(ctx, draft.draftId, model.TEAMS_PLAYING)

		if err != nil {
			log.Warn(ctx, "Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	} else {
		// Reload the cached draft model so PickNotifier gets fresh data
		_, _ = dm.GetDraft(dm.serverCtx, draftId, true)
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
	draft, err := dm.GetDraft(dm.serverCtx, draftId, false)

	if err != nil {
		log.Warn(draft.ctx, "Skip Current Pick Error", "Draft Id", draftId, "Error", err)
		return err
	}

	err = draft.pickManager.SkipCurrentPick()
	if err != nil {
		return err
	}

	// Reload the cached draft model so PickNotifier gets fresh data
	_, _ = dm.GetDraft(dm.serverCtx, draftId, true)
	return nil
}

func (dm *DraftManager) AddPickListener(draftId int, listener picking.PickListener) {
	draft, err := dm.GetDraft(dm.serverCtx, draftId, false)
	if err != nil {
		log.Error(dm.serverCtx, "Failed to load draft when adding pick listener", "Error", err)
		return
	}
	draft.pickManager.AddListener(listener)
}

func (dm *DraftManager) RemovePickListener(draftId int, listener picking.PickListener) {
	draft, err := dm.GetDraft(dm.serverCtx, draftId, false)
	if err != nil {
		log.Error(dm.serverCtx, "Failed to load draft when removing pick listener, searching all drafts", "Draft Id", draftId, "Error", err)
		for _, d := range dm.drafts {
			d.pickManager.RemoveListener(listener)
		}
		return
	}
	draft.pickManager.RemoveListener(listener)
}

// TODO decompose this into messages
func (dm *DraftManager) UpdateDraft(draftModel model.DraftModel) error {
	log.Info(dm.serverCtx, "UpdateDraft: acquiring locks", "Draft Id", draftModel.Id)
	loadLock := dm.getLoadLock(draftModel.Id)
	transitionLock := dm.getTransitionLock(draftModel.Id)
	loadLock.Lock()
	log.Info(dm.serverCtx, "UpdateDraft: acquired loadLock", "Draft Id", draftModel.Id)
	transitionLock.Lock()
	log.Info(dm.serverCtx, "UpdateDraft: acquired transitionLock", "Draft Id", draftModel.Id)
	defer transitionLock.Unlock()
	log.Info(dm.serverCtx, "UpdateDraft: calling model.UpdateDraft", "Draft Id", draftModel.Id)
	err := dm.draftStore.UpdateDraft(dm.serverCtx, &draftModel)
	log.Info(dm.serverCtx, "UpdateDraft: model.UpdateDraft returned", "Draft Id", draftModel.Id, "Error", err)
	loadLock.Unlock()
	if err != nil {
		log.Warn(dm.serverCtx, "Failed to update draft", "Error", err)
		return err
	}
	log.Info(dm.serverCtx, "UpdateDraft: calling GetDraft with reload", "Draft Id", draftModel.Id)
	_, err = dm.GetDraft(dm.serverCtx, draftModel.Id, true)
	log.Info(dm.serverCtx, "UpdateDraft: GetDraft returned", "Draft Id", draftModel.Id, "Error", err)
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

	newExpirationTime := utils.GetPickExpirationTime(dm.serverCtx, currentPick.ExpirationTime, extention)
	log.Info(dm.serverCtx, "Setting new pick expiration time", "Current Pick Time", currentPick.ExpirationTime, "New Expiration Time", newExpirationTime, "Pick Id", currentPick.Id)

	err = dm.draftStore.UpdatePickExpirationTime(dm.serverCtx, currentPick.Id, newExpirationTime)
	if err != nil {
		log.Error(dm.serverCtx, "Failed to update pick expiration time", "Pick Id", currentPick.Id, "Error", err)
		loadLock.Unlock()
		return errors.New("failed to update pick expiration time")
	}

	loadLock.Unlock()
	_, err = dm.GetDraft(dm.serverCtx, draftId, true)

	return err
}

func (dm *DraftManager) GetTbaHandler() *tbaHandler.TbaHandler {
	return dm.tbaHandler
}
