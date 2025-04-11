package background

import (
	"database/sql"
	"errors"
	"log/slog"
	"server/handler"
	"server/model"
	"server/utils"
	"sync"
	"time"
)

type DraftDaemon struct {
    database *sql.DB
    running bool
    interval int //Time to wait between runs
    mu sync.Mutex
    runningDrafts map[int]bool
    notifier *handler.PickNotifier
}

func NewDraftDaemon(database *sql.DB, notifier *handler.PickNotifier) *DraftDaemon {
    return &DraftDaemon{
        database: database,
        running: false,
        runningDrafts: make(map[int]bool),
        notifier: &handler.PickNotifier{},
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
        return errors.New("Cannot start a draft daemon that has already been started")
    }
}

func (d *DraftDaemon) Run() {
    for d.running {
        //Get current picks for the running drafts
        slog.Info("Starting iteration of the Draft Daemon")
        for draftId := range d.runningDrafts {
            curPick := model.GetCurrentPick(d.database, draftId)
            slog.Info("Checking expiration time", "Draft Id", draftId, "Current Pick Player", curPick.Player)
            now := time.Now()
            if curPick.ExpirationTime.Before(now) {
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

        time.Sleep(1 * time.Minute)
    }
}

func (d *DraftDaemon) AddDraft(draftId int) error {
    if d.runningDrafts[draftId] {
        return errors.New("Draft already added to Daemon")
    }
    d.runningDrafts[draftId] = true
    slog.Info("Added draft to daemon", "Draft Id", draftId)
    return nil
}


func (d *DraftDaemon) RemoveDraft(draftId int) error {
    if !d.runningDrafts[draftId] {
        return errors.New("Draft not in Daemon")
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
