package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type DiscordWebhookBus struct {
	client *http.Client
}

type DiscordWebhook struct {
	Username string `json:"username"`
	Content string `json:"content"`
	AllowedMentions AllowedMentions `json:"allowed_mentions"`
}

type AllowedMentions struct {
	Users []string `json:"users,omitempty"`
	Parse []string `json:"parse,omitempty"`
}

type NextPickDiscordEvent struct {
	PreviousPickedTeam string
	PreviousPickDiscordId string
	DiscordId string
	Webhook string
}

func NewBus() *DiscordWebhookBus {
	return &DiscordWebhookBus {
		client: &http.Client{},
	}
}

func (d *DiscordWebhookBus) PostPreMatchNotification() error {
	return nil
}

func (d *DiscordWebhookBus) PostPickNotification(event NextPickDiscordEvent) error {
	webhook := DiscordWebhook {
		Username: "Pick Notifier",
		Content: fmt.Sprintf(
			"<@%s> has picked %s. <@%s> it is your pick.",
			event.PreviousPickDiscordId,
			event.PreviousPickedTeam,
			event.DiscordId,
		),
		AllowedMentions: AllowedMentions {
			Users: []string {
				event.DiscordId,
			},
		},
	}

	jsonData, err := json.Marshal(webhook)

	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", event.Webhook, bytes.NewBuffer(jsonData))
	req.Header.Add("Content-Type", "application/json")

	resp, err := d.client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("Discord webhook was not successful: %s", string(body))
	}

	return nil
}
