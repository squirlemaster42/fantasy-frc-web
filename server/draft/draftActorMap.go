package draft

import (
	"context"
	"server/discord"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"sync"
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

func NewDraftActorMap(draftStore model.DraftStore, tbaHandler *tbaHandler.TbaHandler, discordStore model.DiscordStore, discordWebhookBus *discord.DiscordWebhookBus) (*DraftActorMap) {
	return &DraftActorMap{
		draftStore: draftStore,
		tbaHandler: tbaHandler,
		discordStore: discordStore,
		discordWebhookBus: discordWebhookBus,
		// TODO how do we new up this? pickNotifier: pickNotifier,
	}
}

func (d *DraftActorMap) GetActor(ctx context.Context, draftId int) (*DraftActor, error) {
	actor, ok := d.actorMap.Load(draftId)
	if !ok {
		d.getLoadLock(draftId).Lock()
		defer d.getLoadLock(draftId).Unlock()

		actor, ok := d.actorMap.Load(draftId)
		if ok {
			// Actor was loaded by another process before we got the lock
			// We dont want to load it again
			return actor.(*DraftActor), nil
		}

		actor, err := NewDraftActor(ctx, draftId, d.draftStore, d.tbaHandler, d.discordStore, d.discordWebhookBus, d.pickNotifier)
		if err != nil {
			return nil, err
		}
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
