package background

import (
	"database/sql"
	"errors"
	"log/slog"
	"server/assert"
	"sync"
	"time"
)

type CleanupService struct {
	database  *sql.DB
	interval  int //Number of minutes to wait between runs
	running   bool
	startLock *sync.Mutex
}

func NewCleanupService(database *sql.DB, interval int) *CleanupService {
	return &CleanupService{
		database: database,
		interval: interval,
		running:  false,
	}
}

// TODO Are we using any of this? If not we should start
func (c *CleanupService) Start() error {
	c.startLock.Lock()
	defer c.startLock.Unlock()
	if c.running {
		return errors.New("clean up service already running")
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
	c.startLock.Lock()
	defer c.startLock.Unlock()
	if !c.running {
		return errors.New("clean up service already stopped")
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
	defer func() {
		if err := stmt.Close(); err != nil {
			slog.Error("Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.Exec()
	assert.NoError(err, "Failed To Cleanup Session Tokens")
	slog.Info("Finished iteration of cleanup service")
}
