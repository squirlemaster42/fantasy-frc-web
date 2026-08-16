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
)

var (
	defaultsOnce sync.Once
	defaults     tbaDefaults
)

type tbaDefaults struct {
	allianceMaxRetries int
	allianceBackoffBase time.Duration
}

func loadDefaults() tbaDefaults {
	return tbaDefaults{
		allianceMaxRetries:  utils.MustGetEnvInt(tbaAllianceMaxRetriesEnvKey, defaultTbaAllianceMaxRetries),
		allianceBackoffBase: utils.MustGetEnvDuration(tbaAllianceBackoffBaseEnvKey, defaultTbaAllianceBackoffBase),
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
