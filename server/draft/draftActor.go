package draft

import (
	"context"
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
	Initiator int
	UpdatedOwnerId int
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
	discordBus *discord.DiscordWebhookBus
	// TODO Does tba handler need to be a pointer?
	tbaHandler *tbaHandler.TBAHandler
	pickNotifier *picking.PickNotifier
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

func NewDraftActor(ctx context.Context, draftId int, draftStore model.DraftStore, tbaHandler *tbaHandler.TBAHandler, discordStore model.DiscordStore, discordBus *discord.DiscordWebhookBus, pickNotifier *picking.PickNotifier) (*DraftActor, error) {
	actor := &DraftActor {
		inbox: make(chan Message, 100),
		draftStore: draftStore,
		tbaHandler: tbaHandler,
		discordStore: discordStore,
		discordBus: discordBus,
		pickNotifier: pickNotifier,
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
	executeTransition(context context.Context, draft model.DraftModel) error
}

type ToStartTransition struct {
	draftStore model.DraftStore
}

func (tst *ToStartTransition) executeTransition(ctx context.Context, draft model.DraftModel) error {
	return tst.draftStore.UpdateDraftStatus(ctx, draft.Id, model.WAITING_TO_START)
}

type ToPickingTransition struct {
	draftStore model.DraftStore
}

func (tpt *ToPickingTransition) executeTransition(ctx context.Context, draft model.DraftModel) error {
	err := tpt.draftStore.RandomizePickOrder(ctx, draft.Id)
	if err != nil {
		return err
	}
	nextPickPlayer, err := tpt.draftStore.NextPick(ctx, draft.Id)
	if err != nil {
		log.Error(ctx, "failed to get next pick when transitioning to picking", "draftId", draft.Id, "error", err)
		return err
	}
	_, err = tpt.draftStore.MakePickAvailable(ctx, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(ctx, time.Now(), utils.PICK_TIME))
	if err != nil {
		log.Error(ctx, "failed to make first pick available", "draftId", draft.Id, "error", err)
	}
	err = tpt.draftStore.UpdateDraftStatus(ctx, draft.Id, model.PICKING)
	if err != nil {
		log.Error(ctx, "Failed to update draft status", "draftId", draft.Id, "error", err)
		return err
	}
	return nil
}

type ToPlayingTransition struct {
	draftStore model.DraftStore
}

func (tpt *ToPlayingTransition) executeTransition(ctx context.Context, draft model.DraftModel) error {
	log.Info(ctx, "Executing TEAMS_PLAYING playing transition", "draftId", draft.Id)
	err := tpt.draftStore.UpdateDraftStatus(ctx, draft.Id, model.TEAMS_PLAYING)
	if err != nil {
		log.Error(ctx, "Failed to update draft status", "draftId", draft.Id, "error", err)
	}

	//Remove the draft from the pick daemon
	return nil
}

type ToCompleteTransition struct {
	draftStore model.DraftStore
}

func (tct *ToCompleteTransition) executeTransition(ctx context.Context, draft model.DraftModel) error {
	return tct.draftStore.UpdateDraftStatus(ctx, draft.Id, model.COMPLETE)
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
	states[model.FILLING].transitions[model.PICKING] = &ToPickingTransition{
		draftStore: draftStore,
	}

	states[model.PICKING] = &state{
		state:       model.PICKING,
		transitions: make(map[model.DraftState]stateTransition),
	}
	states[model.PICKING].transitions[model.TEAMS_PLAYING] = &ToPlayingTransition{
		draftStore: draftStore,
	}

	states[model.TEAMS_PLAYING] = &state{
		state:       model.TEAMS_PLAYING,
		transitions: make(map[model.DraftState]stateTransition),
	}
	states[model.TEAMS_PLAYING].transitions[model.COMPLETE] = &ToCompleteTransition{
		draftStore: draftStore,
	}

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
			break
		}

		result := d.handleMessage(message)

		if message.Reply != nil {
			select {
			case message.Reply <- result:
			case <- time.After(5 * time.Second):
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

	// If more than 8 players are invites then we cancel the other outstanding invites
	// Maybe we need an active bool
	// Check that accepting this invite will not lead to more than eight players being in the draft
	numPlayers, err := d.draftStore.GetNumPlayersInInvitedDraft(ctx, msg.InviteId)
	if err != nil {
		log.Error(ctx, "Failed to get num players in invited draft", "error", err, "inviteId", msg.InviteId)
		return Result{
			Error: err,
		}
	}
	if numPlayers >= 8 {
		if err := d.draftStore.CancelOutstandingInvites(ctx, d.draftState.Id); err != nil {
			log.Error(ctx, "Failed to cancel outstanding invites", "error", err, "draftId", d.draftState.Id)
		}
		return Result{
			Error: errors.New("too many players are already in the draft; please contact the draft owner if you think this is an error"),
		}
	}

	draftId, playerId, err := d.draftStore.AcceptInvite(ctx, msg.InviteId)
	if err != nil {
		log.Error(ctx, "Failed to accept invite", "error", err, "inviteId", msg.InviteId)
		return Result{
			Error: err,
		}
	}
	if err := d.draftStore.AddPlayerToDraft(ctx, draftId, playerId); err != nil {
		log.Error(ctx, "Failed to add player to draft", "error", err, "draftId", draftId, "playerId", playerId)
		return Result{
			Error: err,
		}
	}

	if numPlayers >= 7 {
		if err := d.draftStore.CancelOutstandingInvites(ctx, d.draftState.Id); err != nil {
			log.Error(ctx, "Failed to cancel outstanding invites", "error", err, "draftId", d.draftState.Id)
			return Result{
				Error: err,
			}
		}
	}

	// Reload draft state so cached model is not stale
	updatedDraft, err := d.draftStore.GetDraft(ctx, d.draftState.Id)
	if err != nil {
		log.Error(ctx, "Failed to reload draft after accepting invite", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}
	d.mu.Lock()
	d.draftState = updatedDraft
	d.mu.Unlock()

	return Result{}
}

func (d *DraftActor) handleDeclineInvite(ctx context.Context, msg DeclineInviteMessage) Result {
	invite, err := d.draftStore.GetInvite(ctx, msg.InviteId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Result{
				Error: errors.New("invite not found. It may have been cancelled or expired."),
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
			Error: errors.New("you are not allowed to decline invites for other players."),
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

	if acceptedPlayers < 8 && d.draftState.Status == model.WAITING_TO_START {
		err = d.draftStore.UpdateDraftStatus(ctx, d.draftState.Id, model.FILLING)
		if err != nil {
			log.Error(ctx, "Failed to revert draft status to filling after decline", "error", err, "draftId", d.draftState.Id)
		}
	}

	// Reload draft state so cached model is not stale
	updatedDraft, err := d.draftStore.GetDraft(ctx, d.draftState.Id)
	if err != nil {
		log.Error(ctx, "Failed to reload draft after declining invite", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}
	d.mu.Lock()
	d.draftState = updatedDraft
	d.mu.Unlock()

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

	// Reload draft state so cached model is not stale
	updatedDraft, err := d.draftStore.GetDraft(ctx, d.draftState.Id)
	if err != nil {
		log.Error(ctx, "Failed to reload draft after inviting player", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}
	d.mu.Lock()
	d.draftState = updatedDraft
	d.mu.Unlock()

	return Result{}
}

func (d *DraftActor) handleUninvitePlayer(ctx context.Context, msg UninvitePlayerMessage) Result {
	if d.draftState.Status != model.FILLING {
		return Result{
			Error: errors.New("Draft must be in FILLING state to uninvite players"),
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

	// Reload draft state so cached model is not stale
	updatedDraft, err := d.draftStore.GetDraft(ctx, d.draftState.Id)
	if err != nil {
		log.Error(ctx, "Failed to reload draft after uninviting player", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}
	d.mu.Lock()
	d.draftState = updatedDraft
	d.mu.Unlock()

	return Result{}
}

func (d *DraftActor) handleStateTransition(ctx context.Context, msg StateTransitionMessage) Result {
	log.Info(ctx, "Got request to execute draft state transition", "draftId", d.draftState.Id, "requestedState", msg.RequestedState)
	assert := assert.CreateAssertWithContext("Execute Draft State Transition")

	assert.AddContext("draftId", d.draftState.Id)
	assert.AddContext("Current Draft Model State", string(d.draftState.Status))
	assert.AddContext("requestedState", string(msg.RequestedState))

	state, stateFound := d.states[d.draftState.Status]
	assert.AddContext("Current Draft State", state)
	assert.RunAssert(ctx, stateFound, "Current draft state is not registered in state machine")
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
	err := transition.executeTransition(ctx, d.draftState)
	if err != nil {
		log.Error(ctx, "Failed to execute draft state transition", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}
	log.Info(ctx, "Executed draft state transition", "draftId", d.draftState.Id)

	// Reload draft state so cached model is not stale
	updatedDraft, err := d.draftStore.GetDraft(ctx, d.draftState.Id)
	if err != nil {
		log.Error(ctx, "Failed to reload draft after state transition", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}
	d.mu.Lock()
	d.draftState = updatedDraft
	d.mu.Unlock()

	return Result{}
}

func (d *DraftActor) handlePick(ctx context.Context, msg PickMessage) Result {
	pickingComplete := false

	if !msg.Pick.Pick.Valid {
		return Result{
			Error: errors.New("no team entered"),
			Value: false,
		}
	}

	// Check that we are still trying to make the current pick
	currentPick := d.draftState.CurrentPick
	if currentPick.Id != msg.Pick.Id {
		log.Warn(ctx, "Pick attempt made against pick that is not the current pick", "currentPickId", currentPick.Id, "attemptedPickId", msg.Pick.Id)
		return Result{
			Error: errors.New("attempting to make pick that is not the current pick"),
			Value: false,
		}
	}

	validator := NewPickValidator(d.tbaHandler, d.draftStore, d.draftState.Id)
	err := validator.ValidatePick(ctx, msg.Pick)
	if err != nil {
		return Result{
			Error: err,
			Value: false,
		}
	}

	//If we have not found any errors indicating that the pick is invalid, make the pick
	err = d.draftStore.MakePick(ctx, msg.Pick)
	if err != nil {
		return Result{
			Error: err,
			Value: false,
		}
	}

	// Reload draft state so cached model is not stale
	updatedDraft, err := d.draftStore.GetDraft(ctx, d.draftState.Id)
	if err != nil {
		log.Error(ctx, "Failed to reload draft after pick", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
			Value: false,
		}
	}
	d.mu.Lock()
	d.draftState = updatedDraft
	d.mu.Unlock()

		nextPickPlayer, err := d.draftStore.NextPick(ctx, d.draftState.Id)
		if err != nil {
			log.Error(ctx, "failed to get next pick", "pickId", msg.Pick.Id, "error", err)
			return Result{
				Error: err,
				Value: false,
			}
		}

		//Make the next pick available if we havn't already made all picks
		picks, err := d.draftStore.GetPicks(ctx, d.draftState.Id)
		if err != nil {
			log.Error(ctx, "Failed to get picks", "draftId", d.draftState.Id, "error", err)
			return Result{
				Error: err,
				Value: false,
			}
		}

		log.Debug(ctx, "Checking if we should make another pick available", "numPicks", len(picks))
		if len(picks) < 64 {
			log.Debug(ctx, "Making next pick available", "draftId", d.draftState.Id)
		expirationTime := utils.GetPickExpirationTime(ctx, time.Now(), utils.PICK_TIME)
		_, err = d.draftStore.MakePickAvailable(ctx, nextPickPlayer.Id, time.Now(), expirationTime)
		if err != nil {
			log.Error(ctx, "Failed to make pick available", "draftPlayerId", msg.Pick.Player, "error", err)
			return Result{
				Error: err,
				Value: false,
			}
		}
	} else {
		log.Info(ctx, "Draft Complete", "draftId", d.draftState.Id)
		pickingComplete = true
	}

	currPickDiscordId, err := d.discordStore.GetPlayerDiscordId(ctx, currentPick.Player)
	if err != nil {
		log.Warn(ctx, "Could not get current pick draft player id", "draftPlayerId", msg.Pick.Player, "error", err)
	}

	currPickUser, err := d.draftStore.GetDraftPlayerUser(ctx, currentPick.Player)
	if err != nil {
		log.Warn(ctx, "Could not get current pick draft player name", "draftPlayerId", msg.Pick.Player, "error", err)
	}
	currPickName := currPickUser.Username

	draftWebhook, err := d.discordStore.GetDraftWebhook(ctx, d.draftState.Id)
	if err != nil {
		log.Warn(ctx, "Could not get draft webhook", "draftId", d.draftState.Id, "error", err)
	}

	event := discord.NextPickDiscordEvent{
		PreviousPickedTeam:    msg.Pick.Pick.String,
		PreviousPickName:      currPickName,
		PreviousPickDiscordId: currPickDiscordId,
		Webhook:               draftWebhook,
		DraftComplete:         pickingComplete,
	}

	if len(picks) < 64 {
		nextPickDiscordId, err := d.discordStore.GetPlayerDiscordId(ctx, nextPickPlayer.Id)
		if err != nil {
			log.Warn(ctx, "Could not get next pick draft player id", "draftPlayerId", nextPickPlayer.Id, "error", err)
		}

		nextPickUser, err := d.draftStore.GetDraftPlayerUser(ctx, nextPickPlayer.Id)
		if err != nil {
			log.Warn(ctx, "Could not get next pick draft player name", "draftPlayerId", nextPickPlayer.Id, "error", err)
		}
		nextPickName := nextPickUser.Username

		expirationTime := utils.GetPickExpirationTime(ctx, time.Now().UTC(), utils.PICK_TIME)
		event.NextPickName = nextPickName
		event.NextPickDiscordId = nextPickDiscordId
		event.ExpirationTime = expirationTime
	}

	go func() {
		if err := d.discordBus.PostPickNotification(event); err != nil {
			log.Error(ctx, "Failed to post discord webhook", "error", err)
		}
	}()

	if pickingComplete {
		log.Info(ctx, "Update status to TEAMS_PLAYING", "draftId", d.draftState.Id)
		err := d.PostMessage(ctx, Message{
			Content: StateTransitionMessage{
				RequestedState: model.TEAMS_PLAYING,
			},
		})
		if err != nil {
			log.Error(ctx, "Failed to post state transition message after pick", "draftId", d.draftState.Id, "error", err)
		}
	}

	log.Info(ctx, "Pick successful", "draftId", d.draftState.Id, "pickId", msg.Pick.Id, "team", msg.Pick.Pick.String)

	// Notify listeners on every successful pick so live updates work
	go d.notifyListeners(ctx, picking.PickEvent{
		Pick:    msg.Pick,
		Success: true,
		Err:     nil,
		DraftId: d.draftState.Id,
	})

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

	newExpirationTime := utils.GetPickExpirationTime(ctx, d.draftState.CurrentPick.ExpirationTime, msg.Extension)
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
	// TODO: Wrap SkipPick and MakePickAvailable in a database transaction to prevent partial failure.
	// If SkipPick succeeds but MakePickAvailable fails, the draft will be stuck with a skipped pick and no next pick.
	// This requires refactoring skipPick() and makePickAvailable() in model/draft.go to accept a DBTX interface
	// (works with both *sql.DB and *sql.Tx), then adding a new DraftStore method like:
	//   SkipAndMakeNextPickAvailable(ctx, currentPickId, nextDraftPlayerId, availableTime, expirationTime) (newPickId, error)
	// which runs both operations inside a single sql.Tx.
	if msg.CurrentPickId != d.draftState.CurrentPick.Id {
		log.Warn(ctx, "Stale skip request rejected", "Message PickId", msg.CurrentPickId, "Current PickId", d.draftState.CurrentPick.Id)
		return Result{
			Error: errors.New("pick has changed since skip was requested"),
		}
	}

	pickingComplete := false

	err := d.draftStore.SkipPick(ctx, d.draftState.CurrentPick.Id)
	if err != nil {
		log.Error(ctx, "Failed to skip current pick", "currentPickId", d.draftState.CurrentPick.Id, "error", err)
		return Result{
			Error: err,
		}
	}

	// Only make the next pick available if the draft is not already complete
	if len(d.draftState.Picks) < 64 {
		nextPickPlayer := d.getNextPick(ctx)
		_, err = d.draftStore.MakePickAvailable(ctx, nextPickPlayer.Id, time.Now().UTC(), utils.GetPickExpirationTime(ctx, time.Now().UTC(), utils.PICK_TIME))
		if err != nil {
			log.Error(ctx, "Failed to make pick available when skipping current pick", "currentPickId", d.draftState.CurrentPick.Id, "error", err)
			return Result{
				Error: err,
			}
		}
	} else {
		log.Info(ctx, "Draft complete after skipping last pick", "draftId", d.draftState.Id)
		pickingComplete = true
	}

	// Reload draft state after skip so cached model is not stale
	updatedDraft, err := d.draftStore.GetDraft(ctx, d.draftState.Id)
	if err != nil {
		log.Error(ctx, "Failed to reload draft after skip", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}
	d.mu.Lock()
	d.draftState = updatedDraft
	d.mu.Unlock()

	if pickingComplete {
		log.Info(ctx, "Update status to TEAMS_PLAYING", "draftId", d.draftState.Id)
		err := d.PostMessage(ctx, Message{
			Content: StateTransitionMessage{
				RequestedState: model.TEAMS_PLAYING,
			},
		})
		if err != nil {
			log.Error(ctx, "Failed to post state transition message after skip", "draftId", d.draftState.Id, "error", err)
		}
	}

	log.Info(ctx, "Pick skipped", "draftId", d.draftState.Id, "pickId", d.draftState.CurrentPick.Id)

	event := picking.PickEvent{
		Pick:    model.Pick{},
		Success: true,
		Err:     nil,
		DraftId: d.draftState.Id,
	}

	go d.notifyListeners(ctx, event)

	return Result{Value: true}
}

func (d *DraftActor) handleUndoLastPick(ctx context.Context, msg UndoLastPickMessage) Result {
	// TODO: Wrap DeletePick and ResetPick in a database transaction to prevent partial failure.
	// If DeletePick succeeds but ResetPick fails, the draft will be stuck with the current pick deleted
	// and the previous pick not reset. Same pattern as skip: refactor deletePick() and resetPick()
	// in model/draft.go to accept a DBTX interface, then add an UndoLastPick() transactional method.
	// Use the database to get the previous pick reliably
	previousPick, err := d.draftStore.GetPreviousPick(ctx, d.draftState.Id, d.draftState.CurrentPick.Id)
	if err != nil {
		log.Error(ctx, "Failed to get previous pick", "draftId", d.draftState.Id, "currentPickId", d.draftState.CurrentPick.Id, "error", err)
		return Result{
			Error: errors.New("failed to get previous pick"),
		}
	}

	// Delete the current pick
	err = d.draftStore.DeletePick(ctx, d.draftState.CurrentPick.Id)
	if err != nil {
		log.Error(ctx, "Failed to delete current pick", "pickId", d.draftState.CurrentPick.Id, "error", err)
		return Result{
			Error: errors.New("failed to delete current pick"),
		}
	}

	// Set the expiration time to 3 hours from now
	newExpirationTime := time.Now().UTC().Add(3 * time.Hour)

	// Reset the previous pick (null out pick and pickTime, and set new expiration)
	err = d.draftStore.ResetPick(ctx, previousPick.Id, newExpirationTime)
	if err != nil {
		log.Error(ctx, "Failed to reset previous pick", "pickId", previousPick.Id, "error", err)
		return Result{
			Error: errors.New("failed to reset previous pick"),
		}
	}

	// Reload draft state after undo so cached model is not stale
	updatedDraft, err := d.draftStore.GetDraft(ctx, d.draftState.Id)
	if err != nil {
		log.Error(ctx, "Failed to reload draft after undo", "draftId", d.draftState.Id, "error", err)
		return Result{
			Error: err,
		}
	}
	d.mu.Lock()
	d.draftState = updatedDraft
	d.mu.Unlock()

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
	// TODO: Add store method for transferring ownership when available
	return Result{
		Error: errors.New("transfer draft ownership is not yet supported"),
	}
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

func (d *DraftActor) getNextPick(ctx context.Context) model.DraftPlayer {
	assert := assert.CreateAssertWithContext("Get Next Pick")
	assert.AddContext("draftId", d.draftState.Id)
	assert.AddContext("currentPickId", d.draftState.CurrentPick)
	assert.RunAssert(ctx, len(d.draftState.Players) > 0, "Draft has no players when finding next pick")

	var nextPlayer model.DraftPlayer

	// Only two players is an edge case so we just hard code it
	if len(d.draftState.Picks) < 2 {
		for _, player := range d.draftState.Players {
			if int(player.PlayerOrder.Int16) == len(d.draftState.Picks) {
				nextPlayer = player
			}
		}
		assert.RunAssert(ctx, nextPlayer.Id != 0, "Next player has invalid id")
		return nextPlayer
	}

	lastPlayer := GetDraftPlayerFromDraft(ctx, d.draftState, d.draftState.Picks[len(d.draftState.Picks)-1].Player)
	secondLastPick := GetDraftPlayerFromDraft(ctx, d.draftState, d.draftState.Picks[len(d.draftState.Picks)-2].Player)
	assert.RunAssert(ctx, lastPlayer.PlayerOrder.Valid, "Got player order which was not set when finding next pick")
	direction := lastPlayer.PlayerOrder.Int16 - secondLastPick.PlayerOrder.Int16
	if lastPlayer.User.UserUuid == secondLastPick.User.UserUuid {
		if int(lastPlayer.PlayerOrder.Int16) == len(d.draftState.Players)-1 {
			direction = -1
		} else {
			direction = 1
		}
	}
	if len(d.draftState.Picks) % len(d.draftState.Players) == 0 {
		direction = 0
	}

	nextIndex := lastPlayer.PlayerOrder.Int16 + direction
	assert.RunAssert(ctx, nextIndex >= 0 && int(nextIndex) < len(d.draftState.Players), "next pick is out of bounds")
	nextPlayer = d.draftState.Players[nextIndex]
	assert.RunAssert(ctx, nextPlayer.Id != 0, "Next player has invalid id")
	return nextPlayer
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

func (d *DraftActor) close() {
	d.mu.Lock()
	d.shutdown = true
	d.mu.Unlock()
}

func GetDraftPlayerFromDraft(ctx context.Context, draft model.DraftModel, draftPlayerId int) model.DraftPlayer {
	for _, p := range draft.Players {
		if p.Id == draftPlayerId {
			return p
		}
	}
	return model.DraftPlayer{}
}

func (d *DraftActor) GetDraftState() model.DraftModel {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.draftState
}
