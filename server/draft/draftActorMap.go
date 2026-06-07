package draft

import (
	"context"
	"errors"
	"server/discord"
	"server/log"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"sync"
	"time"
)

// TODO should we LRU this?
type DraftActorMap struct {
	actorMap sync.Map
	loadLocks sync.Map
	draftStore model.DraftStore
	tbaHandler *tbaHandler.TbaHandler
	discordStore model.DiscordStore
	discordWebhookBus *discord.DiscordWebhookBus
	pickNotifier *picking.PickNotifier
}

func NewDraftActorMap(draftStore model.DraftStore, tbaHandler *tbaHandler.TbaHandler, discordStore model.DiscordStore, discordWebhookBus *discord.DiscordWebhookBus, pickNotifier *picking.PickNotifier) *DraftActorMap {
	return &DraftActorMap{
		draftStore: draftStore,
		tbaHandler: tbaHandler,
		discordStore: discordStore,
		discordWebhookBus: discordWebhookBus,
		pickNotifier: pickNotifier,
	}
}

func (d *DraftActorMap) GetActor(ctx context.Context, draftId int) (*DraftActor, error) {
	actor, ok := d.actorMap.Load(draftId)
	if !ok {
		d.getLoadLock(draftId).Lock()
		defer d.getLoadLock(draftId).Unlock()

		actor, ok = d.actorMap.Load(draftId)
		if ok {
			// Actor was loaded by another process before we got the lock
			// We dont want to load it again
			return actor.(*DraftActor), nil
		}

		newActor, err := NewDraftActor(ctx, draftId, d.draftStore, d.tbaHandler, d.discordStore, d.discordWebhookBus, d.pickNotifier)
		if err != nil {
			return nil, err
		}
		d.actorMap.Store(draftId, newActor)
		return newActor, nil
	}
	return actor.(*DraftActor), nil
}

func (d *DraftActorMap) getLoadLock(draftId int) *sync.Mutex {
	//Get the lock if it exists for the draft, if not register it
	lock, ok := d.loadLocks.Load(draftId)
	if !ok {
		mtx := &sync.Mutex{}
		d.loadLocks.Store(draftId, mtx)
		return mtx
	}
	return lock.(*sync.Mutex)
}

func (d *DraftActorMap) SkipCurrentPick(ctx context.Context, draftId int) error {
	draftActor, err := d.GetActor(ctx, draftId)
	if err != nil {
		return err
	}

	replyChan := make(chan Result)
	message := Message{
		Content: SkipCurrentPickMessage{
			CurrentPickId: draftActor.GetDraftState().CurrentPick.Id,
		},
		Reply: replyChan,
	}
	err = draftActor.PostMessage(ctx, message)
	if err != nil {
		return err
	}
	select {
	case result := <-message.Reply:
		return result.Error
	case <-time.After(5 * time.Second):
		return errors.New("timeout skipping current pick")
	}
}

func (d *DraftActorMap) ModifyCurrentPickExpirationTime(ctx context.Context, draftId int, extension time.Duration) error {
	draftActor, err := d.GetActor(ctx, draftId)
	if err != nil {
		return err
	}

	replyChan := make(chan Result)
	message := Message{
		Content: ModifyExpirationTimeMessage{
			PickId:    draftActor.GetDraftState().CurrentPick.Id,
			Extension: extension,
		},
		Reply: replyChan,
	}
	err = draftActor.PostMessage(ctx, message)
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

func (d *DraftActorMap) GetCurrentPick(ctx context.Context, draftId int) (model.Pick, error) {
	draftActor, err := d.GetActor(ctx, draftId)
	if err != nil {
		return model.Pick{}, err
	}
	return draftActor.GetDraftState().CurrentPick, nil
}

func (d *DraftActorMap) MakePick(ctx context.Context, draftId int, pick model.Pick) error {
	draftActor, err := d.GetActor(ctx, draftId)
	if err != nil {
		return err
	}

	replyChan := make(chan Result)
	message := Message{
		Content: PickMessage{
			Pick: pick,
		},
		Reply: replyChan,
	}
	err = draftActor.PostMessage(ctx, message)
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

func (d *DraftActorMap) UndoLastPick(ctx context.Context, draftId int) error {
	draftActor, err := d.GetActor(ctx, draftId)
	if err != nil {
		return err
	}

	replyChan := make(chan Result)
	message := Message{
		Content: UndoLastPickMessage{
			CurrentPickId: draftActor.GetDraftState().CurrentPick.Id,
		},
		Reply: replyChan,
	}
	err = draftActor.PostMessage(ctx, message)
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

func (d *DraftActorMap) GetDraft(ctx context.Context, draftId int) (model.DraftModel, error) {
	draftActor, err := d.GetActor(ctx, draftId)
	if err != nil {
		return model.DraftModel{}, err
	}
	return draftActor.GetDraftState(), nil
}

func (d *DraftActorMap) UpdateDraft(ctx context.Context, draftModel model.DraftModel) error {
	draftActor, err := d.GetActor(ctx, draftModel.Id)
	if err != nil {
		return err
	}

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
	err = draftActor.PostMessage(ctx, message)
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

func (d *DraftActorMap) ExecuteDraftStateTransition(ctx context.Context, draftId int, requestedState model.DraftState) error {
	draftActor, err := d.GetActor(ctx, draftId)
	if err != nil {
		return err
	}

	replyChan := make(chan Result)
	message := Message{
		Content: StateTransitionMessage{
			RequestedState: requestedState,
		},
		Reply: replyChan,
	}
	err = draftActor.PostMessage(ctx, message)
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

	if requestedState == model.COMPLETE {
		log.Info(ctx, "Draft transitioning to COMPLETE, evicting actor", "Draft Id", draftId)
		if err := d.ShutdownActor(ctx, draftId); err != nil {
			log.Warn(ctx, "Failed to shutdown draft actor after completion", "Draft Id", draftId, "Error", err)
		}
	}
	return nil
}

func (d *DraftActorMap) ShutdownActor(ctx context.Context, draftId int) error {
	actor, ok := d.actorMap.Load(draftId)
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

	d.actorMap.Delete(draftId)
	log.Info(ctx, "Evicted draft actor from map", "Draft Id", draftId)
	return nil
}

func (d *DraftActorMap) RegisterWatcher(draftId int) *picking.Watcher {
	if d.pickNotifier == nil {
		log.Warn(context.TODO(), "PickNotifier is nil, cannot register watcher")
		return nil
	}
	return d.pickNotifier.RegisterWatcher(draftId)
}

func (d *DraftActorMap) UnregisterWatcher(watcher *picking.Watcher) {
	if d.pickNotifier == nil || watcher == nil {
		return
	}
	d.pickNotifier.UnregisterWatcher(watcher)
}
