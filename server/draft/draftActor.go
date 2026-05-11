package draft

import (
	"context"
	"database/sql"
	"fmt"
	"server/discord"
	"server/log"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"server/utils"
	"time"
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
	// TODO If we actually want this we should record who initiated the transfer
	Initiator int
	UpdatedOwnerId int
}

type InvitePlayerMessage struct {
	Invite model.DraftInvite
}

type AcceptInviteMessage struct {
	InviteId int
}

type DeclineInviteMessage struct {
	InviteId int
}

type DraftActor struct {
	inbox chan Message
	database *sql.DB
	draftState model.DraftModel
	discordBus *discord.DiscordWebhook
	tbaHandler *tbaHandler.TbaHandler
	states map[model.DraftState]*state
}

type Message struct {
	Content any
	Context context.Context
	Reply chan Result
}

type Result struct {
	Value any
	Error error
}

func NewDraftActor(ctx context.Context, draftId int, database *sql.DB, tbaHandler *tbaHandler.TbaHandler, discordBus *discord.DiscordWebhook) (*DraftActor, error) {
	actor := &DraftActor {
		inbox: make(chan Message, 100),
		database: database,
		tbaHandler: tbaHandler,
		discordBus: discordBus,
		states: setupStates(ctx, database),
	}

	draft, err := model.GetDraft(ctx, database, draftId)
	if err != nil {
		return &DraftActor{}, err
	}

	actor.draftState = draft

	go actor.run()

	return actor, nil
}

type stateTransition interface {
	executeTransition(context context.Context, draft Draft) error
}

type ToStartTransition struct {
	database *sql.DB
}

func (tst *ToStartTransition) executeTransition(ctx context.Context, draft Draft) error {
	return model.UpdateDraftStatus(ctx, tst.database, draft.draftId, model.WAITING_TO_START)
}

type ToPickingTransition struct {
	database *sql.DB
}

func (tpt *ToPickingTransition) executeTransition(ctx context.Context, draft Draft) error {
	err := model.RandomizePickOrder(ctx, tpt.database, draft.draftId)
	if err != nil {
		return err
	}
	nextPickPlayer := model.NextPick(ctx, tpt.database, draft.draftId)
	model.MakePickAvailable(ctx, tpt.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(ctx, time.Now(), utils.PICK_TIME))
	err = model.UpdateDraftStatus(ctx, tpt.database, draft.draftId, model.PICKING)
	if err != nil {
		log.Error(ctx, "Failed to update draft status", "Draft Id", draft.draftId, "Error", err)
		return err
	}
	return nil
}

type ToPlayingTransition struct {
	database *sql.DB
}

func (tpt *ToPlayingTransition) executeTransition(ctx context.Context, draft Draft) error {
	log.Info(ctx, "Executing TEAMS_PLAYING playing transition", "Draft Id", draft.draftId)
	err := model.UpdateDraftStatus(ctx, tpt.database, draft.draftId, model.TEAMS_PLAYING)
	if err != nil {
		log.Error(ctx, "Failed to update draft status", "Draft Id", draft.draftId, "Error", err)
	}
	//Remove the draft from the pick daemon
	return nil
}

type ToCompleteTransition struct {
	database *sql.DB
}

func (tct *ToCompleteTransition) executeTransition(ctx context.Context, draft Draft) error {
	return model.UpdateDraftStatus(ctx, tct.database, draft.draftId, model.COMPLETE)
}

type state struct {
	state       model.DraftState
	transitions map[model.DraftState]stateTransition
}

func setupStates(ctx context.Context, database *sql.DB) map[model.DraftState]*state {
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

func (d *DraftActor) run() {
	for message := range d.inbox {
		result := d.handleMessage(message)

		select {
		case message.Reply <- result:
		case <- time.After(5 * time.Second):
		}
	}

	d.close()
}

func (d *DraftActor) handleMessage(message Message) Result {
	switch msg := message.Content.(type) {
	case StateTransitionMessage:
		return d.handleStateTransition(message.Context, msg)
	case PickMessage:
		return d.handlePick(message.Context, msg)
	case ModifyExpirationTimeMessage:
		return d.handleModifyExpiraitonTime(message.Context, msg)
	case AddPickListenerMessage:
		return d.handleAddPickListener(message.Context, msg)
	case RemovePickListenerMessage:
		return d.handleRemovePickListener(message.Context, msg)
	case SkipCurrentPickMessage:
		return d.handleSkipCurrentPick(message.Context, msg)
	case UndoLastPickMessage:
		return d.handleUndoLastPick(message.Context, msg)
	case UpdateDraftProfileMessage:
		return d.handleUpdateDraftProfile(message.Context, msg)
	case TransferDraftOwnershipMessage:
		return d.handleTransferDraftOwnership(message.Context, msg)
	case InvitePlayerMessage:
		return d.handleInvitePlayer(message.Context, msg)
	case AcceptInviteMessage:
		return d.handleAcceptInvite(message.Context, msg)
	case DeclineInviteMessage:
		return d.handleDeclineInvite(message.Context, msg)
	default:
		return Result{
			Error: fmt.Errorf("unknown message type: %T", msg),
		}
	}
}

func (d *DraftActor) handleAcceptInvite(ctx context.Context, msg AcceptInviteMessage) Result {
	return Result{}
}

func (d *DraftActor) handleDeclineInvite(ctx context.Context, msg DeclineInviteMessage) Result {
	return Result{}
}

func (d *DraftActor) handleInvitePlayer(ctx context.Context, msg InvitePlayerMessage) Result {
	return Result{}
}

func (d *DraftActor) handleStateTransition(ctx context.Context, msg StateTransitionMessage) Result {
	return Result{}
}

func (d *DraftActor) handlePick(ctx context.Context, msg PickMessage) Result {
	return Result{}
}

func (d *DraftActor) handleModifyExpiraitonTime(ctx context.Context, msg ModifyExpirationTimeMessage) Result {
	return Result{}
}

func (d *DraftActor) handleAddPickListener(ctx context.Context, msg AddPickListenerMessage) Result {
	return Result{}
}

func (d *DraftActor) handleRemovePickListener(ctx context.Context, msg RemovePickListenerMessage) Result {
	return Result{}
}

func (d *DraftActor) handleSkipCurrentPick(ctx context.Context, msg SkipCurrentPickMessage) Result {
	return Result{}
}

func (d *DraftActor) handleUndoLastPick(ctx context.Context, msg UndoLastPickMessage) Result {
	return Result{}
}

func (d *DraftActor) handleUpdateDraftProfile(ctx context.Context, msg UpdateDraftProfileMessage) Result {
	return Result{}
}

func (d *DraftActor) handleTransferDraftOwnership(ctx context.Context, msg TransferDraftOwnershipMessage) Result {
	return Result{}
}

func (d *DraftActor) close() {
}

func (d *DraftActor) GetDraftState() (model.DraftModel) {
	return d.draftState
}

func (d *DraftActor) PostMessage(ctx context.Context, message Message) error {
	return nil
}
