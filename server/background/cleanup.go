package background

import (
	"database/sql"
	"errors"
	"server/assert"
	"server/log"
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

func (c *CleanupService) Start() error {
	c.startLock.Lock()
	defer c.startLock.Unlock()
	if c.running {
		return errors.New("clean up service already running")
	}
	c.running = true
	log.InfoNoContext("Started cleanup service")
	go func() {
		for c.running {
			c.cleanExpiredSessionTokens()
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

func (c *CleanupService) cleanExpiredSessionTokens() {
	log.InfoNoContext("Starting iteration of cleanup service")
	query := `Delete from UserSessions Where expirationTime < (now()::timestamp + '2 hours');`
	assert := assert.CreateAssertWithContext("Clean Expired Session Tokens")
	stmt, err := c.database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.WarnNoContext("CleanExpiredSessionTokens: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.Exec()
	assert.NoError(err, "Failed To Cleanup Session Tokens")
	log.InfoNoContext("Finished iteration of cleanup service")
}
