package draft

import (
	"context"
	"errors"
	"time"

	"uuid"

	"server/log"
	"server/model"
	"server/picking"
)

// postAndWait sends a message to the draft actor and returns the result or a
// timeout error. It is the common backbone for all actor request helpers.
func postAndWait(ctx context.Context, draftActor *DraftActor, content any) (Result, error) {
	replyChan := make(chan Result)
	message := Message{
		Content: content,
		Reply:   replyChan,
	}
	if err := draftActor.PostMessage(ctx, message); err != nil {
		return Result{}, err
	}
	select {
	case result := <-message.Reply:
		return result, nil
	case <-time.After(DraftActorRequestTimeout()):
		return Result{}, errors.New("timeout waiting for draft actor response")
	}
}

func SkipCurrentPick(ctx context.Context, draftActor *DraftActor, draftId int, currentPickId int) bool {
	result, err := postAndWait(ctx, draftActor, SkipCurrentPickMessage{CurrentPickId: currentPickId})
	if err != nil {
		log.Error(ctx, "Failed to post skip message to draft actor", "draftId", draftId, "error", err)
		return false
	}
	if result.Error != nil {
		log.Error(ctx, "Skipping current pick in draft failed", "draftId", draftId, "currentPickId", draftActor.GetDraftState().CurrentPick.Id, "error", result.Error)
		return false
	}
	skipped, ok := result.Value.(bool)
	if !ok {
		log.Error(ctx, "Skip current pick returned non-bool value", "draftId", draftId, "currentPickId", draftActor.GetDraftState().CurrentPick.Id)
		return false
	}
	if !skipped {
		log.Warn(ctx, "Skipping current pick in draft returned false", "draftId", draftId, "currentPickId", draftActor.GetDraftState().CurrentPick.Id)
	}
	return skipped
}

func EndDraft(ctx context.Context, draftActor *DraftActor, draftId int) bool {
	result, err := postAndWait(ctx, draftActor, EndDraftMessage{})
	if err != nil {
		log.Error(ctx, "Failed to post end draft message to draft actor", "draftId", draftId, "error", err)
		return false
	}
	if result.Error != nil {
		log.Error(ctx, "Ending draft failed", "draftId", draftId, "error", result.Error)
		return false
	}
	ended, ok := result.Value.(bool)
	if !ok {
		log.Error(ctx, "End draft returned non-bool value", "draftId", draftId)
		return false
	}
	if !ended {
		log.Warn(ctx, "Ending draft returned false", "draftId", draftId)
	}
	return ended
}

func ModifyCurrentPickExpirationTime(ctx context.Context, draftActor *DraftActor, extension time.Duration) error {
	result, err := postAndWait(ctx, draftActor, ModifyExpirationTimeMessage{
		PickId:    draftActor.GetDraftState().CurrentPick.Id,
		Extension: extension,
	})
	if err != nil {
		return err
	}
	return result.Error
}

func GetCurrentPick(draftActor *DraftActor) model.Pick {
	return draftActor.GetDraftState().CurrentPick
}

func MakePick(ctx context.Context, draftActor *DraftActor, pick model.Pick) error {
	result, err := postAndWait(ctx, draftActor, PickMessage{Pick: pick})
	if err != nil {
		return err
	}
	return result.Error
}

func UndoLastPick(ctx context.Context, draftActor *DraftActor) error {
	result, err := postAndWait(ctx, draftActor, UndoLastPickMessage{
		CurrentPickId: draftActor.GetDraftState().CurrentPick.Id,
	})
	if err != nil {
		return err
	}
	return result.Error
}

func GetDraft(draftActor *DraftActor) model.DraftModel {
	return draftActor.GetDraftState()
}

func UpdateDraft(ctx context.Context, draftActor *DraftActor, draftModel model.DraftModel) error {
	result, err := postAndWait(ctx, draftActor, UpdateDraftProfileMessage{
		Name:               draftModel.DisplayName,
		Description:        draftModel.Description,
		DiscordWebhook:     draftModel.DiscordWebhook,
		TimingType:         draftModel.TimingType,
		InitialPickTimeSec: draftModel.InitialPickTimeSec,
		IncrementTimeSec:   draftModel.IncrementTimeSec,
		PerPickExpTimeSec:  draftModel.PerPickExpTimeSec,
	})
	if err != nil {
		return err
	}
	return result.Error
}

func ExecuteDraftStateTransition(ctx context.Context, draftActor *DraftActor, requestedState model.DraftState) error {
	result, err := postAndWait(ctx, draftActor, StateTransitionMessage{RequestedState: requestedState})
	if err != nil {
		return err
	}
	return result.Error
}

func ShutdownActor(actorMap *DraftActorMap, ctx context.Context, draftId int) error {
	actor, ok := actorMap.actorCache.Get(draftId)
	if !ok {
		return nil
	}

	_, err := postAndWait(ctx, actor, ShutdownMessage{})
	if err != nil {
		log.Warn(ctx, "Shutdown message timed out", "draftId", draftId)
	}

	actorMap.actorCache.Remove(draftId)
	log.Info(ctx, "Evicted draft actor from map", "draftId", draftId)
	return nil
}

func InvitePlayer(ctx context.Context, draftActor *DraftActor, invite model.DraftInvite) error {
	result, err := postAndWait(ctx, draftActor, InvitePlayerMessage{Invite: invite})
	if err != nil {
		return err
	}
	return result.Error
}

func UninvitePlayer(ctx context.Context, draftActor *DraftActor, draftId int, ownerUuid uuid.UUID, inviteId int) error {
	result, err := postAndWait(ctx, draftActor, UninvitePlayerMessage{
		DraftId:   draftId,
		OwnerUuid: ownerUuid,
		InviteId:  inviteId,
	})
	if err != nil {
		return err
	}
	return result.Error
}

func DeclineInvite(ctx context.Context, draftActor *DraftActor, inviteId int, userUuid uuid.UUID) error {
	result, err := postAndWait(ctx, draftActor, DeclineInviteMessage{
		InviteId: inviteId,
		UserUuid: userUuid,
	})
	if err != nil {
		return err
	}
	return result.Error
}

func AcceptInvite(ctx context.Context, draftActor *DraftActor, inviteId int, acceptingUserUuid uuid.UUID) error {
	result, err := postAndWait(ctx, draftActor, AcceptInviteMessage{
		InviteId:          inviteId,
		AcceptingUserUuid: acceptingUserUuid,
	})
	if err != nil {
		return err
	}
	return result.Error
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
