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
	mu            sync.Mutex
	runningDrafts map[int]bool
	draftActorMap *draft.DraftActorMap
}

func NewDraftDaemon(draftStore model.DraftStore, draftActorMap *draft.DraftActorMap) *DraftDaemon {
	return &DraftDaemon{
		draftStore:    draftStore,
		running:       false,
		runningDrafts: make(map[int]bool),
		draftActorMap: draftActorMap,
	}
}

func (d *DraftDaemon) Start() error {
	log.Info(context.TODO(), "Attempting to start draft daemon")
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.running {
		d.running = true
		log.Info(context.TODO(), "Started draft daemon")
		go d.Run()
		return nil
	} else {
		return errors.New("daemon already started")
	}
}

func (d *DraftDaemon) Run() {
	for d.running {
		//Get current picks for the running drafts
		log.DebugNoContext("Starting iteration of the Draft Daemon")
		err := d.checkForDraftsToStart()
		if err != nil {
			log.Error(context.TODO(), "Failed to start draft", "Error", err)
		}
		d.checkForPicksToSkip()

		time.Sleep(1 * time.Minute)
	}
}

func (d *DraftDaemon) checkForDraftsToStart() error {
	//Get all drafts that are in the waiting to start status
	log.Debug(context.Background(), "Checking for drafts to Start")
	now := time.Now()
	draftIds, err := d.draftStore.GetDraftsToStart(context.TODO(), now)
	if err != nil && draftIds == nil {
		return err
	}

	if len(draftIds) > 0 {
		log.Debug(context.Background(), "Found drafts to start")
	} else {
		log.DebugNoContext("Found no drafts to start")
	}

	if err != nil {
		log.Error(context.Background(), "Failed to get drafts to start", "Now", now, "Error", err)
	}

	for _, draftId := range draftIds {
		assert := assert.CreateAssertWithContext("Check For Drafts To Start")
		assert.AddContext("Draft Id", draftId)
		draftActor, err := d.draftActorMap.GetActor(context.TODO(), draftId)
		if err != nil {
			log.Warn(context.TODO(), "Failed to get draft acotr", "Draft Id", draftId, "Error", err)
			continue
		}
		draftState := draftActor.GetDraftState()
		assert.AddContext("Draft Status", draftState.Status)
		assert.RunAssert(context.TODO(), draftState.Status == model.WAITING_TO_START, "Invalid draft status to transition to picking")

		message := draft.Message {
			Content: draft.StateTransitionMessage{
				RequestedState: model.PICKING,
			},
		}
		err = draftActor.PostMessage(context.TODO(), message)
		if err != nil {
			log.Error(context.TODO(), "Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		}
	}

	return nil
}

func (d *DraftDaemon) checkForPicksToSkip() {
	for draftId, running := range d.runningDrafts {
		if !running {
			continue
		}

		draftActor, err := d.draftActorMap.GetActor(context.TODO(), draftId)
		if err != nil {
			log.Warn(context.TODO(), "Failed to get draft actor", "Draft Id", draftId, "Error", err)
			continue
		}
		draftState := draftActor.GetDraftState()
		skipped := false

		//Check if the current player if skipping their pick. If so we
		//should skip them
		log.DebugNoContext("Checking if player wants to be skipped", "Draft Id", draftId, "Current Pick Player", draftState.CurrentPick.Player)
		shouldSkip, err := d.draftStore.ShouldSkipPick(context.TODO(), draftState.CurrentPick.Player)
		if err != nil {
			log.Warn(context.TODO(), "Failed to check if player should be skipped", "Draft Id", draftId, "Player", draftState.CurrentPick.Player, "Error", err)
			shouldSkip = false
		}
		if shouldSkip {
			log.DebugNoContext("Skipping player", "Pick Id", draftState.CurrentPick.Id, "Player", draftState.CurrentPick.Player)
			skipped = draft.SkipCurrentPick(context.TODO(), draftActor, draftId, draftState.CurrentPick.Id)
		}

		log.DebugNoContext("Checking expiration time", "Draft Id", draftId, "Current Pick Player", draftState.CurrentPick.Player)
		now := time.Now()
		if draftState.CurrentPick.ExpirationTime.Before(now) && !skipped {
			log.DebugNoContext("Pick expired", "Pick Id", draftState.CurrentPick.Id, "Expiration Time", draftState.CurrentPick.ExpirationTime, "Now", now)
			skipped = draft.SkipCurrentPick(context.TODO(), draftActor, draftId, draftState.CurrentPick.Id)
		} else {
			log.DebugNoContext("Pick is not expired yet", "Pick Id", draftState.CurrentPick.Id, "Expiration Time", draftState.CurrentPick.ExpirationTime, "Now", now)
		}
	}
}

func (d *DraftDaemon) AddDraft(draftId int) error {
	if d.runningDrafts[draftId] {
		return errors.New("draft already added to daemon")
	}
	d.runningDrafts[draftId] = true
	log.Info(context.TODO(), "Added draft to daemon", "Draft Id", draftId)
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
	log.Info(context.TODO(), "Stopped draft daemon")
	d.running = false
	return nil
}
