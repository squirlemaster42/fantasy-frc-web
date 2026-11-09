package discord

import (
	"sync"
	"time"

	"server/config"
)

var (
	defaultsOnce sync.Once
	defaults     discordDefaults
)

type discordDefaults struct {
	WebhookTimeout      time.Duration `env:"DISCORD_WEBHOOK_TIMEOUT" default:"15s"`
	PreMatchQueueBuffer int           `env:"DISCORD_PREMATCH_QUEUE_BUFFER" default:"100"`
	MinIdLength         int           `env:"DISCORD_MIN_ID_LENGTH" default:"17"`
}

func loadDefaults() discordDefaults {
	var d discordDefaults
	config.MustLoad(&d)
	return d
}

func getDefaults() *discordDefaults {
	defaultsOnce.Do(func() { defaults = loadDefaults() })
	return &defaults
}

// DiscordWebhookTimeout returns the HTTP client timeout for Discord webhook calls.
func DiscordWebhookTimeout() time.Duration { return getDefaults().WebhookTimeout }

// DiscordPreMatchQueueBuffer returns the capacity of the pre-match notification channel.
func DiscordPreMatchQueueBuffer() int { return getDefaults().PreMatchQueueBuffer }

// DiscordMinIdLength returns the minimum valid Discord snowflake length.
func DiscordMinIdLength() int { return getDefaults().MinIdLength }
