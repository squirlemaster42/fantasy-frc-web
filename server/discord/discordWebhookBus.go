package discord

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
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
	PreviousPickedTeam    string
	PreviousPickName      string
	PreviousPickDiscordId sql.NullString
	NextPickName          string
	NextPickDiscordId     sql.NullString
	Webhook               string
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
	previousIdentifier := event.PreviousPickName
	if event.PreviousPickDiscordId.Valid {
		previousPickId := event.PreviousPickDiscordId.String
		_, err := strconv.ParseUint(previousPickId, 10, 64)
		if len(previousPickId) >= 17 && err == nil {
			previousIdentifier = fmt.Sprintf("<@%s>", previousPickId)
		}
	}

	var allowedUserMentions []string
	nextIdentifier := event.NextPickName
	if event.NextPickDiscordId.Valid {
		nextPickId := event.NextPickDiscordId.String
		_, err := strconv.ParseUint(nextPickId, 10, 64)
		if len(nextPickId) >= 17 && err == nil {
			nextIdentifier = fmt.Sprintf("<@%s>", nextPickId)
			allowedUserMentions = []string{
				nextPickId,
			}
		}
	}

	message := "%s has picked %s. %s it is your pick."
	if previousIdentifier == nextIdentifier {
		message = "%s has picked %s, and %s it is your turn again."
	}

	webhook := DiscordWebhook{
		Username: "Pick Notifier",
		Content: fmt.Sprintf(
			message,
			previousIdentifier,
			strings.Trim(event.PreviousPickedTeam, "frc"),
			nextIdentifier,
		),
		AllowedMentions: AllowedMentions{
			Users: allowedUserMentions,
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
