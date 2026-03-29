package discord

import "net/http"

type DiscordWebhookBus struct {
	client *http.Client
}

func NewBus() *DiscordWebhookBus {
	return &DiscordWebhookBus {
		client: &http.Client{},
	}
}

func (d *DiscordWebhookBus) PostPreMatchNotification() error {
	return nil
}
