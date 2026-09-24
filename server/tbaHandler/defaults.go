package tbaHandler

import (
	"server/utils"
	"sync"
	"time"
)

const (
	tbaAllianceMaxRetriesEnvKey = "TBA_ALLIANCE_MAX_RETRIES"
	defaultTbaAllianceMaxRetries = 5

	tbaAllianceBackoffBaseEnvKey = "TBA_ALLIANCE_BACKOFF_BASE"
	defaultTbaAllianceBackoffBase = 1 * time.Second

	tbaRequestTimeoutEnvKey = "TBA_REQUEST_TIMEOUT"
	defaultTbaRequestTimeout = 30 * time.Second
)

var (
	defaultsOnce sync.Once
	defaults     tbaDefaults
)

type tbaDefaults struct {
	allianceMaxRetries  int
	allianceBackoffBase time.Duration
	requestTimeout      time.Duration
}

func loadDefaults() tbaDefaults {
	return tbaDefaults{
		allianceMaxRetries:  utils.MustGetEnvInt(tbaAllianceMaxRetriesEnvKey, defaultTbaAllianceMaxRetries),
		allianceBackoffBase: utils.MustGetEnvDuration(tbaAllianceBackoffBaseEnvKey, defaultTbaAllianceBackoffBase),
		requestTimeout:      utils.MustGetEnvDuration(tbaRequestTimeoutEnvKey, defaultTbaRequestTimeout),
	}
}

func getDefaults() *tbaDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// TbaAllianceMaxRetries returns the number of empty-alliance retries.
func TbaAllianceMaxRetries() int { return getDefaults().allianceMaxRetries }

// TbaAllianceBackoffBase returns the base duration for exponential backoff.
func TbaAllianceBackoffBase() time.Duration { return getDefaults().allianceBackoffBase }

// TbaRequestTimeout returns the timeout for outbound TBA API requests.
func TbaRequestTimeout() time.Duration { return getDefaults().requestTimeout }
