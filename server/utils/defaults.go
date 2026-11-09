package utils

import (
	"strings"
	"sync"

	"server/config"
)

const (
	defaultTimezone = "America/New_York"

	// Sentinel values for URL helpers.
	CreateDraftId = -1

	// Match competition levels.
	MatchLevelQual       = "qm"
	MatchLevelQuarters   = "qf"
	MatchLevelSemifinals = "sf"
	MatchLevelFinals     = "f"

	// Match key parsing constants.
	matchLevelPrefixLengthLong  = 2
	matchLevelPrefixLengthShort = 1
)

// DefaultEventCodes returns the championship event codes for the current season.
func DefaultEventCodes() []string {
	return eventCodes()
}

var (
	defaultsOnce sync.Once
	defaults     utilsDefaults
)

type utilsDefaults struct {
	EventCodes        []string `env:"TBA_EVENT_CODES" default:"arc,cur,dal,gal,hop,joh,mil,new,cmptx"`
	WebhookSecretFile string   `env:"TBA_WEBHOOK_SECRET_FILE" default:"./webhookSecret.txt"`
}

func loadDefaults() utilsDefaults {
	var d utilsDefaults
	config.MustLoad(&d)
	d.EventCodes = trimStrings(d.EventCodes)
	return d
}

func getDefaults() *utilsDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// Timezone returns the canonical IANA timezone used for scheduling and display.
func Timezone() string { return defaultTimezone }

// WebhookSecretFile returns the path to the TBA webhook verification file.
func WebhookSecretFile() string { return getDefaults().WebhookSecretFile }

func eventCodes() []string { return getDefaults().EventCodes }

func trimStrings(s []string) []string {
	for i := range s {
		s[i] = strings.TrimSpace(s[i])
	}
	return s
}
