package discord

import (
	"server/utils"
	"sync"
	"time"
)

const (
	discordWebhookTimeoutEnvKey = "DISCORD_WEBHOOK_TIMEOUT"
	defaultDiscordWebhookTimeout = 15 * time.Second

	discordPreMatchQueueBufferEnvKey = "DISCORD_PREMATCH_QUEUE_BUFFER"
	defaultDiscordPreMatchQueueBuffer = 100

	discordMinIdLengthEnvKey = "DISCORD_MIN_ID_LENGTH"
	defaultDiscordMinIdLength = 17
)

var (
	defaultsOnce sync.Once
	defaults     discordDefaults
)

type discordDefaults struct {
	webhookTimeout     time.Duration
	preMatchQueueBuffer int
	minIdLength        int
}

func loadDefaults() discordDefaults {
	return discordDefaults{
		webhookTimeout:      utils.MustGetEnvDuration(discordWebhookTimeoutEnvKey, defaultDiscordWebhookTimeout),
		preMatchQueueBuffer: utils.MustGetEnvInt(discordPreMatchQueueBufferEnvKey, defaultDiscordPreMatchQueueBuffer),
		minIdLength:         utils.MustGetEnvInt(discordMinIdLengthEnvKey, defaultDiscordMinIdLength),
	}
}

func getDefaults() *discordDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// DiscordWebhookTimeout returns the HTTP client timeout for Discord webhook calls.
func DiscordWebhookTimeout() time.Duration { return getDefaults().webhookTimeout }

// DiscordPreMatchQueueBuffer returns the capacity of the pre-match notification channel.
func DiscordPreMatchQueueBuffer() int { return getDefaults().preMatchQueueBuffer }

// DiscordMinIdLength returns the minimum valid Discord snowflake length.
func DiscordMinIdLength() int { return getDefaults().minIdLength }
