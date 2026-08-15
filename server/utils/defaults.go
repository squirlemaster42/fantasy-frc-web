package utils

import (
	"strings"
	"sync"
)

const (
	tbaEventCodesEnvKey = "TBA_EVENT_CODES"
	defaultTbaEventCodes = "arc,cur,dal,gal,hop,joh,mil,new,cmptx"

	defaultTimezone = "America/New_York"

	tbaWebhookSecretFileEnvKey = "TBA_WEBHOOK_SECRET_FILE"
	defaultTbaWebhookSecretFile = "./webhookSecret.txt"

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
	eventCodes         []string
	timezone           string
	webhookSecretFile  string
}

func loadDefaults() utilsDefaults {
	codes := MustGetEnvString(tbaEventCodesEnvKey, defaultTbaEventCodes)
	return utilsDefaults{
		eventCodes:        strings.Split(codes, ","),
		timezone:          defaultTimezone,
		webhookSecretFile: MustGetEnvString(tbaWebhookSecretFileEnvKey, defaultTbaWebhookSecretFile),
	}
}

func getDefaults() *utilsDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// Timezone returns the canonical IANA timezone used for scheduling and display.
func Timezone() string { return getDefaults().timezone }

// WebhookSecretFile returns the path to the TBA webhook verification file.
func WebhookSecretFile() string { return getDefaults().webhookSecretFile }

func eventCodes() []string { return getDefaults().eventCodes }
