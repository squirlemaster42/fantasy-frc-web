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

		if !d.IsRunning() {
			return
		}

		// Create a per-tick context with timeout so one slow operation doesn't block the daemon
		tickCtx, cancel := context.WithTimeout(ctx, 55*time.Second)

		log.Debug(tickCtx, "Starting iteration of the Draft Daemon")
		// err := d.checkForDraftsToStart(tickCtx)
		// if err != nil {
		//     log.Error(tickCtx, "Failed to check for drafts to start", "Error", err)
		// }
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
			log.Warn(ctx, "Failed to get draft actor", "Draft Id", draftId, "Error", err)
			continue
		}
		draftState := draftActor.GetDraftState()

		skipped := false

		log.Debug(ctx, "Checking if player wants to be skipped", "Draft Id", draftId, "Current Pick Player", draftState.CurrentPick.Player)
		shouldSkip, err := d.draftStore.ShouldSkipPick(ctx, draftState.CurrentPick.Player)
		if err != nil {
			log.Warn(ctx, "Failed to check if player should be skipped", "Draft Id", draftId, "Player", draftState.CurrentPick.Player, "Error", err)
			shouldSkip = false
		}
		if shouldSkip {
			log.Debug(ctx, "Skipping player", "Pick Id", draftState.CurrentPick.Id, "Player", draftState.CurrentPick.Player)
			skipped = draft.SkipCurrentPick(ctx, draftActor, draftId, draftState.CurrentPick.Id)
		}

		log.Debug(ctx, "Checking expiration time", "Draft Id", draftId, "Current Pick Player", draftState.CurrentPick.Player)
		now := time.Now().UTC()
		if draftState.CurrentPick.ExpirationTime.Before(now) && !skipped {
			log.Debug(ctx, "Pick expired", "Pick Id", draftState.CurrentPick.Id, "Expiration Time", draftState.CurrentPick.ExpirationTime, "Now", now)
			draft.SkipCurrentPick(ctx, draftActor, draftId, draftState.CurrentPick.Id)
		} else {
			log.Debug(ctx, "Pick is not expired yet", "Pick Id", draftState.CurrentPick.Id, "Expiration Time", draftState.CurrentPick.ExpirationTime, "Now", now)
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
	log.Info(ctx, "Added draft to daemon", "Draft Id", draftId)
	return nil
}

func (d *DraftDaemon) RemoveDraft(ctx context.Context, draftId int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.runningDrafts[draftId] {
		return errors.New("draft not in daemon")
	}
	d.runningDrafts[draftId] = false
	log.Info(ctx, "Removed draft from daemon", "Draft Id", draftId)
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
