package background

import (
	"context"
	"database/sql"
	"errors"
	"server/database"
	"server/log"
	"sync"
	"sync/atomic"
	"time"
)

type CleanupService struct {
	database  *sql.DB
	interval  int // Number of minutes to wait between runs
	running   atomic.Bool
	startLock *sync.Mutex
}

func NewCleanupService(database *sql.DB, interval int) *CleanupService {
	return &CleanupService{
		database:  database,
		interval:  interval,
		startLock: &sync.Mutex{},
	}
}

func (c *CleanupService) Start(ctx context.Context) error {
	c.startLock.Lock()
	defer c.startLock.Unlock()
	if c.running.Load() {
		return errors.New("clean up service already running")
	}
	c.running.Store(true)
	log.Info(ctx, "Started cleanup service")
	go func() {
		for c.running.Load() {
			c.cleanExpiredSessionTokens(ctx)
			time.Sleep(time.Duration(c.interval) * time.Minute)
		}
	}()
	return nil
}

func (c *CleanupService) Stop(ctx context.Context) error {
	c.startLock.Lock()
	defer c.startLock.Unlock()
	if !c.running.Load() {
		return errors.New("clean up service already stopped")
	}
	c.running.Store(false)
	log.Info(ctx, "Stopped cleanup service")
	return nil
}

func (c *CleanupService) cleanExpiredSessionTokens(ctx context.Context) {
	log.Debug(ctx, "Starting iteration of cleanup service")
	query := `Delete from UserSessions Where expirationTime < (now()::timestamptz + make_interval(hours => $1));`
	stmt, err := database.Prepare(ctx, c.database, query)
	if err != nil {
		log.Error(ctx, "CleanExpiredSessionTokens: Failed to prepare statement", "error", err)
		return
	}
	defer database.CloseStatement(ctx, stmt, "CleanExpiredSessionTokens")
	_, err = stmt.ExecContext(ctx, CleanupSessionLeewayHours())
	if err != nil {
		log.Error(ctx, "CleanExpiredSessionTokens: Failed to cleanup session tokens", "error", err)
		return
	}
	log.Debug(ctx, "Finished iteration of cleanup service")
}
