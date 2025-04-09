package background

import (
	"database/sql"
	"errors"
	"log/slog"
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
}

func NewDraftDaemon(database *sql.DB) *DraftDaemon {
    return &DraftDaemon{
        database: database,
        running: false,
        runningDrafts: make(map[int]bool),
    }
}

func (d *DraftDaemon) Start() error {
    slog.Info("Started draft daemon")
    d.mu.Lock()
    defer d.mu.Unlock()
    if !d.running {
        d.running = true
        go d.Run()
        return nil
    } else {
        return errors.New("Cannot start a draft daemon that has already been started")
    }
}

func (d *DraftDaemon) Run() {
    for d.running {
        //Get current picks for the running drafts
        for draftId := range d.runningDrafts {
            curPick := model.GetCurrentPick(d.database, draftId)
            slog.Info("Checking expiration time", "Draft Id", draftId, "Current Pick Player", curPick.Player)
            if curPick.ExpirationTime.After(time.Now()) {
                slog.Info("Pick expired", "Pick Id", curPick.Id, "Expiration Time", curPick.ExpirationTime)
                //We want to skip the current pick and go to the next one
                nextPickPlayer := model.NextPick(d.database, draftId)
                model.SkipPick(d.database, curPick.Id)
                model.MakePickAvailable(d.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now()))
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
