package draft

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"server/log"
	"server/model"
	"server/picking"
	"time"
)

func SkipCurrentPick(ctx context.Context, draftActor *DraftActor, draftId int, currentPickId int) bool {
	replyChan := make(chan Result)
	skipped := false
	message := Message{
		Content: SkipCurrentPickMessage{
			CurrentPickId: currentPickId,
		},
		Reply: replyChan,
	}
	err := draftActor.PostMessage(ctx, message)
	if err != nil {
		log.Warn(ctx, "Failed to post skip message to draft actor", "Draft Id", draftId, "Error", err)
		return false
	}
	select {
	case result := <-message.Reply:
		if result.Error != nil || !result.Value.(bool) {
			log.Warn(ctx, "Skipping current pick in draft failed", "Draft Id", draftId, "Current Pick Id", draftActor.GetDraftState().CurrentPick.Id, result.Error)
			skipped = false
		} else {
			skipped = true
		}
	case <-time.After(5 * time.Second):
		log.Warn(ctx, "Skipping current pick in draft timed out", "Draft Id", draftId, "Current Pick Id", draftActor.GetDraftState().CurrentPick.Id)
		skipped = false
	}
	return skipped
}

func ModifyCurrentPickExpirationTime(ctx context.Context, draftActor *DraftActor, extension time.Duration) error {
	replyChan := make(chan Result)
	message := Message{
		Content: ModifyExpirationTimeMessage{
			PickId:    draftActor.GetDraftState().CurrentPick.Id,
			Extension: extension,
		},
		Reply: replyChan,
	}
	err := draftActor.PostMessage(ctx, message)
	if err != nil {
		return err
	}
	select {
	case result := <-message.Reply:
		return result.Error
	case <-time.After(5 * time.Second):
		return errors.New("timeout modifying pick expiration time")
	}
}

func GetCurrentPick(draftActor *DraftActor) model.Pick {
	return draftActor.GetDraftState().CurrentPick
}

func MakePick(ctx context.Context, draftActor *DraftActor, pick model.Pick) error {
	replyChan := make(chan Result)
	message := Message{
		Content: PickMessage{
			Pick: pick,
		},
		Reply: replyChan,
	}
	err := draftActor.PostMessage(ctx, message)
	if err != nil {
		return err
	}
	var pickError error
	select {
	case result := <-message.Reply:
		if result.Error != nil {
			pickError = result.Error
		}
	case <-time.After(5 * time.Second):
		return errors.New("timeout making pick")
	}
	return pickError
}

func UndoLastPick(ctx context.Context, draftActor *DraftActor) error {
	replyChan := make(chan Result)
	message := Message{
		Content: UndoLastPickMessage{
			CurrentPickId: draftActor.GetDraftState().CurrentPick.Id,
		},
		Reply: replyChan,
	}
	err := draftActor.PostMessage(ctx, message)
	if err != nil {
		return err
	}
	select {
	case result := <-message.Reply:
		return result.Error
	case <-time.After(5 * time.Second):
		return errors.New("timeout undoing last pick")
	}
}

func GetDraft(draftActor *DraftActor) model.DraftModel {
	return draftActor.GetDraftState()
}

func UpdateDraft(ctx context.Context, draftActor *DraftActor, draftModel model.DraftModel) error {
	replyChan := make(chan Result)
	message := Message{
		Content: UpdateDraftProfileMessage{
			Name:           draftModel.DisplayName,
			Description:    draftModel.Description,
			Interval:       draftModel.Interval,
			StartTime:      draftModel.StartTime,
			EndTime:        draftModel.EndTime,
			DiscordWebhook: draftModel.DiscordWebhook,
		},
		Reply: replyChan,
	}
	err := draftActor.PostMessage(ctx, message)
	if err != nil {
		return err
	}
	select {
	case result := <-message.Reply:
		return result.Error
	case <-time.After(5 * time.Second):
		return errors.New("timeout updating draft")
	}
}

func ExecuteDraftStateTransition(ctx context.Context, draftActor *DraftActor, requestedState model.DraftState) error {
	replyChan := make(chan Result)
	message := Message{
		Content: StateTransitionMessage{
			RequestedState: requestedState,
		},
		Reply: replyChan,
	}
	err := draftActor.PostMessage(ctx, message)
	if err != nil {
		return err
	}
	select {
	case result := <-message.Reply:
		if result.Error != nil {
			return result.Error
		}
	case <-time.After(5 * time.Second):
		return errors.New("timeout executing draft state transition")
	}
	return nil
}

func ShutdownActor(actorMap *DraftActorMap, ctx context.Context, draftId int) error {
	actor, ok := actorMap.actorMap.Load(draftId)
	if !ok {
		return nil
	}

	draftActor := actor.(*DraftActor)
	replyChan := make(chan Result)
	message := Message{
		Content: ShutdownMessage{},
		Reply:   replyChan,
	}
	err := draftActor.PostMessage(ctx, message)
	if err != nil {
		return err
	}
	select {
	case <-message.Reply:
	case <-time.After(5 * time.Second):
		log.Warn(ctx, "Shutdown message timed out", "Draft Id", draftId)
	}

	actorMap.actorMap.Delete(draftId)
	log.Info(ctx, "Evicted draft actor from map", "Draft Id", draftId)
	return nil
}

func InvitePlayer(ctx context.Context, draftActor *DraftActor, invite model.DraftInvite) error {
	replyChan := make(chan Result)
	message := Message{
		Content: InvitePlayerMessage{
			Invite: invite,
		},
		Reply: replyChan,
	}
	err := draftActor.PostMessage(ctx, message)
	if err != nil {
		return err
	}
	select {
	case result := <-message.Reply:
		return result.Error
	case <-time.After(5 * time.Second):
		return errors.New("timeout inviting player")
	}
}

func AcceptInvite(ctx context.Context, draftActor *DraftActor, inviteId int, acceptingUserUuid uuid.UUID) error {
	replyChan := make(chan Result)
	message := Message{
		Content: AcceptInviteMessage{
			InviteId:          inviteId,
			AcceptingUserUuid: acceptingUserUuid,
		},
		Reply: replyChan,
	}
	err := draftActor.PostMessage(ctx, message)
	if err != nil {
		return err
	}
	select {
	case result := <-message.Reply:
		return result.Error
	case <-time.After(5 * time.Second):
		return errors.New("timeout accepting invite")
	}
}

func RegisterWatcher(ctx context.Context, actorMap *DraftActorMap, draftId int) *picking.Watcher {
	if actorMap.pickNotifier == nil {
		log.Warn(ctx, "PickNotifier is nil, cannot register watcher")
		return nil
	}
	return actorMap.pickNotifier.RegisterWatcher(draftId)
}

func UnregisterWatcher(ctx context.Context, actorMap *DraftActorMap, watcher *picking.Watcher) {
	if actorMap.pickNotifier == nil || watcher == nil {
		return
	}
	actorMap.pickNotifier.UnregisterWatcher(ctx, watcher)
}
