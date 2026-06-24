package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"server/discord"
	"server/log"
	"server/swagger"
	"server/utils"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type TbaWebsocketEvent struct {
	MessageType string          `json:"message_type"`
	MessageData json.RawMessage `json:"message_data"`
}

func validMAC(message []byte, messageMAC string, key []byte) bool {
	messageMACBytes, err := hex.DecodeString(messageMAC)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	return hmac.Equal(messageMACBytes, expectedMAC)
}

func (h *Handler) ConsumeTbaWebhook(c echo.Context) error {
	log.Debug(c.Request().Context(), "Received webhook message")
	c.Request().Body = http.MaxBytesReader(c.Response().Writer, c.Request().Body, 1<<20) // 1 MB
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			log.Warn(c.Request().Context(), "Webhook payload too large")
			return c.NoContent(http.StatusRequestEntityTooLarge)
		}
		log.Error(c.Request().Context(), "Failed to read request body", "error", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	// Validate HMAC BEFORE processing any events
	messageMac := c.Request().Header.Get("X-TBA-HMAC")
	valid := validMAC(body, messageMac, []byte(h.TbaWebhookSecret))

	if !valid {
		log.Warn(c.Request().Context(), "Webhook event authentication failed", "message", string(body))
		return c.NoContent(http.StatusOK)
	}

	var event TbaWebsocketEvent
	err = json.Unmarshal(body, &event)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to decode webhook message", "error", err, "message", string(body))
		return c.NoContent(http.StatusInternalServerError)
	}

	if event.MessageType == "verification" {
		h.HandleVerificationEvent(c.Request().Context(), event.MessageData)
		return c.NoContent(http.StatusOK)
	}

	log.Debug(c.Request().Context(), "Routing event", "messageType", event.MessageType)
	switch event.MessageType {
	case "upcoming_match":
		go h.HandleUpcomingMatchEvent(c.Request().Context(), event.MessageData)
	case "match_score":
		go h.HandleMatchScoreEvent(c.Request().Context(), event.MessageData)
	case "match_video":
		h.HandleMatchVideoEvent(c.Request().Context(), event.MessageData)
	case "starting_comp_level":
		h.HandleCompLevelStartingEvent(c.Request().Context(), event.MessageData)
	case "alliance_selection":
		go h.HandleAllianceSelectionEvent(c.Request().Context(), event.MessageData)
	case "awards_posted":
		h.HandleAwardsPostedEvent(c.Request().Context(), event.MessageData)
	case "schedule_updated":
		h.HandleEventScheduleUpdatedEvent(c.Request().Context(), event.MessageData)
	case "ping":
		h.HandlePingEvent(c.Request().Context(), event.MessageData)
	case "broadcast":
		h.HandleBroadcastEvent(c.Request().Context(), event.MessageData)
	default:
		log.Warn(c.Request().Context(), "Unknown websocket event detected", "messageType", event.MessageType, "message", event.MessageData)
	}

	return c.NoContent(http.StatusOK)
}

type MatchScoreNotification struct {
	EventKey  string        `json:"event_key"`
	MatchKey  string        `json:"match_key"`
	TeamKey   string        `json:"team_key"`
	EventName string        `json:"event_name"`
	Match     swagger.Match `json:"match"`
}

func (h *Handler) HandleMatchScoreEvent(ctx context.Context, messageData json.RawMessage) {
	log.Debug(ctx, "Received match score event", "message", messageData)
	var scoreNotification MatchScoreNotification
	err := json.Unmarshal(messageData, &scoreNotification)
	if err != nil {
		log.Warn(ctx, "Failed to decode match score notification", "error", err, "message", messageData)
		return
	}
	h.Scorer.AddMatchToScore(scoreNotification.Match)
}

type AllianceSelectionNotification struct {
	EventKey  string        `json:"event_key"`
	TeamKey   string        `json:"team_key"`
	EventName string        `json:"event_name"`
	Event     swagger.Event `json:"event"`
}

func (h *Handler) HandleAllianceSelectionEvent(ctx context.Context, messageData json.RawMessage) {
	log.Debug(ctx, "Received alliance selection event", "message", messageData)
	var notification AllianceSelectionNotification
	err := json.Unmarshal(messageData, &notification)
	if err != nil {
		log.Warn(ctx, "Failed to decode alliance selection notification", "error", err, "message", messageData)
		return
	}
	h.Scorer.ScoreAllianceSelection(ctx, notification.EventKey)
}

type UpcomingMatchEvent struct {
	EventKey      string          `json:"event_key"`
	MatchKey      string          `json:"match_key"`
	TeamKey       string          `json:"team_key"`
	EventName     string          `json:"event_name"`
	TeamKeys      []string        `json:"team_keys"`
	ScheduledTime int64           `json:"scheduled_time"`
	PredictedTime int64           `json:"predicted_time"`
	Webcast       swagger.Webcast `json:"webcast"`
}

func (h *Handler) HandleUpcomingMatchEvent(ctx context.Context, messageData json.RawMessage) {
	var tbaEvent UpcomingMatchEvent
	if err := json.Unmarshal(messageData, &tbaEvent); err != nil {
		log.Warn(ctx, "Failed to decode upcoming match event data", "error", err)
		return
	}

	if len(tbaEvent.TeamKeys) != 6 {
		log.Warn(ctx, "Upcoming match received without 6 teams", "TeamCount", len(tbaEvent.TeamKeys))
		return
	}

	rows, err := h.DraftStore.GetDraftPickRows(ctx, tbaEvent.TeamKeys)

	if err != nil {
		log.Error(ctx, "Failed to get picked rows", "error", err)
		return
	}

	// map of drafts to events
	draftMap := make(map[int]*discord.PreMatchDiscordEvent)

	for _, row := range rows {
		// init the event for this draft if it doesn't exist and the webhook is valid
		if row.Webhook.Valid {
			_, exists := draftMap[row.DraftId]
			if !exists && row.Webhook.Valid {
				draftMap[row.DraftId] = &discord.PreMatchDiscordEvent{
					EventName:     tbaEvent.EventName,
					PredictedTime: time.Unix(tbaEvent.PredictedTime, 0),
					Webhook:       row.Webhook.String,
					IdsToTeams:    make(map[string][]string),
				}
			}

			// Username by default but use discord id if found
			discordId := row.Username
			if row.DiscordId.Valid {
				// discord IDs must be 17+ characters and all numbers, so this is a quick way to mostly validate
				// that the id in the database is not just a random string
				_, err := strconv.ParseUint(row.DiscordId.String, 10, 64)
				if len(row.DiscordId.String) >= 17 && err == nil {
					discordId = fmt.Sprintf("<@%s>", row.DiscordId.String)
				}
			}

			// add user with that pick to that draft
			draftMap[row.DraftId].IdsToTeams[discordId] = append(draftMap[row.DraftId].IdsToTeams[discordId], row.Pick)
		}
	}

	// Send each event to the Discord Bus
	for _, event := range draftMap {
		if len(event.IdsToTeams) > 0 {
			log.Info(ctx, "Posting pre match notification webhook")
			err := h.DiscordWebhookBus.PostPreMatchNotification(*event)
			if err != nil {
				log.Error(ctx, "Failed to post pre match notification webhook", "error", err)
			}
		}
	}
}

type MatchVideoNotification struct {
	EventKey  string        `json:"event_key"`
	MatchKey  string        `json:"match_key"`
	TeamKey   string        `json:"team_key"`
	EventName string        `json:"event_name"`
	Match     swagger.Match `json:"match"`
}

func (h *Handler) HandleMatchVideoEvent(ctx context.Context, messageData json.RawMessage) {
	log.Debug(ctx, "Received match video event", "message", messageData)
}

type CompLevelStartingEvent struct {
	EventKey      string `json:"event_key"`
	EventName     string `json:"event_name"`
	CompLevel     string `json:"comp_level"`
	ScheduledTime string `json:"scheduled_time"`
}

func (h *Handler) HandleCompLevelStartingEvent(ctx context.Context, messageData json.RawMessage) {
	log.Debug(ctx, "Received comp level starting event", "message", messageData)
}

type AwardsPostedEvent struct {
	EventKey  string          `json:"event_key"`
	TeamKey   string          `json:"team_key"`
	EventName string          `json:"event_name"`
	Awards    []swagger.Award `json:"awards"`
}

func (h *Handler) HandleAwardsPostedEvent(ctx context.Context, messageData json.RawMessage) {
	log.Debug(ctx, "Received awards posted event", "message", messageData)
}

type EventScheduleUpdatedEvent struct {
	EventKey       string `json:"event_key"`
	EventName      string `json:"event_name"`
	FirstMatchTime int64  `json:"first_match_time"`
}

func (h *Handler) HandleEventScheduleUpdatedEvent(ctx context.Context, messageData json.RawMessage) {
	log.Debug(ctx, "Received event schedule updated event", "message", messageData)
}

type PingEvent struct {
	Title       string `json:"title"`
	Description string `json:"desc"`
}

func (h *Handler) HandlePingEvent(ctx context.Context, messageData json.RawMessage) {
	log.Debug(ctx, "Received ping event", "message", messageData)
}

type BroadcastEvent struct {
	Title       string `json:"title"`
	Description string `json:"desc"`
	Url         string `json:"url"`
}

func (h *Handler) HandleBroadcastEvent(ctx context.Context, messageData json.RawMessage) {
	log.Debug(ctx, "Received broadcast event", "message", messageData)
}

type VerificationEvent struct {
	VerificationKey string `json:"verification_key"`
}

func (h *Handler) HandleVerificationEvent(ctx context.Context, messageData json.RawMessage) {
	log.Debug(ctx, "Received Verification Event", "message", messageData)

	var event VerificationEvent
	err := json.Unmarshal(messageData, &event)
	if err != nil {
		log.Warn(ctx, "Failed to decode verification event", "error", err, "message", messageData)
		return
	}

	h.TbaVerificationCode = event.VerificationKey

	// Only create the file if it doesn't already exist
	_, err = os.Stat(utils.GetWebhookFilePath())
	if os.IsNotExist(err) {
		err = os.WriteFile(utils.GetWebhookFilePath(), []byte(h.TbaVerificationCode), 0600)
		if err != nil {
			log.Error(ctx, "Failed to write tba webhook file body", "error", err)
		}
	} else if err != nil {
		log.Error(ctx, "Failed to check if webhook file exists", "error", err)
	}
}
