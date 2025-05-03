package background

import (
	"database/sql"
	"errors"
	"log/slog"
	"server/assert"
	"time"
)

type CleanupService struct {
    database *sql.DB
    interval int //Number of minutes to wait between runs
    running bool
}

func NewCleanupService(database *sql.DB, interval int) *CleanupService {
    return &CleanupService{
        database: database,
        interval: interval,
        running: false,
    }
}

func (c *CleanupService) Start() error {
    //TODO we need a mutex on this start
    if c.running {
        return errors.New("Clean up service already running")
    }
    c.running = true
    slog.Info("Started cleanup service")
    go func() {
        for c.running {
            c.CleanExpiredSessionTokens()
            time.Sleep(time.Duration(c.interval) * time.Minute)
        }
    }()
    return nil
}

func (c *CleanupService) Stop() error {
    if !c.running {
        return errors.New("Clean up service already stopped")
    }
    c.running = false
    return nil
}

func (c *CleanupService) CleanExpiredSessionTokens() {
    slog.Info("Starting iteration of cleanup service")
    query := `Delete from UserSessions Where expirationTime < (now()::timestamp + '2 hours');`
    assert := assert.CreateAssertWithContext("Clean Expired Session Tokens")
    stmt, err := c.database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec()
    assert.NoError(err, "Failed To Cleanup Session Tokens")
    slog.Info("Finished iteration of cleanup service")
}

