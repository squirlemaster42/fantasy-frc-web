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
	"time"
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

type PreMatchDiscordEvent struct {
	EventName     string
	PredictedTime time.Time
	IdsToTeams    map[string][]string
	Webhook       string
}

type NextPickDiscordEvent struct {
	PreviousPickedTeam    string
	PreviousPickName      string
	PreviousPickDiscordId sql.NullString
	NextPickName          string
	NextPickDiscordId     sql.NullString
	Webhook               string
	ExpirationTime        time.Time
}

func NewBus() *DiscordWebhookBus {
	return &DiscordWebhookBus{
		client: &http.Client{},
	}
}

func (d *DiscordWebhookBus) PostPreMatchNotification(event PreMatchDiscordEvent) error {
	var message string
	message += fmt.Sprintf("Upcoming Match at %s\nExpected to start at <t:%d:f>\n", event.EventName, event.PredictedTime.Unix())

	for discordId, teamIds := range event.IdsToTeams {
		for _, teamId := range teamIds {
			teamNumber, _ := strings.CutPrefix(teamId, "frc")
			message += fmt.Sprintf("%s, team %s is about to compete.\n", discordId, teamNumber)
		}
	}

	webhook := DiscordWebhook{
		Username: "Match Notifier",
		Content:  message,
		AllowedMentions: AllowedMentions{
			Parse: []string{"users"},
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

func (d *DiscordWebhookBus) PostPickNotification(event NextPickDiscordEvent) error {
	previousIdentifier := event.PreviousPickName
	if event.PreviousPickDiscordId.Valid {
		previousPickId := event.PreviousPickDiscordId.String
		// discord IDs are unique 17+ digit integers, so we can validate them by checking length and seeing if they are numbers
		_, err := strconv.ParseUint(previousPickId, 10, 64)
		if len(previousPickId) >= 17 && err == nil {
			previousIdentifier = fmt.Sprintf("<@%s>", previousPickId)
		}
	}

	var allowedUserMentions []string
	nextIdentifier := event.NextPickName
	if event.NextPickDiscordId.Valid {
		nextPickId := event.NextPickDiscordId.String
		// discord IDs are unique 17+ digit integers, so we can validate them by checking length and seeing if they are numbers
		_, err := strconv.ParseUint(nextPickId, 10, 64)
		if len(nextPickId) >= 17 && err == nil {
			nextIdentifier = fmt.Sprintf("<@%s>", nextPickId)
			allowedUserMentions = []string{
				nextPickId,
			}
		}
	}

	message := "%s has picked %s. %s it is your pick. Your pick expires at <t:%d:f>."
	if previousIdentifier == nextIdentifier {
		message = "%s has picked %s, and %s it is your turn again. Your pick expires at <t:%d:f>."
	}

	webhook := DiscordWebhook{
		Username: "Pick Notifier",
		Content: fmt.Sprintf(
			message,
			previousIdentifier,
			strings.Trim(event.PreviousPickedTeam, "frc"),
			nextIdentifier,
			event.ExpirationTime.Unix(),
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
