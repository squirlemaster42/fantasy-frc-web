package background

import (
	"database/sql"
	"errors"
	"log/slog"
	"server/assert"
	"server/draft"
	"server/model"
	"sync"
	"time"
)

type DraftDaemon struct {
    database *sql.DB
    running bool
    mu sync.Mutex
    runningDrafts map[int]bool
    draftManager *draft.DraftManager
}

func NewDraftDaemon(database *sql.DB, draftManager *draft.DraftManager) *DraftDaemon {
    return &DraftDaemon{
        database: database,
        running: false,
        runningDrafts: make(map[int]bool),
        draftManager: draftManager,
    }
}

func (d *DraftDaemon) Start() error {
    slog.Info("Attempting to start draft daemon")
    d.mu.Lock()
    defer d.mu.Unlock()
    if !d.running {
        d.running = true
        slog.Info("Started draft daemon")
        go d.Run()
        return nil
    } else {
        return errors.New("daemon already started")
    }
}

func (d *DraftDaemon) Run() {
    for d.running {
        //Get current picks for the running drafts
        slog.Info("Starting iteration of the Draft Daemon")
		err := d.checkForDraftsToStart()
		if err != nil {
			slog.Error("Failde to start draft", "Error", err)
		}
        d.checkForPicksToSkip()

        time.Sleep(1 * time.Minute)
    }
}

func (d *DraftDaemon) checkForDraftsToStart() error {
    //Get all drafts that are in the waiting to start status
    slog.Info("Checking for drafts to Start")
    now := time.Now()
    draftIds, err := model.GetDraftsToStart(d.database, now)
    if err != nil && draftIds == nil {
        return err
    }

    if len(draftIds) > 0 {
        slog.Info("Found drafts to start")
    }

    if err != nil {
        slog.Error("Failed to get drafts to start", "Now", now, "Error", err)
    }

    for _, draftId := range draftIds {
		assert := assert.CreateAssertWithContext("Check For Drafts To Start")
		assert.AddContext("Draft Id", draftId)
        _, err := d.draftManager.GetDraft(draftId, false)
        if err != nil {
            slog.Warn("Failed to load draft", "Draft Id", draftId, "Error", err)
            continue
        }
		draft, err := d.draftManager.GetDraft(draftId, false)
		assert.NoError(err, "Failed to load draft")
		assert.AddContext("Draft Status", draft.GetStatus())
		assert.RunAssert(draft.GetStatus() == model.WAITING_TO_START, "Invalid draft status to transition to picking")
		err = d.draftManager.ExecuteDraftStateTransition(draftId, model.PICKING)
		if err != nil {
			slog.Error("Failed to execute draft state transition", "Draft Id", draftId, "Error", err)
		}
    }

    return nil
}

func (d *DraftDaemon) checkForPicksToSkip() {
    for draftId, running := range d.runningDrafts {
        if !running {
            continue
        }

        curPick := model.GetCurrentPick(d.database, draftId)
        skipped := false

        //Check if the current player if skipping their pick. If so we
        //should skip them
        slog.Info("Checking if player wants to be skipped", "Draft Id", draftId, "Current Pick Player", curPick.Player)
        shouldSkip := model.ShoudSkipPick(d.database, curPick.Player)
        if shouldSkip {
            slog.Info("Skipping player", "Pick Id", curPick.Id, "Player", curPick.Player)
			err := d.draftManager.SkipCurrentPick(draftId)
            skipped = err == nil
        }

        slog.Info("Checking expiration time", "Draft Id", draftId, "Current Pick Player", curPick.Player)
        now := time.Now()
        if curPick.ExpirationTime.Before(now) && !skipped {
            slog.Info("Pick expired", "Pick Id", curPick.Id, "Expiration Time", curPick.ExpirationTime, "Now", now)
            //We want to skip the current pick and go to the next one
			err := d.draftManager.SkipCurrentPick(draftId)
			if err != nil {
				slog.Warn("Failed to skip pick", "Draft Id", draftId, "Current Pick", curPick.Id, "Error", err)
			}
        } else {
            slog.Info("Pick is not expired yet", "Pick Id", curPick.Id, "Expiration Time", curPick.ExpirationTime, "Now", now)
        }
    }
}

func (d *DraftDaemon) AddDraft(draftId int) error {
    if d.runningDrafts[draftId] {
        return errors.New("draft already added to daemon")
    }
    d.runningDrafts[draftId] = true
    slog.Info("Added draft to daemon", "Draft Id", draftId)
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
    slog.Info("Stopped draft daemon")
    d.running = false
    return nil
}
