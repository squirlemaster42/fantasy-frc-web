package draft

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/database"
	"server/discord"
	"server/log"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"server/utils"
	"sync"
	"time"

	"github.com/google/uuid"
)

type StateTransitionMessage struct {
	RequestedState model.DraftState
}

type PickMessage struct {
	Pick model.Pick
}

type ModifyExpirationTimeMessage struct {
	PickId int
	Extension time.Duration
}

type SkipCurrentPickMessage struct {
	CurrentPickId int
}

type UndoLastPickMessage struct {
	CurrentPickId int
}

type UpdateDraftProfileMessage struct {
	Name           string
	Description    string
	Interval       int
	DiscordWebhook string
}

type TransferDraftOwnershipMessage struct {
	Initiator      int
	UpdatedOwnerId uuid.UUID
}

type InvitePlayerMessage struct {
	Invite model.DraftInvite
}

type AcceptInviteMessage struct {
	InviteId int
	AcceptingUserUuid uuid.UUID
}

type UninvitePlayerMessage struct {
	DraftId   int
	OwnerUuid uuid.UUID
	InviteId  int
}

type DeclineInviteMessage struct {
	InviteId int
	UserUuid uuid.UUID
}

type ShutdownMessage struct{}

type DraftActor struct {
	inbox chan Message
	draftStore model.DraftStore
	draftState model.DraftModel
	discordStore model.DiscordStore
	discordBus discord.DiscordNotifier
	tbaHandler tbaHandler.TBAInterface
	pickNotifier *picking.PickNotifier
	pickConfig utils.PickWindowConfig
	states map[model.DraftState]*state
	shutdown bool
	mu sync.RWMutex
}

type Message struct {
	Content any
	context context.Context
	Reply chan Result
}

type Result struct {
	Value any
	Error error
}

type invalidStateTransitionError struct {
	currentState   model.DraftState
	requestedState model.DraftState
}

func (e *invalidStateTransitionError) Error() string {
	return fmt.Sprintf("Invalid state transition where current state was %s and requested state was %s", e.currentState, e.requestedState)
}

func NewDraftActor(ctx context.Context, draftId int, draftStore model.DraftStore, tbaHandler tbaHandler.TBAInterface, discordStore model.DiscordStore, discordBus discord.DiscordNotifier, pickNotifier *picking.PickNotifier, pickConfig utils.PickWindowConfig) (*DraftActor, error) {
	actor := &DraftActor {
		inbox: make(chan Message, 100),
		draftStore: draftStore,
		tbaHandler: tbaHandler,
		discordStore: discordStore,
		discordBus: discordBus,
		pickNotifier: pickNotifier,
		pickConfig: pickConfig,
		states: setupStates(ctx, draftStore),
	}

	draft, err := draftStore.GetDraft(ctx, draftId)
	if err != nil {
		return &DraftActor{}, err
	}

	actor.draftState = draft

	go actor.run()

	return actor, nil
}

type stateTransition interface {
	executeTransition(ctx context.Context, store model.DraftStore, draft model.DraftModel, pickConfig utils.PickWindowConfig) error
}

type ToStartTransition struct{}

func (tst *ToStartTransition) executeTransition(ctx context.Context, store model.DraftStore, draft model.DraftModel, pickConfig utils.PickWindowConfig) error {
	return store.UpdateDraftStatus(ctx, draft.Id, model.WAITING_TO_START)
}

type ToPickingTransition struct{}

func (tpt *ToPickingTransition) executeTransition(ctx context.Context, store model.DraftStore, draft model.DraftModel, pickConfig utils.PickWindowConfig) error {
	if err := store.RandomizePickOrder(ctx, draft.Id); err != nil {
		return err
	}
	nextPickPlayer, err := store.NextPick(ctx, draft.Id)
	if err != nil {
		log.Error(ctx, "failed to get next pick when transitioning to picking", "draftId", draft.Id, "error", err)
		return err
	}
	if _, err := store.MakePickAvailable(ctx, nextPickPlayer.Id, time.Now(), pickConfig.GetPickExpirationTime(ctx, time.Now(), pickConfig.PickTime)); err != nil {
		log.Error(ctx, "failed to make first pick available", "draftId", draft.Id, "error", err)
		return err
	}
	if err := store.UpdateDraftStatus(ctx, draft.Id, model.PICKING); err != nil {
		log.Error(ctx, "Failed to update draft status", "draftId", draft.Id, "error", err)
		return err
	}
	return nil
}

type ToPlayingTransition struct{}

func (tpt *ToPlayingTransition) executeTransition(ctx context.Context, store model.DraftStore, draft model.DraftModel, pickConfig utils.PickWindowConfig) error {
	log.Info(ctx, "Executing TEAMS_PLAYING playing transition", "draftId", draft.Id)
	if err := store.UpdateDraftStatus(ctx, draft.Id, model.TEAMS_PLAYING); err != nil {
		log.Error(ctx, "Failed to update draft status", "draftId", draft.Id, "error", err)
		return err
	}

	//Remove the draft from the pick daemon
	return nil
}

type ToCompleteTransition struct{}

func (tct *ToCompleteTransition) executeTransition(ctx context.Context, store model.DraftStore, draft model.DraftModel, pickConfig utils.PickWindowConfig) error {
	return store.UpdateDraftStatus(ctx, draft.Id, model.COMPLETE)
}

type state struct {
	state       model.DraftState
	transitions map[model.DraftState]stateTransition
}

func setupStates(ctx context.Context, draftStore model.DraftStore) map[model.DraftState]*state {
	states := make(map[model.DraftState]*state)
	states[model.FILLING] = &state{
		state:       model.FILLING,
		transitions: make(map[model.DraftState]stateTransition),
	}
	states[model.FILLING].transitions[model.PICKING] = &ToPickingTransition{}

	states[model.PICKING] = &state{
		state:       model.PICKING,
		transitions: make(map[model.DraftState]stateTransition),
	}
	states[model.PICKING].transitions[model.TEAMS_PLAYING] = &ToPlayingTransition{}

	states[model.TEAMS_PLAYING] = &state{
		state:       model.TEAMS_PLAYING,
		transitions: make(map[model.DraftState]stateTransition),
	}
	states[model.TEAMS_PLAYING].transitions[model.COMPLETE] = &ToCompleteTransition{}

	states[model.COMPLETE] = &state{
		state:       model.COMPLETE,
		transitions: make(map[model.DraftState]stateTransition),
	}
	return states
}

func (d *DraftActor) PostMessage(ctx context.Context, message Message) error {
	d.mu.RLock()
	shutdown := d.shutdown
	d.mu.RUnlock()
	if shutdown {
		return errors.New("draft actor is shutting down")
	}

	// Detach from HTTP request so actor work survives request completion
	detachedCtx := context.Background()
	if corrID := log.GetCorrelationID(ctx); corrID != "" {
		detachedCtx = log.WithCorrelationID(detachedCtx, corrID)
	}
	message.context = detachedCtx

	select {
	case d.inbox <- message:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return errors.New("timeout posting message to draft actor inbox")
	}
}

func (d *DraftActor) run() {
	for message := range d.inbox {
		if _, isShutdown := message.Content.(ShutdownMessage); isShutdown {
			d.close()
			if message.Reply != nil {
				select {
				case message.Reply <- Result{}:
				case <-time.After(5 * time.Second):
				}
			}
			break
		}

		result := d.handleMessage(message)

		if message.Reply != nil {
			select {
			case message.Reply <- result:
			case <-time.After(5 * time.Second):
			}
		}
	}
}

func (d *DraftActor) handleMessage(message Message) Result {
	switch msg := message.Content.(type) {
	case StateTransitionMessage:
		return d.handleStateTransition(message.context, msg)
	case PickMessage:
		return d.handlePick(message.context, msg)
	case ModifyExpirationTimeMessage:
		return d.handleModifyExpirationTime(message.context, msg)
	case ShutdownMessage:
		return d.handleShutdown(message.context, msg)
	case SkipCurrentPickMessage:
		return d.handleSkipCurrentPick(message.context, msg)
	case UndoLastPickMessage:
		return d.handleUndoLastPick(message.context, msg)
	case UpdateDraftProfileMessage:
		return d.handleUpdateDraftProfile(message.context, msg)
	case TransferDraftOwnershipMessage:
		return d.handleTransferDraftOwnership(message.context, msg)
	case InvitePlayerMessage:
		return d.handleInvitePlayer(message.context, msg)
	case AcceptInviteMessage:
		return d.handleAcceptInvite(message.context, msg)
	case UninvitePlayerMessage:
		return d.handleUninvitePlayer(message.context, msg)
	case DeclineInviteMessage:
		return d.handleDeclineInvite(message.context, msg)
	default:
		return Result{
			Error: fmt.Errorf("unknown message type: %T", msg),
		}
	}
}

var ErrTooManyPlayers = errors.New("too many players are already in the draft; please contact the draft owner if you think this is an error")

func (d *DraftActor) handleAcceptInvite(ctx context.Context, msg AcceptInviteMessage) Result {
	invite, err := d.draftStore.GetInvite(ctx, msg.InviteId)
	if err != nil {
		log.Error(ctx, "Failed to get invite", "error", err, "inviteId", msg.InviteId)
		if errors.Is(err, sql.ErrNoRows) {
			return Result{
				Error: errors.New("invite not found; it may have been cancelled or expired"),
			}
		}
		return Result{
			Error: fmt.Errorf("could not accept invite; if this continued please contact support and provide this reference id: %s", log.GetCorrelationID(ctx)),
		}
	}

	//Make sure that other players cannot accept someones draft
	if invite.InvitedUserUuid != msg.AcceptingUserUuid {
		log.Warn(ctx, "Invited player to draft", "invitedUserUuid", invite.InvitedUserUuid, "acceptingUserUuid", msg.AcceptingUserUuid)
		return Result{
			Error: errors.New("you are not allowed to accept drafts for other players"),
		}
	}

	log.Info(ctx, "Accepting invite from player", "inviteId", msg.InviteId, "userUuid", msg.AcceptingUserUuid)

	err = d.draftStore.RunInTransaction(ctx, func(tx database.DBTX) error {
		store := d.draftStore.WithTx(tx)

		if err := store.LockDraft(ctx, d.draftState.Id); err != nil {
			return err
		}

		numPlayers, err := store.GetNumPlayersInDraft(ctx, d.draftState.Id)
		if err != nil {
			return err
		}
		if numPlayers >= model.DraftPlayerCount {
			return ErrTooManyPlayers
		}

		draftId, playerId, err := store.AcceptInvite(ctx, msg.InviteId)
		if err != nil {
			return err
		}
		if err := store.AddPlayerToDraft(ctx, draftId, playerId); err != nil {
			return err
		}

		if numPlayers >= model.DraftPlayerCount-1 {
			return store.CancelOutstandingInvites(ctx, d.draftState.Id)
		}
		return nil
	})
	if err != nil {
	if errors.Is(err, ErrTooManyPlayers) {
		if cancelErr := d.draftStore.CancelOutstandingInvites(ctx, d.draftState.Id); cancelErr != nil {
			log.Error(ctx, "Failed to cancel outstanding invites", "error", cancelErr, "draftId", d.draftState.Id)
		}
	}

		log.Error(ctx, "Failed to accept invite", "error", err, "inviteId", msg.InviteId)
		return Result{
			Error: err,
		}
	}

	if err := d.reloadDraftState(ctx); err != nil {
		log.Error(ctx, "Failed to reload draft after accepting invite", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}

	return Result{}
}

func (d *DraftActor) handleDeclineInvite(ctx context.Context, msg DeclineInviteMessage) Result {
	invite, err := d.draftStore.GetInvite(ctx, msg.InviteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Result{
				Error: errors.New("invite not found. It may have been cancelled or expired"),
			}
		}
		log.Error(ctx, "Failed to get invite", "error", err, "inviteId", msg.InviteId)
		return Result{
			Error: fmt.Errorf("could not decline invite. If this continues please contact support and provide this reference id: %s", log.GetCorrelationID(ctx)),
		}
	}

	if invite.InvitedUserUuid != msg.UserUuid {
		log.Info(ctx, "User attempted to decline invite for another player", "InvitedUserUuid", invite.InvitedUserUuid, "RequestingUserUuid", msg.UserUuid)
		return Result{
			Error: errors.New("you are not allowed to decline invites for other players"),
		}
	}

	err = d.draftStore.CancelInvite(ctx, msg.InviteId)
	if err != nil {
		log.Error(ctx, "Failed to cancel invite", "error", err, "inviteId", msg.InviteId)
		return Result{
			Error: fmt.Errorf("could not decline invite. If this continues please contact support and provide this reference id: %s", log.GetCorrelationID(ctx)),
		}
	}

	// Check whether the draft should revert from WAITING_TO_START to FILLING
	acceptedPlayers := 0
	for _, player := range d.draftState.Players {
		if !player.Pending {
			acceptedPlayers++
		}
	}

	if acceptedPlayers < model.DraftPlayerCount && d.draftState.Status == model.WAITING_TO_START {
		err = d.draftStore.UpdateDraftStatus(ctx, d.draftState.Id, model.FILLING)
		if err != nil {
			log.Error(ctx, "Failed to revert draft status to filling after decline", "error", err, "draftId", d.draftState.Id)
		}
	}

	if err := d.reloadDraftState(ctx); err != nil {
		log.Error(ctx, "Failed to reload draft after declining invite", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}

	return Result{}
}

func (d *DraftActor) handleInvitePlayer(ctx context.Context, msg InvitePlayerMessage) Result {
	// Check that the draft is in the correct state
	if d.draftState.Status != model.FILLING {
		return Result{
			Error: errors.New("draft must be in FILLING state to invite players"),
		}
	}

	isOwner := msg.Invite.InvitingUserUuid == d.draftState.Owner.UserUuid
	if !isOwner {
		return Result{
			Error: errors.New("you must own the draft to invite a player"),
		}
	}

	_, err := d.draftStore.InvitePlayer(ctx, d.draftState.Id, msg.Invite.InvitingUserUuid, msg.Invite.InvitedUserUuid)
	if err != nil {
		log.Error(ctx, "Failed to invite player", "error", err)
		return Result{
			Error: err,
		}
	}

	if err := d.reloadDraftState(ctx); err != nil {
		log.Error(ctx, "Failed to reload draft after inviting player", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}

	return Result{}
}

func (d *DraftActor) handleUninvitePlayer(ctx context.Context, msg UninvitePlayerMessage) Result {
	if d.draftState.Status != model.FILLING {
		return Result{
			Error: errors.New("draft must be in FILLING state to uninvite players"),
		}
	}

	if msg.OwnerUuid != d.draftState.Owner.UserUuid {
		return Result{
			Error: errors.New("you must own the draft to uninvite a player"),
		}
	}

	err := d.draftStore.UninvitePlayer(ctx, msg.DraftId, msg.OwnerUuid, msg.InviteId)
	if err != nil {
		log.Error(ctx, "Failed to uninvite player", "error", err)
		return Result{
			Error: err,
		}
	}

	if err := d.reloadDraftState(ctx); err != nil {
		log.Error(ctx, "Failed to reload draft after uninviting player", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}

	return Result{}
}

func (d *DraftActor) handleStateTransition(ctx context.Context, msg StateTransitionMessage) Result {
	log.Info(ctx, "Got request to execute draft state transition", "draftId", d.draftState.Id, "requestedState", msg.RequestedState)

	state, stateFound := d.states[d.draftState.Status]
	if !stateFound {
		return Result{Error: fmt.Errorf("current draft state is not registered in state machine")}
	}
	log.Debug(ctx, "Found draft state", "draftId", d.draftState.Id, "State", state.state)
	transition, transitionFound := state.transitions[msg.RequestedState]
	if !transitionFound {
		log.Error(ctx, "Did not find draft state transition", "currentState", d.draftState.Status, "requestedState", msg.RequestedState)
		return Result{
			Error: &invalidStateTransitionError{
				currentState: d.draftState.Status,
				requestedState: msg.RequestedState,
			},
		}
	}

	log.Info(ctx, "Executing Draft State Transition", "draftId", d.draftState.Id, "requestedState", msg.RequestedState)
		err := d.draftStore.RunInTransaction(ctx, func(tx database.DBTX) error {
		transactedStore := d.draftStore.WithTx(tx)
		return transition.executeTransition(ctx, transactedStore, d.draftState, d.pickConfig)
	})
	if err != nil {
		log.Error(ctx, "Failed to execute draft state transition", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}
	log.Info(ctx, "Executed draft state transition", "draftId", d.draftState.Id)

	if err := d.reloadDraftState(ctx); err != nil {
		log.Error(ctx, "Failed to reload draft after state transition", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}

	return Result{}
}

func (d *DraftActor) executeTransition(ctx context.Context, store model.DraftStore, requestedState model.DraftState) error {
	state, stateFound := d.states[d.draftState.Status]
	if !stateFound {
		return fmt.Errorf("current draft state is not registered in state machine")
	}
	transition, transitionFound := state.transitions[requestedState]
	if !transitionFound {
		log.Error(ctx, "Did not find draft state transition", "currentState", d.draftState.Status, "requestedState", requestedState)
		return &invalidStateTransitionError{
			currentState:   d.draftState.Status,
			requestedState: requestedState,
		}
	}
	return transition.executeTransition(ctx, store, d.draftState, d.pickConfig)
}



func (d *DraftActor) validatePickInput(ctx context.Context, msg PickMessage) error {
	if !msg.Pick.Pick.Valid {
		return errors.New("no team entered")
	}

	if d.draftState.CurrentPick.Id != msg.Pick.Id {
		log.Warn(ctx, "Pick attempt made against pick that is not the current pick", "currentPickId", d.draftState.CurrentPick.Id, "attemptedPickId", msg.Pick.Id)
		return errors.New("attempting to make pick that is not the current pick")
	}

	validator := NewPickValidator(d.tbaHandler, d.draftStore, d.draftState.Id)
	return validator.ValidatePick(ctx, msg.Pick)
}

func (d *DraftActor) handlePick(ctx context.Context, msg PickMessage) Result {
	if err := d.validatePickInput(ctx, msg); err != nil {
		return Result{
			Error: err,
			Value: false,
		}
	}

	previousPickPlayerId := d.draftState.CurrentPick.Player
	previouslyPickedTeam := msg.Pick.Pick.String

	pickingComplete := len(d.draftState.Picks) == model.PicksPerDraft
	expirationTime := d.pickConfig.GetPickExpirationTime(ctx, time.Now(), d.pickConfig.PickTime)

	var nextPickPlayer model.DraftPlayer
	var nextPickErr error
	if !pickingComplete {
		nextPickPlayer, nextPickErr = model.DetermineNextPick(d.draftState.Players, d.draftState.Picks)
		if nextPickErr != nil {
			log.Error(ctx, "Failed to determine next pick", "draftId", d.draftState.Id, "error", nextPickErr)
			return Result{
				Error: nextPickErr,
				Value: false,
			}
		}
	}

	err := d.draftStore.RunInTransaction(ctx, func(tx database.DBTX) error {
		store := d.draftStore.WithTx(tx)
		if err := store.MakePick(ctx, msg.Pick); err != nil {
			return err
		}
		if pickingComplete {
			return d.executeTransition(ctx, store, model.TEAMS_PLAYING)
		}
		_, err := store.MakePickAvailable(ctx, nextPickPlayer.Id, time.Now(), expirationTime)
		return err
	})
	if err != nil {
		return Result{
			Error: err,
			Value: false,
		}
	}

	if err := d.reloadDraftState(ctx); err != nil {
		log.Error(ctx, "Failed to reload draft after pick", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
			Value: false,
		}
	}

	d.sendPickDiscordNotification(ctx, previousPickPlayerId, previouslyPickedTeam, nextPickPlayer, pickingComplete, false)

	log.Info(ctx, "Pick successful", "draftId", d.draftState.Id, "pickId", msg.Pick.Id, "team", msg.Pick.Pick.String)

	d.notifyPickListeners(ctx, msg.Pick)

	return Result{
		Value: true,
	}
}

func (d *DraftActor) handleModifyExpirationTime(ctx context.Context, msg ModifyExpirationTimeMessage) Result {
	if msg.PickId != d.draftState.CurrentPick.Id {
		log.Warn(ctx, "Attempted to modify expiration time for stale pick", "messagePickId", msg.PickId, "currentPickId", d.draftState.CurrentPick.Id)
		return Result{
			Error: errors.New("pick id does not match current pick"),
		}
	}

	newExpirationTime := d.pickConfig.GetPickExpirationTime(ctx, d.draftState.CurrentPick.ExpirationTime, msg.Extension)
	log.Debug(ctx, "Setting new pick expiration time", "currentPickTime", d.draftState.CurrentPick.ExpirationTime, "newExpirationTime", newExpirationTime, "pickId", d.draftState.CurrentPick.Id)

	err := d.draftStore.UpdatePickExpirationTime(ctx, d.draftState.CurrentPick.Id, newExpirationTime)
	if err != nil {
		log.Error(ctx, "Failed to update pick expiration time", "pickId", d.draftState.CurrentPick.Id, "error", err)
		return Result{
			Error: errors.New("failed to update pick expiration time"),
		}
	}
	d.mu.Lock()
	d.draftState.CurrentPick.ExpirationTime = newExpirationTime
	d.mu.Unlock()

	return Result{
		Value: newExpirationTime,
	}
}

func (d *DraftActor) handleShutdown(ctx context.Context, msg ShutdownMessage) Result {
	log.Info(ctx, "Shutting down draft actor", "draftId", d.draftState.Id)
	return Result{}
}

func (d *DraftActor) handleSkipCurrentPick(ctx context.Context, msg SkipCurrentPickMessage) Result {
	if msg.CurrentPickId != d.draftState.CurrentPick.Id {
		log.Warn(ctx, "Stale skip request rejected", "Message PickId", msg.CurrentPickId, "Current PickId", d.draftState.CurrentPick.Id)
		return Result{
			Error: errors.New("pick has changed since skip was requested"),
		}
	}

	skippedPlayerId := d.draftState.CurrentPick.Player
	var nextPickPlayer model.DraftPlayer
	var pickingComplete bool

	if len(d.draftState.Picks) < model.PicksPerDraft {
		nextPick, err := d.getNextPick(ctx)
		if err != nil {
			log.Error(ctx, "Failed to get next pick when skipping current pick", "currentPickId", d.draftState.CurrentPick.Id, "error", err)
			return Result{
				Error: err,
			}
		}
		nextPickPlayer = nextPick
		err = d.draftStore.RunInTransaction(ctx, func(tx database.DBTX) error {
			store := d.draftStore.WithTx(tx)
			if err := store.SkipPick(ctx, d.draftState.CurrentPick.Id); err != nil {
				return err
			}
			_, err := store.MakePickAvailable(ctx, nextPick.Id, time.Now().UTC(), d.pickConfig.GetPickExpirationTime(ctx, time.Now().UTC(), d.pickConfig.PickTime))
			return err
		})
		if err != nil {
			log.Error(ctx, "Failed to skip pick and make next pick available", "currentPickId", d.draftState.CurrentPick.Id, "error", err)
			return Result{
				Error: err,
			}
		}
	} else {
		err := d.draftStore.RunInTransaction(ctx, func(tx database.DBTX) error {
			store := d.draftStore.WithTx(tx)
			if err := store.SkipPick(ctx, d.draftState.CurrentPick.Id); err != nil {
				return err
			}
			return d.executeTransition(ctx, store, model.TEAMS_PLAYING)
		})
		if err != nil {
			log.Error(ctx, "Failed to skip current pick", "currentPickId", d.draftState.CurrentPick.Id, "error", err)
			return Result{
				Error: err,
			}
		}
		pickingComplete = true
	}

	if err := d.reloadDraftState(ctx); err != nil {
		return Result{Error: err}
	}

	if !pickingComplete {
		d.sendPickDiscordNotification(ctx, skippedPlayerId, "", nextPickPlayer, false, true)
	}

	log.Info(ctx, "Pick skipped", "draftId", d.draftState.Id, "pickId", d.draftState.CurrentPick.Id)

	d.notifyPickListeners(ctx, model.Pick{})

	return Result{Value: true}
}

func (d *DraftActor) handleUndoLastPick(ctx context.Context, msg UndoLastPickMessage) Result {
	previousPick, err := d.draftStore.GetPreviousPick(ctx, d.draftState.Id, d.draftState.CurrentPick.Id)
	if err != nil {
		log.Error(ctx, "Failed to get previous pick", "draftId", d.draftState.Id, "currentPickId", d.draftState.CurrentPick.Id, "error", err)
		return Result{
			Error: errors.New("failed to get previous pick"),
		}
	}

	newExpirationTime := d.pickConfig.GetPickExpirationTime(ctx, time.Now().UTC(), d.pickConfig.PickTime)

	err = d.draftStore.RunInTransaction(ctx, func(tx database.DBTX) error {
		store := d.draftStore.WithTx(tx)
		if err := store.DeletePick(ctx, d.draftState.CurrentPick.Id); err != nil {
			return err
		}
		return store.ResetPick(ctx, previousPick.Id, newExpirationTime)
	})
	if err != nil {
		log.Error(ctx, "Failed to reset previous pick", "pickId", previousPick.Id, "error", err)
		return Result{
			Error: errors.New("failed to reset previous pick"),
		}
	}

	if err := d.reloadDraftState(ctx); err != nil {
		log.Error(ctx, "Failed to reload draft after undo", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}

	return Result{}
}

func (d *DraftActor) handleUpdateDraftProfile(ctx context.Context, msg UpdateDraftProfileMessage) Result {
	draftModel := d.draftState
	draftModel.DisplayName = msg.Name
	draftModel.Description = msg.Description
	draftModel.Interval = msg.Interval
	draftModel.DiscordWebhook = msg.DiscordWebhook

	err := d.draftStore.UpdateDraft(ctx, &draftModel)
	if err != nil {
		log.Error(ctx, "Failed to update draft profile", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: errors.New("failed to update draft profile"),
		}
	}

	// Update cached fields directly — we know exactly what changed
	d.mu.Lock()
	d.draftState.DisplayName = msg.Name
	d.draftState.Description = msg.Description
	d.draftState.Interval = msg.Interval
	d.draftState.DiscordWebhook = msg.DiscordWebhook
	d.mu.Unlock()

	return Result{}
}

func (d *DraftActor) handleTransferDraftOwnership(ctx context.Context, msg TransferDraftOwnershipMessage) Result {
	err := d.draftStore.TransferOwnership(ctx, d.draftState.Id, msg.UpdatedOwnerId)
	if err != nil {
		log.Error(ctx, "Failed to transfer draft ownership", "draftId", d.draftState.Id, "error", err)
		return Result{Error: err}
	}
	d.mu.Lock()
	d.draftState.Owner.UserUuid = msg.UpdatedOwnerId
	d.mu.Unlock()
	return Result{}
}

func (d *DraftActor) getPreviousPick(ctx context.Context) (model.Pick, error) {
	if len(d.draftState.Picks) == 0 {
		return model.Pick{}, errors.New("cannot undo pick from draft with no picks")
	}

	if len(d.draftState.Picks) == 1 {
		return model.Pick{}, errors.New("cannot undo the first pick")
	}

	return d.draftState.Picks[len(d.draftState.Picks) - 2], nil
}

func (d *DraftActor) getNextPick(ctx context.Context) (model.DraftPlayer, error) {
	return model.DetermineNextPick(d.draftState.Players, d.draftState.Picks)
}

func (d *DraftActor) buildNextPickDiscordEvent(ctx context.Context, previousDraftPlayerId int, previousPickedTeam string, nextPickPlayer model.DraftPlayer, pickingComplete bool, skipped bool) (discord.NextPickDiscordEvent, error) {
	if d.discordStore == nil {
		return discord.NextPickDiscordEvent{}, errors.New("discord store not configured")
	}

	draftWebhook, err := d.discordStore.GetDraftWebhook(ctx, d.draftState.Id)
	if err != nil {
		return discord.NextPickDiscordEvent{}, err
	}

	currPickDiscordId, err := d.discordStore.GetPlayerDiscordId(ctx, previousDraftPlayerId)
	if err != nil {
		return discord.NextPickDiscordEvent{}, err
	}

	currPickUser, err := d.draftStore.GetDraftPlayerUser(ctx, previousDraftPlayerId)
	if err != nil {
		return discord.NextPickDiscordEvent{}, err
	}

	event := discord.NextPickDiscordEvent{
		PreviousPickedTeam:    previousPickedTeam,
		PreviousPickName:      currPickUser.Username,
		PreviousPickDiscordId: currPickDiscordId,
		Webhook:               draftWebhook,
		DraftComplete:         pickingComplete,
		Skipped:               skipped,
	}

	if !pickingComplete {
		nextPickDiscordId, err := d.discordStore.GetPlayerDiscordId(ctx, nextPickPlayer.Id)
		if err != nil {
			return discord.NextPickDiscordEvent{}, err
		}

		nextPickUser, err := d.draftStore.GetDraftPlayerUser(ctx, nextPickPlayer.Id)
		if err != nil {
			return discord.NextPickDiscordEvent{}, err
		}

		expirationTime := d.pickConfig.GetPickExpirationTime(ctx, time.Now().UTC(), d.pickConfig.PickTime)
		event.NextPickName = nextPickUser.Username
		event.NextPickDiscordId = nextPickDiscordId
		event.ExpirationTime = expirationTime
	}

	return event, nil
}

func (d *DraftActor) notifyListeners(ctx context.Context, pickEvent picking.PickEvent) {
	log.Debug(ctx, "Started notifying pick listeners", "draftId", pickEvent.DraftId, "pick", pickEvent.Pick.Pick.String)

	if d.pickNotifier != nil {
		go func() {
			if err := d.pickNotifier.ReceivePickEvent(ctx, pickEvent); err != nil {
				log.Error(ctx, "PickNotifier returned error", "draftId", pickEvent.DraftId, "error", err)
			}
		}()
	}
	log.Debug(ctx, "Finished notifying pick listeners", "draftId", pickEvent.DraftId)
}

func (d *DraftActor) reloadDraftState(ctx context.Context) error {
	updatedDraft, err := d.draftStore.GetDraft(ctx, d.draftState.Id)
	if err != nil {
		return err
	}
	d.mu.Lock()
	d.draftState = updatedDraft
	d.mu.Unlock()
	return nil
}

func (d *DraftActor) notifyPickListeners(ctx context.Context, pick model.Pick) {
	go d.notifyListeners(ctx, picking.PickEvent{
		Pick:    pick,
		Success: true,
		DraftId: d.draftState.Id,
	})
}

func (d *DraftActor) sendPickDiscordNotification(ctx context.Context, skippedPlayerId int, previousPickedTeam string, nextPickPlayer model.DraftPlayer, pickingComplete bool, skipped bool) {
	if d.discordBus == nil {
		return
	}
	event, err := d.buildNextPickDiscordEvent(ctx, skippedPlayerId, previousPickedTeam, nextPickPlayer, pickingComplete, skipped)
	if err != nil {
		log.Warn(ctx, "Failed to build pick notification event", "draftId", d.draftState.Id, "error", err)
		return
	}
	go func() {
		if err := d.discordBus.PostPickNotification(event); err != nil {
			log.Error(ctx, "Failed to post discord webhook", "error", err)
		}
	}()
}

func (d *DraftActor) close() {
	d.mu.Lock()
	d.shutdown = true
	d.mu.Unlock()
}

func (d *DraftActor) IsShutdown() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.shutdown
}

func (d *DraftActor) GetDraftPlayerIdByUuid(userUuid uuid.UUID) (int, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	for _, player := range d.draftState.Players {
		if player.User.UserUuid == userUuid {
			return player.Id, nil
		}
	}
	return 0, fmt.Errorf("player not found in draft")
}

func (d *DraftActor) GetDraftState() model.DraftModel {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.draftState
}
