package discord

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"server/log"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DiscordWebhookBus struct {
	client     *http.Client
	preMatchCh chan PreMatchDiscordEvent
	stopCh     chan struct{}
	wg         sync.WaitGroup
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
	d := &DiscordWebhookBus{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		preMatchCh: make(chan PreMatchDiscordEvent, 100),
		stopCh:     make(chan struct{}),
	}
	d.wg.Add(1)
	go d.worker()
	return d
}

func (d *DiscordWebhookBus) Stop() {
	close(d.stopCh)
	d.wg.Wait()
}

func (d *DiscordWebhookBus) worker() {
	defer d.wg.Done()
	for {
		select {
		case event := <-d.preMatchCh:
			d.sendPreMatchNotification(event)
		case <-d.stopCh:
			// Drain remaining events before stopping.
			for {
				select {
				case event := <-d.preMatchCh:
					d.sendPreMatchNotification(event)
				default:
					return
				}
			}
		}
	}
}

func (d *DiscordWebhookBus) PostPreMatchNotification(event PreMatchDiscordEvent) error {
	select {
	case d.preMatchCh <- event:
		return nil
	default:
		return fmt.Errorf("pre-match notification queue is full, dropping event for %s", event.EventName)
	}
}

func (d *DiscordWebhookBus) sendPreMatchNotification(event PreMatchDiscordEvent) {
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
		log.WarnNoContext("Failed to marshal discord pre-match webhook", "Error", err)
		return
	}

	req, err := http.NewRequest("POST", event.Webhook, bytes.NewBuffer(jsonData))
	if err != nil {
		log.WarnNoContext("Failed to create discord pre-match webhook request", "Error", err)
		return
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		log.WarnNoContext("Failed to post discord pre-match webhook", "Error", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.WarnNoContext("Failed to read discord error response body", "Error", err)
			return
		}
		log.WarnNoContext("Discord pre-match webhook was not successful", "Status", resp.StatusCode, "Body", string(body))
	}
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
