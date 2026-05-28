package draft

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/assert"
	"server/discord"
	"server/log"
	"server/middleware"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"server/utils"
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

type AddPickListenerMessage struct {
	Listener picking.PickListener
}

type RemovePickListenerMessage struct {
	Listener picking.PickListener
}

type SkipCurrentPickMessage struct {
	CurrentPickId int
}

type UndoLastPickMessage struct {
	CurrentPickId int
}

type UpdateDraftProfileMessage struct {
	Name string
	Description string
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

type DeclineInviteMessage struct {
	InviteId int
}

type DraftActor struct {
	inbox chan Message
	draftStore model.DraftStore
	draftState model.DraftModel
	discordStore model.DiscordStore
	discordBus *discord.DiscordWebhookBus
	tbaHandler *tbaHandler.TbaHandler
	// TODO pickNotifier is stored but never used; remove or wire up
	pickNotifier *picking.PickNotifier
	states map[model.DraftState]*state
	listeners    []picking.PickListener
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

func NewDraftActor(ctx context.Context, draftId int, draftStore model.DraftStore, tbaHandler *tbaHandler.TbaHandler, discordStore model.DiscordStore, discordBus *discord.DiscordWebhookBus, pickNotifier *picking.PickNotifier) (*DraftActor, error) {
	actor := &DraftActor {
		inbox: make(chan Message, 100),
		draftStore: draftStore,
		tbaHandler: tbaHandler,
		discordStore: discordStore,
		discordBus: discordBus,
		pickNotifier: pickNotifier,
		// TODO duplicate state machine: DraftManager also sets up identical states; consolidate to single source of truth
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
		log.Warn(ctx, "failed to get next pick when transitioning to picking", "Draft Id", draft.Id, "Error", err)
		return err
	}
	_, err = tpt.draftStore.MakePickAvailable(ctx, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(ctx, time.Now(), utils.PICK_TIME))
	if err != nil {
		log.Warn(ctx, "failed to make first pick available", "Draft Id", draft.Id, "Error", err)
 	}
	err = tpt.draftStore.UpdateDraftStatus(ctx, draft.Id, model.PICKING)
	if err != nil {
		log.Error(ctx, "Failed to update draft status", "Draft Id", draft.Id, "Error", err)
		return err
	}
	return nil
}

type ToPlayingTransition struct {
	draftStore model.DraftStore
}

func (tpt *ToPlayingTransition) executeTransition(ctx context.Context, draft model.DraftModel) error {
	log.Info(ctx, "Executing TEAMS_PLAYING playing transition", "Draft Id", draft.Id)
	err := tpt.draftStore.UpdateDraftStatus(ctx, draft.Id, model.TEAMS_PLAYING)
	if err != nil {
		log.Error(ctx, "Failed to update draft status", "Draft Id", draft.Id, "Error", err)
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
	states[model.FILLING].transitions[model.WAITING_TO_START] = &ToStartTransition{
		draftStore: draftStore,
	}

	states[model.WAITING_TO_START] = &state{
		state:       model.WAITING_TO_START,
		transitions: make(map[model.DraftState]stateTransition),
	}
	states[model.WAITING_TO_START].transitions[model.PICKING] = &ToPickingTransition{
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
	message.context = ctx

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
		result := d.handleMessage(message)

		if message.Reply != nil {
			select {
			case message.Reply <- result:
			case <- time.After(5 * time.Second):
			}
		}
	}

	d.close()
}

func (d *DraftActor) handleMessage(message Message) Result {
	switch msg := message.Content.(type) {
	case StateTransitionMessage:
		return d.handleStateTransition(message.context, msg)
	case PickMessage:
		return d.handlePick(message.context, msg)
	case ModifyExpirationTimeMessage:
		return d.handleModifyExpiraitonTime(message.context, msg)
	case AddPickListenerMessage:
		return d.handleAddPickListener(message.context, msg)
	case RemovePickListenerMessage:
		return d.handleRemovePickListener(message.context, msg)
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
	case DeclineInviteMessage:
		return d.handleDeclineInvite(message.context, msg)
	default:
		return Result{
			Error: fmt.Errorf("unknown message type: %T", msg),
		}
	}
}

func (d *DraftActor) handleAcceptInvite(ctx context.Context, msg AcceptInviteMessage) Result {
	// TODO Way too much db stuff going on here
	invite, err := d.draftStore.GetInvite(ctx, msg.InviteId)
	if err != nil {
		log.Error(ctx, "Failed to get invite", "error", err, "inviteId", msg.InviteId)
		if errors.Is(err, sql.ErrNoRows) {
			return Result{
				Error: errors.New("invite not found. It may have been cancelled or expired."),
			}
		}
		return Result{
			Error: fmt.Errorf("Could not accept invite. If this continued please contact support and provide this reference id: %s", middleware.GetCorrelationID(ctx)),
		}
	}

	//Make sure that other players cannot accept someones draft
	if invite.InvitedUserUuid != msg.AcceptingUserUuid {
		log.Info(ctx, "Invited player to draft", "Invited User Uuid", invite.InvitedUserUuid, "Inviting User Uuid", msg.AcceptingUserUuid)
		return Result{
			Error: errors.New("you are not allowed to accept drafts for other players."),
		}
	}

	log.Info(ctx, "Accepting invite from player", "Invite Id", msg.InviteId, "User Id", msg.AcceptingUserUuid)

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
			Error: errors.New("too many players are already in the draft. Please contect the draft owner if you think this is an error."),
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

	// TODO Figure out how to get the draft player id
	// playerIdx := d.getPlayerIndex()

	return Result{}
}

func (d *DraftActor) getPlayerIndex(userUuid uuid.UUID) int {
	for i, player := range d.draftState.Players {
		if player.User.UserUuid == userUuid {
			return i
		}
	}
	return -1
}

func (d *DraftActor) handleDeclineInvite(ctx context.Context, msg DeclineInviteMessage) Result {
	// TODO support declines here
	return Result{}
}

func (d *DraftActor) handleInvitePlayer(ctx context.Context, msg InvitePlayerMessage) Result {
	// Check that the draft is in the correct state
	if d.draftState.Status != model.FILLING {
		return Result{
			Error: errors.New("Draft must be in FILLING state to invite players"),
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

	return Result{}
}

func (d *DraftActor) handleStateTransition(ctx context.Context, msg StateTransitionMessage) Result {
	log.Info(ctx, "Got request to execute draft state transition", "Draft Id", d.draftState.Id, "Requested State", msg.RequestedState)
	assert := assert.CreateAssertWithContext("Execute Draft State Transition")

	assert.AddContext("Draft Id", d.draftState.Id)
	assert.AddContext("Current Draft Model State", string(d.draftState.Status))
	assert.AddContext("Requested State", string(msg.RequestedState))

	state, stateFound := d.states[d.draftState.Status]
	assert.AddContext("Current Draft State", state)
	assert.RunAssert(ctx, stateFound, "Current draft state is not registed in state machine")
	log.Debug(ctx, "Found draft state", "Draft Id", d.draftState.Id, "State", state.state)
	transition, transitionFound := state.transitions[msg.RequestedState]
	if !transitionFound {
		log.Error(ctx, "Did not find draft state transition", "Current State", d.draftState.Status, "Requested State", msg.RequestedState)
		return Result{
			Error: &invalidStateTransitionError{
				currentState: d.draftState.Status,
				requestedState: msg.RequestedState,
			},
		}
	}

	log.Info(ctx, "Executing Draft State Transition", "Draft Id", d.draftState.Id, "Requested State", msg.RequestedState)
	err := transition.executeTransition(ctx, d.draftState)
	if err != nil {
		log.Warn(ctx, "Failed to execute draft state transition", "Draft Id", d.draftState.Id, "Error", err)
		return Result{
			Error: err,
		}
	}
	log.Info(ctx, "Executed draft state transition", "Draft Id", d.draftState.Id)
	// TODO actor state is never updated after transition; d.draftState.Status is now stale

	return Result{}
}

// TODO this function is still 180 lines. We need to split it up
func (d *DraftActor) handlePick(ctx context.Context, msg PickMessage) Result {
	pickingComplete := false

	var err error
	if !msg.Pick.Pick.Valid {
		err = errors.New("no team entered")
		return Result{
			Error: err,
			Value: false,
		}
	}

	// Check that we are still trying to make the current pick
	currentPick := d.draftState.CurrentPick
	if currentPick.Id != msg.Pick.Id {
		log.Warn(ctx, "Pick attempt made against pick that is not the current pick", "Current Pick", currentPick.Id, "Attempted Pick", msg.Pick.Id)
		return Result{
			Error: errors.New("attempting to make pick that is not the current pick"),
			Value: false,
		}
	}

	// TODO dead code: err is guaranteed nil here (set to nil on line 410 and never changed)
	if err != nil {
		return Result {
			Error: err,
			Value: false,
		}
	}

	validator := NewPickValidator(d.tbaHandler, d.draftStore, d.draftState.Id)
	err = validator.ValidatePick(ctx, msg.Pick)
	if err != nil {
		return Result {
			Error: err,
			Value: false,
		}
	}

	// TODO decide what we need to migrate out of the database layer
	//If we have not found any errors indicating that the pick is invalid, make the pick
	err = d.draftStore.MakePick(ctx, msg.Pick)
	if err != nil {
		return Result{
			Error: err,
			Value: false,
		}
	}
	// TODO cached draftState is not updated after MakePick; CurrentPick and Picks are stale

	// TODO probably migrate this out of the db layer
	nextPickPlayer, err := d.draftStore.NextPick(ctx, d.draftState.Id)
	if err != nil {
		log.Warn(ctx, "failed to get next pick", "Pick Id", msg.Pick.Id, "Errors", err)
		return Result{
			Error: err,
			Value: false,
		}
	}

	//Make the next pick available if we havn't aleady made all picks
	picks, err := d.draftStore.GetPicks(ctx, d.draftState.Id)

	if err != nil {
		log.Warn(ctx, "Failed to get picks", "Draft Id", d.draftState.Id, "Error", err)
		return Result{
			Error: err,
			Value: false,
		}
	}

	log.Info(ctx, "Checking if we should make another pick available", "Num picks", len(picks))
	// TODO magic number 64 is fragile; does not account for skips or variable player counts
	if len(picks) < 64 {
		log.Info(ctx, "Making next pick available", "Draft Id", d.draftState.Id)
		expirationTime := utils.GetPickExpirationTime(ctx, time.Now(), utils.PICK_TIME)
		_, err = d.draftStore.MakePickAvailable(ctx, nextPickPlayer.Id, time.Now(), expirationTime)
		if err != nil {
			log.Warn(ctx, "Failed to make pick available", "Draft Player Id", msg.Pick.Player, "Error", err)
			return Result{
				Error: err,
				Value: false,
			}
		}

		currPickDiscordId, err := d.discordStore.GetPlayerDiscordId(ctx, currentPick.Player)
		if err != nil {
			log.Warn(ctx, "Could not get current pick draft player id", "Draft Player Id", msg.Pick.Player, "Error", err)
			return Result{
				Error: err,
				Value: false,
			}
		}

		currPickUser, err := d.draftStore.GetDraftPlayerUser(ctx, currentPick.Player)
		if err != nil {
			log.Warn(ctx, "Could not get current pick draft player name", "Draft Player Id", msg.Pick.Player, "Error", err)
			return Result{
				Error: err,
				Value: false,
			}
		}
		currPickName := currPickUser.Username

		nextPickDiscordId, err := d.discordStore.GetPlayerDiscordId(ctx, nextPickPlayer.Id)
		if err != nil {
			log.Warn(ctx, "Could not get next pick draft player id", "Draft Player Id", nextPickPlayer.Id, "Error", err)
			return Result{
				Error: err,
				Value: false,
			}
		}

		nextPickUser, err := d.draftStore.GetDraftPlayerUser(ctx, nextPickPlayer.Id)
		if err != nil {
			log.Warn(ctx, "Could not get next pick draft player name", "Draft Player Id", nextPickPlayer.Id, "Error", err)
			return Result{
				Error: err,
				Value: false,
			}
		}
		nextPickName := nextPickUser.Username

		draftWebhook, err := d.discordStore.GetDraftWebhook(ctx, d.draftState.Id)
		if err != nil {
			log.Warn(ctx, "Could not get draft webhook", "Draft Id", d.draftState.Id, "Error", err)
			return Result{
				Error: err,
				Value: false,
			}
		}

		event := discord.NextPickDiscordEvent{
			PreviousPickedTeam:    msg.Pick.Pick.String,
			PreviousPickName:      currPickName,
			PreviousPickDiscordId: currPickDiscordId,
			NextPickName:          nextPickName,
			NextPickDiscordId:     nextPickDiscordId,
			Webhook:               draftWebhook,
			ExpirationTime: 	   expirationTime,
		}
		go func() {
			err = d.discordBus.PostPickNotification(event)
			if err != nil {
				log.Warn(ctx, "Failed to post discord webhook", "Error", err)
			}
		}()
	} else {
		log.Info(ctx, "Draft Complete", "Draft Id", d.draftState.Id)
		// Set draft to the teams playing state
		// This isnt entirely correct becuase it doesnt account for skips
		// But I dont care about that for this year
		pickingComplete = true
	}

	if err != nil {
		log.Info(ctx, "Failed to make pick", "Pick", msg.Pick.Pick.String, "Pick Id", msg.Pick.Id, "Player", msg.Pick.Player, "Error", err)
		return Result{
			Error: err,
			Value: false,
		}
	}

	if pickingComplete {
		log.Info(ctx, "Update status to TEAMS_PLAYING", "Draft Id", d.draftState.Id)
		// TODO post message to this service. Need to figure out how we want this to work becuase that new message will be blocked
		// TODO compilation error: DraftActor has no method ExecuteDraftStateTransition (only DraftManager does)
		// TODO compilation error: d.draftState.id should be Id
		d.PostMessage(ctx, Message{
			Content: StateTransitionMessage{
				RequestedState: model.TEAMS_PLAYING,
			},
		})

		// TODO notify listeners
		// TODO undefined variables: draft.pickManager and pick; this code does not compile
		go d.notifyListeners(ctx, picking.PickEvent{
			Pick:    msg.Pick,
			Success: err == nil,
			Err:     err,
			DraftId: d.draftState.Id,
		})
	}
	return Result{
		Value: true,
	}
}

func (d *DraftActor) handleModifyExpiraitonTime(ctx context.Context, msg ModifyExpirationTimeMessage) Result {
	newExpirationTime := utils.GetPickExpirationTime(ctx, d.draftState.CurrentPick.ExpirationTime, msg.Extension)
	log.Info(ctx, "Setting new pick expiration time", "Current Pick Time", d.draftState.CurrentPick.ExpirationTime, "New Expiration Time", newExpirationTime, "Pick Id", d.draftState.CurrentPick.Id)

	err := d.draftStore.UpdatePickExpirationTime(ctx, d.draftState.CurrentPick.Id, newExpirationTime)
	if err != nil {
		log.Error(ctx, "Failed to update pick expiration time", "Pick Id", d.draftState.CurrentPick.Id, "Error", err)
		return Result{
			Error: errors.New("failed to update pick expiration time"),
		}
	}
	d.draftState.CurrentPick.ExpirationTime = newExpirationTime

	return Result{
		Value: newExpirationTime,
	}
}

func (d *DraftActor) handleAddPickListener(ctx context.Context, msg AddPickListenerMessage) Result {
	// TODO not implemented: does not actually add the listener to d.listeners
	return Result{}
}

func (d *DraftActor) handleRemovePickListener(ctx context.Context, msg RemovePickListenerMessage) Result {
	// TODO not implemented: does not actually remove the listener from d.listeners
	return Result{}
}

func (d *DraftActor) handleSkipCurrentPick(ctx context.Context, msg SkipCurrentPickMessage) Result {
	nextPickPlayer := d.getNextPick(ctx)
	// TODO Wrap skip and make next available in a transaction
	err := d.draftStore.SkipPick(ctx, d.draftState.CurrentPick.Id)
	if err != nil {
		log.Warn(ctx, "Failed to skip current pick", "Current pick", d.draftState.CurrentPick.Id, "Error", err)
		return Result{
			Error: err,
		}
	}
	_, err = d.draftStore.MakePickAvailable(ctx, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(ctx, time.Now(), utils.PICK_TIME))
	if err != nil {
		log.Warn(ctx, "Failed to make pick available when skipping current pick", "Current pick", d.draftState.CurrentPick.Id, "Error", err)
		return Result{
			Error: err,
		}
	}

	event := picking.PickEvent{
		Pick:    model.Pick{},
		Success: true,
		Err:     nil,
		DraftId: d.draftState.Id,
	}

	// TODO How do we want to make this happen?
	go d.notifyListeners(ctx, event)

	return Result{}
}

func (d *DraftActor) handleUndoLastPick(ctx context.Context, msg UndoLastPickMessage) Result {
	// TODO Will need some transactions here too
	// Get the previous pick
	previousPick, err := d.getPreviousPick(ctx)
	if err != nil {
		return Result{
			Error: err,
		}
	}

	// Delete the current pick
	err = d.draftStore.DeletePick(ctx, d.draftState.CurrentPick.Id)
	if err != nil {
		log.Error(ctx, "Failed to delete current pick", "Pick Id", d.draftState.CurrentPick.Id, "Error", err)
		return Result{
			Error: errors.New("failed to delete current pick"),
		}
	}
	d.draftState.CurrentPick = previousPick

	// Set the expiration time to 3 hours from now
	newExpirationTime := time.Now().Add(3 * time.Hour)

	// Reset the previous pick (null out pick and pickTime, and set new expiration)
	err = d.draftStore.ResetPick(ctx, previousPick.Id, newExpirationTime)
	if err != nil {
		log.Error(ctx, "Failed to reset previous pick", "Pick Id", previousPick.Id, "Error", err)
		return Result{
			Error: errors.New("failed to reset previous pick"),
		}
	}
	d.draftState.CurrentPick.Pick = sql.NullString{
		Valid: false,
	}

	return Result{}
}

func (d *DraftActor) handleUpdateDraftProfile(ctx context.Context, msg UpdateDraftProfileMessage) Result {
	// TODO not implemented
	return Result{}
}

func (d *DraftActor) handleTransferDraftOwnership(ctx context.Context, msg TransferDraftOwnershipMessage) Result {
	// TODO not implemented
	return Result{}
}

func (d *DraftActor) getPreviousPick(ctx context.Context) (model.Pick, error) {
	if len(d.draftState.Picks) == 0 {
		return model.Pick{}, errors.New("cannot undo pick from draft with no picks")
	}

	if len(d.draftState.Picks) == 1 {
		// TODO returns empty Pick with nil error, which callers treat as valid and assign to CurrentPick
		return model.Pick{}, nil
	}

	return d.draftState.Picks[len(d.draftState.Picks) - 2], nil
}

func (d *DraftActor) getNextPick(ctx context.Context) model.DraftPlayer {
	// TODO risky logic: assertion on line 741 can panic with negative direction; empty DraftPlayer returned silently in edge cases
	assert := assert.CreateAssertWithContext("Get Next Pick")
	assert.AddContext("Draft Id", d.draftState.Id)
	assert.AddContext("Current Pick", d.draftState.CurrentPick)

	//We need to get the last two picks
	var nextPlayer model.DraftPlayer

	//I dont think we need to account for the case where there are only two players
	if len(d.draftState.Picks) < 2 {
		for _, player := range d.draftState.Players {
			if int(player.PlayerOrder.Int16) == len(d.draftState.Picks) {
				nextPlayer = player
			}
		}
	} else {
		//We can then figure out what direction
		//we are going and if we hit the
		//end then we decide what the next pick is
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

		//We know draft.players is order by player order
		assert.RunAssert(ctx, int16(len(d.draftState.Players)) > lastPlayer.PlayerOrder.Int16+direction && lastPlayer.PlayerOrder.Int16+direction >= 0, "Next pick is out of bounds")
		nextPlayer = d.draftState.Players[lastPlayer.PlayerOrder.Int16+direction]
	}

	//Take the pick and make it into a draft player
	return nextPlayer
}

func (d *DraftActor) notifyListeners(ctx context.Context, pickEvent picking.PickEvent) {
	log.DebugNoContext("Started notifying pick listeners", "Draft Id", pickEvent.DraftId, "Pick", pickEvent.Pick.Pick.String, "Num Listeners", len(d.listeners))

	for _, listener := range d.listeners {
		go func(l picking.PickListener) {
			log.DebugNoContext("Notifying pick listener", "Draft Id", pickEvent.DraftId, "Pick", pickEvent.Pick.Pick.String)
			if err := l.ReceivePickEvent(pickEvent); err != nil {
				log.Warn(context.TODO(), "Removing dead listener", "Listener", l, "Error", err)
				d.removeListener(ctx, l)
			}
		}(listener)
	}
	log.DebugNoContext("Finished notifying pick listeners", "Draft Id", pickEvent.DraftId)
}

func (d *DraftActor) removeListener(ctx context.Context, listener picking.PickListener) {
	removalMessage := Message {
		Content: RemovePickListenerMessage {
			Listener: listener,
		},
	}
	err := d.PostMessage(ctx, removalMessage)
	if err != nil {
		log.Warn(ctx, "Failed to remove draft listener", "Draft Id", d.draftState.Id, "Error", err)
	}
}

func (d *DraftActor) close() {
}

func GetDraftPlayerFromDraft(ctx context.Context, draft model.DraftModel, draftPlayerId int) model.DraftPlayer {
	for _, p := range draft.Players {
		if p.Id == draftPlayerId {
			return p
		}
	}
	return model.DraftPlayer{}
}

func (d *DraftActor) GetDraftState() (model.DraftModel) {
	return d.draftState
}
