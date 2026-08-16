package model

import (
	"server/utils"
	"sync"
)

const (
	sessionExpirationDaysEnvKey = "SESSION_EXPIRATION_DAYS"
	defaultSessionExpirationDays = 10

	// Score category labels returned in leaderboard/score maps.
	allianceScoreLabel = "Alliance Score"
	qualScoreLabel     = "Qual Score"
	playoffScoreLabel  = "Playoff Score"
	einsteinScoreLabel = "Einstein Score"
	totalScoreLabel    = "Total Score"
)

var (
	defaultsOnce sync.Once
	defaults     modelDefaults
)

type modelDefaults struct {
	sessionExpirationDays int
}

func loadDefaults() modelDefaults {
	return modelDefaults{
		sessionExpirationDays: utils.MustGetEnvInt(sessionExpirationDaysEnvKey, defaultSessionExpirationDays),
	}
}

func getDefaults() *modelDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// SessionExpirationDays returns the number of days until a session token expires.
func SessionExpirationDays() int { return getDefaults().sessionExpirationDays }
