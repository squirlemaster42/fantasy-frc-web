package background

import (
	"sync"
	"time"

	"server/config"
)

var (
	defaultsOnce sync.Once
	defaults     backgroundDefaults
)

type backgroundDefaults struct {
	CleanupSessionLeewayHours int           `env:"SESSION_CLEANUP_LEEWAY_HOURS" default:"2"`
	CleanupIntervalMinutes    int           `env:"CLEANUP_INTERVAL_MINUTES" default:"60"`
	DraftDaemonTickInterval   time.Duration `env:"DRAFT_DAEMON_TICK_INTERVAL" default:"1m"`
	DraftDaemonTickTimeout    time.Duration `env:"DRAFT_DAEMON_TICK_TIMEOUT" default:"55s"`
}

func loadDefaults() backgroundDefaults {
	var d backgroundDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *backgroundDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// CleanupSessionLeewayHours returns how far before expiration the cleanup
// service deletes sessions.
func CleanupSessionLeewayHours() int { return getDefaults().CleanupSessionLeewayHours }

// CleanupIntervalMinutes returns the delay between cleanup service runs.
func CleanupIntervalMinutes() int { return getDefaults().CleanupIntervalMinutes }

// DraftDaemonTickInterval returns how often the draft daemon wakes up.
func DraftDaemonTickInterval() time.Duration { return getDefaults().DraftDaemonTickInterval }

// DraftDaemonTickTimeout returns the per-tick context timeout for the draft daemon.
func DraftDaemonTickTimeout() time.Duration { return getDefaults().DraftDaemonTickTimeout }
