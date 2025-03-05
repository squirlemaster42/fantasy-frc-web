package background

import (
	"database/sql"
	"errors"
	"server/logging"
	"server/model"
	"sync"
	"time"
)

var PICK_TIME time.Duration = 3 * time.Hour

var ALLOWED_TIMES = map[time.Weekday]TimeRange {
    time.Sunday: {
        startTime: time.Now(),
        endTime: time.Now(),
    },
    time.Monday: {
        startTime: time.Now(),
        endTime: time.Now(),
    },
    time.Tuesday: {
        startTime: time.Now(),
        endTime: time.Now(),
    },
    time.Wednesday: {
        startTime: time.Now(),
        endTime: time.Now(),
    },
    time.Thursday: {
        startTime: time.Now(),
        endTime: time.Now(),
    },
    time.Friday: {
        startTime: time.Now(),
        endTime: time.Now(),
    },
    time.Saturday: {
        startTime: time.Now(),
        endTime: time.Now(),
    },
}

type TimeRange struct {
    startTime time.Time
    endTime time.Time
}

type DraftDaemon struct {
    logger *logging.Logger
    database *sql.DB
    running bool
    interval int //Time to wait between runs
    mu sync.Mutex
    runningDrafts map[int]bool
}

func NewDraftDaemon(logger *logging.Logger, database *sql.DB) *DraftDaemon {
    return &DraftDaemon{
        logger: logger,
        database: database,
        running: false,
        runningDrafts: make(map[int]bool),
    }
}

func (d *DraftDaemon) Start() error {
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

            //TODO We need to for work/school hours
            if time.Since(curPick.AvailableTime) > PICK_TIME {
                //If the pick has not been make after this time we need to mark the
                //Pick as skipped and make the next one
            }
        }

        time.Sleep(5 * time.Minute)
    }
}

func (d *DraftDaemon) AddDraft(draftId int) error {
    if d.runningDrafts[draftId] {
        return errors.New("Draft already added to Daemon")
    }
    d.runningDrafts[draftId] = true
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
    d.running = false
    return nil
}
