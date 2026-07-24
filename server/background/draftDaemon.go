package background

import (
	"context"
	"errors"
	"server/assert"
	"server/draft"
	"server/log"
	"server/model"
	"sync"
	"time"
)

type DraftDaemon struct {
	draftStore    model.DraftStore
	running       bool
	mu            sync.RWMutex
	cancel        context.CancelFunc
	runningDrafts map[int]bool
	draftActorMap *draft.DraftActorMap
}

func NewDraftDaemon(draftStore model.DraftStore, draftActorMap *draft.DraftActorMap) *DraftDaemon {
	assert.AssertCF(context.Background(), draftActorMap != nil, "DraftActorMap cannot be nil")
	return &DraftDaemon{
		draftStore:    draftStore,
		running:       false,
		runningDrafts: make(map[int]bool),
		draftActorMap: draftActorMap,
	}
}

func (d *DraftDaemon) Start(ctx context.Context) error {
	log.Info(ctx, "Attempting to start draft daemon")
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return errors.New("daemon already started")
	}
	d.running = true
	// Create a daemon context that preserves trace values but isn't tied to parent lifecycle
	daemonCtx := context.WithoutCancel(ctx)
	daemonCtx, d.cancel = context.WithCancel(daemonCtx)
	d.mu.Unlock()

	log.Info(daemonCtx, "Started draft daemon")
	go d.Run(daemonCtx)
	return nil
}

func (d *DraftDaemon) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Info(ctx, "Draft daemon shutting down")
			return
		default:
		}

		d.mu.RLock()
		defer d.mu.RUnlock()
		if !d.IsRunning() {
			return
		}

		// Create a per-tick context with timeout so one slow operation doesn't block the daemon
		tickCtx, cancel := context.WithTimeout(ctx, 55*time.Second)

		log.Debug(tickCtx, "Starting iteration of the Draft Daemon")
		d.checkForPicksToSkip(tickCtx)

		cancel()
		time.Sleep(1 * time.Minute)
	}
}

func (d *DraftDaemon) checkForPicksToSkip(ctx context.Context) {
	d.mu.RLock()
	runningDraftsCopy := make(map[int]bool, len(d.runningDrafts))
	for k, v := range d.runningDrafts {
		runningDraftsCopy[k] = v
	}
	d.mu.RUnlock()

	for draftId, running := range runningDraftsCopy {
		if !running {
			continue
		}

		draftActor, err := d.draftActorMap.GetActor(ctx, draftId)
		if err != nil {
			log.Error(ctx, "Failed to get draft actor", "draftId", draftId, "error", err)
			continue
		}
		draftState := draftActor.GetDraftState()

		skipped := false

		log.Debug(ctx, "Checking if player wants to be skipped", "draftId", draftId, "currentPickPlayer", draftState.CurrentPick.Player)
		shouldSkip, err := d.draftStore.ShouldSkipPick(ctx, draftState.CurrentPick.Player)
		if err != nil {
			log.Error(ctx, "Failed to check if player should be skipped", "draftId", draftId, "player", draftState.CurrentPick.Player, "error", err)
			shouldSkip = false
		}
		if shouldSkip {
			log.Debug(ctx, "Skipping player", "pickId", draftState.CurrentPick.Id, "player", draftState.CurrentPick.Player)
			skipped = draft.SkipCurrentPick(ctx, draftActor, draftId, draftState.CurrentPick.Id)
		}

		log.Debug(ctx, "Checking expiration time", "draftId", draftId, "currentPickPlayer", draftState.CurrentPick.Player)
		now := time.Now()
		if draftState.CurrentPick.ExpirationTime.Before(now) && !skipped {
			log.Debug(ctx, "Pick expired", "pickId", draftState.CurrentPick.Id, "expirationTime", draftState.CurrentPick.ExpirationTime, "now", now)
			draft.SkipCurrentPick(ctx, draftActor, draftId, draftState.CurrentPick.Id)
		} else {
			log.Debug(ctx, "Pick is not expired yet", "pickId", draftState.CurrentPick.Id, "expirationTime", draftState.CurrentPick.ExpirationTime, "now", now)
		}
	}
}

func (d *DraftDaemon) AddDraft(ctx context.Context, draftId int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.runningDrafts[draftId] {
		return errors.New("draft already added to daemon")
	}
	d.runningDrafts[draftId] = true
	log.Info(ctx, "Added draft to daemon", "draftId", draftId)
	return nil
}

func (d *DraftDaemon) RemoveDraft(ctx context.Context, draftId int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.runningDrafts[draftId] {
		return errors.New("draft not in daemon")
	}
	d.runningDrafts[draftId] = false
	log.Info(ctx, "Removed draft from daemon", "draftId", draftId)
	return nil
}

func (d *DraftDaemon) IsRunning() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.running
}

func (d *DraftDaemon) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.running {
		return errors.New("daemon not running")
	}
	log.Info(ctx, "Stopped draft daemon")
	d.running = false
	if d.cancel != nil {
		d.cancel()
	}
	return nil
}
