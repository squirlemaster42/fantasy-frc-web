package background

import (
	"context"
	"database/sql"
	"errors"
	"server/assert"
	"server/log"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var cleanupTracer = otel.Tracer("cleanup-service")

type CleanupService struct {
	database  *sql.DB
	interval  int //Number of minutes to wait between runs
	running   bool
	startLock *sync.Mutex
	serverCtx context.Context
}

func NewCleanupService(database *sql.DB, interval int) *CleanupService {
	return &CleanupService{
		database:  database,
		interval:  interval,
		running:   false,
		startLock: &sync.Mutex{},
	}
}

func (c *CleanupService) Start(ctx context.Context) error {
	c.startLock.Lock()
	defer c.startLock.Unlock()
	if c.running {
		return errors.New("clean up service already running")
	}
	c.running = true
	c.serverCtx = ctx
	log.Info(ctx, "Started cleanup service")
	go func() {
		for c.running {
			select {
			case <-c.serverCtx.Done():
				log.Info(c.serverCtx, "Cleanup service shutting down")
				return
			default:
			}
			jobCtx, cancel := context.WithTimeout(c.serverCtx, 10*time.Second)
			c.cleanExpiredSessionTokens(jobCtx)
			cancel()

			select {
			case <-c.serverCtx.Done():
				return
			case <-time.After(time.Duration(c.interval) * time.Minute):
			}
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

func (c *CleanupService) cleanExpiredSessionTokens(ctx context.Context) {
	ctx, span := cleanupTracer.Start(ctx, "cleanup.expired-sessions")
	defer span.End()

	log.Info(ctx, "Starting iteration of cleanup service")
	query := `Delete from UserSessions Where expirationTime < (now()::timestamp + '2 hours');`
	assert := assert.CreateAssertWithContext("Clean Expired Session Tokens")
	stmt, err := c.database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "CleanExpiredSessionTokens: Failed to close statement", "error", err)
			span.RecordError(err)
		}
	}()
	_, err = stmt.ExecContext(ctx, )
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	assert.NoError(ctx, err, "Failed To Cleanup Session Tokens")
	log.Info(ctx, "Finished iteration of cleanup service")
}
