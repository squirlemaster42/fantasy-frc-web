package background

import (
	"database/sql"
	"errors"
	"server/logging"
	"sync"
)

type DraftDaemon struct {
    logger *logging.Logger
    database *sql.DB
    running bool
    interval int //Time to wait between runs
    mu sync.Mutex
}

func NewDraftDaemon(logger *logging.Logger, database *sql.DB) *DraftDaemon {
    return &DraftDaemon{
        logger: logger,
        database: database,
        running: false,
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
        //We need to get all of the current picks
    }
}

func (d *DraftDaemon) IsRunning() bool {
    return d.running
}

func (d *DraftDaemon) Stop() error {
    d.running = false
    return nil
}
