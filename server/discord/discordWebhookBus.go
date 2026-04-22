package discord

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type DiscordWebhookBus struct {
	client *http.Client
}

type DiscordWebhook struct {
	Username        string          `json:"username"`
	Content         string          `json:"content"`
	AllowedMentions AllowedMentions `json:"allowed_mentions"`
}

type AllowedMentions struct {
	Users []string `json:"users,omitempty"`
	Parse []string `json:"parse,omitempty"`
}

type NextPickDiscordEvent struct {
	PreviousPickedTeam     string
	PreviousPickIdentifier string
	NextPickIdentifier     string
	Webhook                string
}

func NewBus() *DiscordWebhookBus {
	return &DiscordWebhookBus{
		client: &http.Client{},
	}
}

func (d *DiscordWebhookBus) PostPreMatchNotification() error {
	return nil
}

func (d *DiscordWebhookBus) PostPickNotification(event NextPickDiscordEvent) error {
	var allowedMentions []string
	nextId, found := strings.CutPrefix(event.NextPickIdentifier, "<@")
	if found {
		nextId = strings.Trim(nextId, "<@>")
		allowedMentions = []string{
			nextId,
		}
	}

	webhook := DiscordWebhook{
		Username: "Pick Notifier",
		Content: fmt.Sprintf(
			"%s has picked %s. %s it is your pick.",
			event.PreviousPickIdentifier,
			strings.Trim(event.PreviousPickedTeam, "frc"),
			event.NextPickIdentifier,
		),
		AllowedMentions: AllowedMentions{
			Users: allowedMentions,
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
