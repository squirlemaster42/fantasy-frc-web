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

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var draftDaemonTracer = otel.Tracer("draft-daemon")

type DraftDaemon struct {
	draftStore    model.DraftStore
	running       bool
	mu            sync.Mutex
	runningDrafts map[int]bool
	draftManager  *draft.DraftManager
	serverCtx     context.Context
}

func NewDraftDaemon(ctx context.Context, draftStore model.DraftStore, draftManager *draft.DraftManager) *DraftDaemon {
	return &DraftDaemon{
		draftStore:    draftStore,
		running:       false,
		runningDrafts: make(map[int]bool),
		draftManager:  draftManager,
		serverCtx:     ctx,
	}
}

func (d *DraftDaemon) Start() error {
	log.Info(d.serverCtx, "Attempting to start draft daemon")
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.running {
		d.running = true
		log.Info(d.serverCtx, "Started draft daemon")
		go d.Run()
		return nil
	} else {
		return errors.New("daemon already started")
	}
}

func (d *DraftDaemon) Run() {
	for d.running {
		select {
		case <-d.serverCtx.Done():
			log.Info(d.serverCtx, "Draft daemon shutting down")
			return
		default:
		}
		//Get current picks for the running drafts
		log.DebugNoContext("Starting iteration of the Draft Daemon")
		ctx, span := draftDaemonTracer.Start(d.serverCtx, "draft-daemon.iteration")
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := d.checkForDraftsToStart(ctx)
		if err != nil {
			log.Error(ctx, "Failed to start draft", "Error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		d.checkForPicksToSkip(ctx)
		cancel()
		span.End()

		select {
		case <-d.serverCtx.Done():
			return
		case <-time.After(1 * time.Minute):
		}
	}
}

func (d *DraftDaemon) checkForDraftsToStart(ctx context.Context) error {
	ctx, span := draftDaemonTracer.Start(ctx, "draft-daemon.check-drafts-to-start")
	defer span.End()

	//Get all drafts that are in the waiting to start status
	log.DebugNoContext("Checking for drafts to Start")
	now := time.Now()
	draftIds, err := d.draftStore.GetDraftsToStart(ctx, now)
	if err != nil && draftIds == nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	if len(draftIds) > 0 {
		log.DebugNoContext("Found drafts to start")
	} else {
		log.DebugNoContext("Found no drafts to start")
	}

	if err != nil {
		log.Error(ctx, "Failed to get drafts to start", "Now", now, "Error", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	for _, draftId := range draftIds {
		assert := assert.CreateAssertWithContext("Check For Drafts To Start")
		assert.AddContext("Draft Id", draftId)
		draft, err := d.draftManager.GetDraft(ctx, draftId, false)
		if err != nil {
			log.Warn(ctx, "Failed to load draft", "Draft Id", draftId, "Error", err)
			span.RecordError(err)
			continue
		}
		assert.AddContext("Draft Status", draft.GetStatus())
		assert.RunAssert(ctx, draft.GetStatus() == model.WAITING_TO_START, "Invalid draft status to transition to picking")
		err = d.draftManager.ExecuteDraftStateTransition(ctx, draftId, model.PICKING)
		if err != nil {
			log.Error(ctx, "Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}

	return nil
}

func (d *DraftDaemon) checkForPicksToSkip(ctx context.Context) {
	ctx, span := draftDaemonTracer.Start(ctx, "draft-daemon.check-picks-to-skip")
	defer span.End()

	for draftId, running := range d.runningDrafts {
		if !running {
			continue
		}

		curPick, err := d.draftManager.GetCurrentPick(draftId)
		if err != nil {
			span.RecordError(err)
			continue
		}
		skipped := false

		//Check if the current player if skipping their pick. If so we
		//should skip them
		log.DebugNoContext("Checking if player wants to be skipped", "Draft Id", draftId, "Current Pick Player", curPick.Player)
		shouldSkip, err := d.draftStore.ShouldSkipPick(ctx, curPick.Player)
		if err != nil {
			log.Warn(ctx, "Failed to check if player should be skipped", "Draft Id", draftId, "Player", curPick.Player, "Error", err)
			span.RecordError(err)
			shouldSkip = false
		}
		if shouldSkip {
			log.DebugNoContext("Skipping player", "Pick Id", curPick.Id, "Player", curPick.Player)
			err := d.draftManager.SkipCurrentPick(draftId)
			skipped = err == nil
			if err != nil {
				span.RecordError(err)
			}
		}

		log.DebugNoContext("Checking expiration time", "Draft Id", draftId, "Current Pick Player", curPick.Player)
		now := time.Now()
		if curPick.ExpirationTime.Before(now) && !skipped {
			log.DebugNoContext("Pick expired", "Pick Id", curPick.Id, "Expiration Time", curPick.ExpirationTime, "Now", now)
			//We want to skip the current pick and go to the next one
			err := d.draftManager.SkipCurrentPick(draftId)
			if err != nil {
				log.Warn(ctx, "Failed to skip pick", "Draft Id", draftId, "Current Pick", curPick.Id, "Error", err)
				span.RecordError(err)
			}
		} else {
			log.DebugNoContext("Pick is not expired yet", "Pick Id", curPick.Id, "Expiration Time", curPick.ExpirationTime, "Now", now)
		}
	}
}

func (d *DraftDaemon) AddDraft(draftId int) error {
	if d.runningDrafts[draftId] {
		return errors.New("draft already added to daemon")
	}
	d.runningDrafts[draftId] = true
	log.Info(d.serverCtx, "Added draft to daemon", "Draft Id", draftId)
	return nil
}

func (d *DraftDaemon) RemoveDraft(draftId int) error {
	if !d.runningDrafts[draftId] {
		return errors.New("draft not in daemon")
	}
	d.runningDrafts[draftId] = false
	return nil
}

func (d *DraftDaemon) IsRunning() bool {
	return d.running
}

func (d *DraftDaemon) Stop() error {
	log.Info(d.serverCtx, "Stopped draft daemon")
	d.running = false
	return nil
}
