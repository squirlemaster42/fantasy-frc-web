package background

import (
	"server/utils"
	"sync"
	"time"
)

const (
	cleanupSessionLeewayHoursEnvKey = "SESSION_CLEANUP_LEEWAY_HOURS"
	defaultCleanupSessionLeewayHours = 2

	cleanupIntervalMinutesEnvKey = "CLEANUP_INTERVAL_MINUTES"
	defaultCleanupIntervalMinutes = 60

	draftDaemonTickIntervalEnvKey = "DRAFT_DAEMON_TICK_INTERVAL"
	defaultDraftDaemonTickInterval = 1 * time.Minute

	draftDaemonTickTimeoutEnvKey = "DRAFT_DAEMON_TICK_TIMEOUT"
	defaultDraftDaemonTickTimeout = 55 * time.Second
)

var (
	defaultsOnce sync.Once
	defaults     backgroundDefaults
)

type backgroundDefaults struct {
	cleanupSessionLeewayHours int
	cleanupIntervalMinutes    int
	draftDaemonTickInterval   time.Duration
	draftDaemonTickTimeout    time.Duration
}

func loadDefaults() backgroundDefaults {
	return backgroundDefaults{
		cleanupSessionLeewayHours: utils.MustGetEnvInt(cleanupSessionLeewayHoursEnvKey, defaultCleanupSessionLeewayHours),
		cleanupIntervalMinutes:    utils.MustGetEnvInt(cleanupIntervalMinutesEnvKey, defaultCleanupIntervalMinutes),
		draftDaemonTickInterval:   utils.MustGetEnvDuration(draftDaemonTickIntervalEnvKey, defaultDraftDaemonTickInterval),
		draftDaemonTickTimeout:    utils.MustGetEnvDuration(draftDaemonTickTimeoutEnvKey, defaultDraftDaemonTickTimeout),
	}
}

func getDefaults() *backgroundDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// CleanupSessionLeewayHours returns how far before expiration the cleanup
// service deletes sessions.
func CleanupSessionLeewayHours() int { return getDefaults().cleanupSessionLeewayHours }

// CleanupIntervalMinutes returns the delay between cleanup service runs.
func CleanupIntervalMinutes() int { return getDefaults().cleanupIntervalMinutes }

// DraftDaemonTickInterval returns how often the draft daemon wakes up.
func DraftDaemonTickInterval() time.Duration { return getDefaults().draftDaemonTickInterval }

// DraftDaemonTickTimeout returns the per-tick context timeout for the draft daemon.
func DraftDaemonTickTimeout() time.Duration { return getDefaults().draftDaemonTickTimeout }
