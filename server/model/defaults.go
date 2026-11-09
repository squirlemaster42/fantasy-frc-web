package model

import (
	"sync"

	"server/config"
)

const (
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
	SessionExpirationDays int `env:"SESSION_EXPIRATION_DAYS" default:"10"`
}

func loadDefaults() modelDefaults {
	var d modelDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *modelDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// SessionExpirationDays returns the number of days until a session token expires.
func SessionExpirationDays() int { return getDefaults().SessionExpirationDays }
