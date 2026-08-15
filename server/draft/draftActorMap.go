package draft

import (
	"context"
	"server/discord"
	"server/log"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"server/utils"
	"sync"
	"time"
)

const defaultDraftActorCacheSize = 128

// DraftActorMap holds the active draft actors in an LRU cache so memory usage
// stays bounded. When an actor is evicted it is sent a shutdown message so its
// goroutine can exit cleanly.
type DraftActorMap struct {
	actorCache        *lruCache[int, *DraftActor]
	loadLocks         sync.Map
	draftStore        model.DraftStore
	tbaHandler        tbaHandler.TBAInterface
	discordStore      model.DiscordStore
	discordWebhookBus discord.DiscordNotifier
	pickNotifier      *picking.PickNotifier
	pickConfig        utils.PickWindowConfig
}

func NewDraftActorMap(draftStore model.DraftStore, tbaHandler tbaHandler.TBAInterface, discordStore model.DiscordStore, discordWebhookBus discord.DiscordNotifier, pickNotifier *picking.PickNotifier, pickConfig utils.PickWindowConfig, cacheSize int) *DraftActorMap {
	if cacheSize < 1 {
		cacheSize = defaultDraftActorCacheSize
	}

	actorMap := &DraftActorMap{
		draftStore:        draftStore,
		tbaHandler:        tbaHandler,
		discordStore:      discordStore,
		discordWebhookBus: discordWebhookBus,
		pickNotifier:      pickNotifier,
		pickConfig:        pickConfig,
	}

	actorMap.actorCache = newLRUCache[int, *DraftActor](cacheSize, actorMap.onEvicted)
	return actorMap
}

func (d *DraftActorMap) onEvicted(draftId int, actor *DraftActor) {
	// If the actor was already shut down (e.g., via ShutdownActor), there is no
	// need to send another shutdown message.
	if actor.IsShutdown() {
		return
	}
	go func() {
		ctx := context.Background()
		reply := make(chan Result)
		msg := Message{
			Content: ShutdownMessage{},
			context: ctx,
			Reply:   reply,
		}
		if err := actor.PostMessage(ctx, msg); err != nil {
			log.Error(ctx, "failed to send shutdown to evicted draft actor", "draftId", draftId, "error", err)
			return
		}
		select {
		case <-reply:
			log.Info(ctx, "evicted draft actor shut down", "draftId", draftId)
		case <-time.After(5 * time.Second):
			log.Warn(ctx, "evicted draft actor shutdown timed out", "draftId", draftId)
		}
	}()
}

func (d *DraftActorMap) GetActor(ctx context.Context, draftId int) (*DraftActor, error) {
	if actor, ok := d.actorCache.Get(draftId); ok {
		return actor, nil
	}

	lock := d.getLoadLock(draftId)
	lock.Lock()
	defer lock.Unlock()

	if actor, ok := d.actorCache.Get(draftId); ok {
		return actor, nil
	}

	newActor, err := NewDraftActor(ctx, draftId, d.draftStore, d.tbaHandler, d.discordStore, d.discordWebhookBus, d.pickNotifier, d.pickConfig)
	if err != nil {
		return nil, err
	}

	d.actorCache.Add(draftId, newActor)
	d.loadLocks.Delete(draftId)
	return newActor, nil
}

func (d *DraftActorMap) getLoadLock(draftId int) *sync.Mutex {
	// LoadOrStore guarantees only one mutex is created per draftId, even if
	// many goroutines race to create it.
	mtx := &sync.Mutex{}
	if actual, loaded := d.loadLocks.LoadOrStore(draftId, mtx); loaded {
		return actual.(*sync.Mutex)
	}
	return mtx
}
