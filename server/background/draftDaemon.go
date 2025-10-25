package background

import (
	"database/sql"
	"errors"
	"log/slog"
	"server/model"
	"server/picking"
	"server/utils"
	"sync"
	"time"
)

type DraftDaemon struct {
    database *sql.DB
    running bool
    mu sync.Mutex
    runningDrafts map[int]bool
    notifier *picking.PickNotifier
}

func NewDraftDaemon(database *sql.DB) *DraftDaemon {
    return &DraftDaemon{
        database: database,
        running: false,
        runningDrafts: make(map[int]bool),
        notifier: &picking.PickNotifier{},
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
        d.checkForDraftsToStart()
        d.checkForPicksToSkip()

        time.Sleep(1 * time.Minute)
    }
}

func (d *DraftDaemon) checkForDraftsToStart() error {
    //Get all drafts that are in the waiting to start status
    now := time.Now()
    draftIds, err := model.GetDraftsToStart(d.database, now)
    if err != nil && draftIds == nil {
        return err
    }

    if err != nil {
        slog.Error("Failed to get drafts to start", "Now", now, "Error", err)
    }

    for _, draftId := range draftIds {
    }

    return nil
}

func (d *DraftDaemon) checkForPicksToSkip() {
    //TODO This needs to go through the draft manager, so we need to pass the
    //draft manager in here instead of what we are doing now
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
            nextPickPlayer := model.NextPick(d.database, draftId)
            model.SkipPick(d.database, curPick.Id)
            model.MakePickAvailable(d.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now()))
            d.notifier.NotifyWatchers(draftId)
            skipped = true
        }

        slog.Info("Checking expiration time", "Draft Id", draftId, "Current Pick Player", curPick.Player)
        now := time.Now()
        if curPick.ExpirationTime.Before(now) && !skipped {
            slog.Info("Pick expired", "Pick Id", curPick.Id, "Expiration Time", curPick.ExpirationTime, "Now", now)
            //We want to skip the current pick and go to the next one
            nextPickPlayer := model.NextPick(d.database, draftId)
            model.SkipPick(d.database, curPick.Id)
            model.MakePickAvailable(d.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now()))
            d.notifier.NotifyWatchers(draftId)
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
