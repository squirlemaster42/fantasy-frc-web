package tbaHandler

import (
	"sync"
	"time"

	"server/config"
)

var (
	defaultsOnce sync.Once
	defaults     tbaDefaults
)

type tbaDefaults struct {
	AllianceMaxRetries  int           `env:"TBA_ALLIANCE_MAX_RETRIES" default:"5"`
	AllianceBackoffBase time.Duration `env:"TBA_ALLIANCE_BACKOFF_BASE" default:"1s"`
	RequestTimeout      time.Duration `env:"TBA_REQUEST_TIMEOUT" default:"30s"`
}

func loadDefaults() tbaDefaults {
	var d tbaDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *tbaDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// TbaAllianceMaxRetries returns the number of empty-alliance retries.
func TbaAllianceMaxRetries() int { return getDefaults().AllianceMaxRetries }

// TbaAllianceBackoffBase returns the base duration for exponential backoff.
func TbaAllianceBackoffBase() time.Duration { return getDefaults().AllianceBackoffBase }

// TbaRequestTimeout returns the timeout for outbound TBA API requests.
func TbaRequestTimeout() time.Duration { return getDefaults().RequestTimeout }
