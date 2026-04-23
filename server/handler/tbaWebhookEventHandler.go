package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"server/discord"
	"server/log"
	"server/model"
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

func validMAC(message []byte, messageMAC []byte, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
	log.InfoNoContext("Validating HMAC", "HMAC", messageMAC, "Expected HMAC", hex.EncodeToString(expectedMAC))
	return hmac.Equal(messageMAC, []byte(hex.EncodeToString(expectedMAC)))
}

func (h *Handler) ConsumeTbaWebhook(c echo.Context) error {
	log.Info(c.Request().Context(), "Received webhook message")
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to read request body", "Error", err)
		return nil
	}

	var event TbaWebsocketEvent
	err = json.Unmarshal(body, &event)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to decode webhook message", "Error", err, "Message", string(body))
		return nil
	}

	if event.MessageType == "verification" {
		h.HandleVerificationEvent(event.MessageData)
	}

	messageMac := c.Request().Header.Get("X-TBA-HMAC")
	valid := validMAC(body, []byte(messageMac), []byte(h.TbaWebhookSecret))

	if !valid {
		log.Warn(c.Request().Context(), "Webhook event authentication failed", "Message", string(body))
		return nil
	}

	log.Info(c.Request().Context(), "Routing event", "Message Type", event.MessageType)
	switch event.MessageType {
	case "upcoming_match":
		h.HandleUpcomingMatchEvent(event.MessageData)
	case "match_score":
		h.HandleMatchScoreEvent(event.MessageData)
	case "match_video":
		h.HandleMatchVideoEvent(event.MessageData)
	case "starting_comp_level":
		h.HandleCompLevelStartingEvent(event.MessageData)
	case "alliance_selection":
		h.HandleAllianceSelectionEvent(event.MessageData)
	case "awards_posted":
		h.HandleAwardsPostedEvent(event.MessageData)
	case "schedule_updated":
		h.HandleEventScheduleUpdatedEvent(event.MessageData)
	case "ping":
		h.HandlePingEvent(event.MessageData)
	case "broadcast":
		h.HandleBroadcastEvent(event.MessageData)
	default:
		log.WarnNoContext("Unknown websocket event detected", "MessageType", event.MessageType, "Message", event.MessageData)
	}

	return nil
}

type MatchScoreNotification struct {
	EventKey  string        `json:"event_key"`
	MatchKey  string        `json:"match_key"`
	TeamKey   string        `json:"team_key"`
	EventName string        `json:"event_name"`
	Match     swagger.Match `json:"match"`
}

func (h *Handler) HandleMatchScoreEvent(messageData json.RawMessage) {
	log.InfoNoContext("Received match score event", "Message", messageData)
	var scoreNotification MatchScoreNotification
	err := json.Unmarshal(messageData, &scoreNotification)
	if err != nil {
		log.WarnNoContext("Failed to decode match score notification", "Error", err, "Message", messageData)
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

func (h *Handler) HandleAllianceSelectionEvent(messageData json.RawMessage) {
	log.InfoNoContext("Received alliance selection event", "Message", messageData)
	var notification AllianceSelectionNotification
	err := json.Unmarshal(messageData, &notification)
	if err != nil {
		log.WarnNoContext("Failed to decode alliance selection notification", "Error", err, "Message", messageData)
		return
	}
	h.Scorer.ScoreAllianceSelection(notification.EventKey)
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

func (h *Handler) HandleUpcomingMatchEvent(messageData json.RawMessage) {
	var tbaEvent UpcomingMatchEvent
	if err := json.Unmarshal(messageData, &tbaEvent); err != nil {
		log.WarnNoContext("Failed to decode upcoming match event data", "Error", err)
		return
	}

	if len(tbaEvent.TeamKeys) != 6 {
		log.WarnNoContext("Upcoming match received without 6 teams", "TeamCount", len(tbaEvent.TeamKeys))
		return
	}

	rows, err := model.GetDraftPickRows(h.Database, tbaEvent.TeamKeys)

	if err != nil {
		log.WarnNoContext("Failed to get picked rows", "Error", err)
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
			log.InfoNoContext("Posting pre match notification webhook")
			err := h.DiscordBus.PostPreMatchNotification(*event)
			if err != nil {
				log.ErrorNoContext("Failed to post pre match notification webhook", "Error", err)
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

func (h *Handler) HandleMatchVideoEvent(messageData json.RawMessage) {
	log.InfoNoContext("Received match video event", "Message", messageData)
}

type CompLevelStartingEvent struct {
	EventKey      string `json:"event_key"`
	EventName     string `json:"event_name"`
	CompLevel     string `json:"comp_level"`
	ScheduledTime string `json:"scheduled_time"`
}

func (h *Handler) HandleCompLevelStartingEvent(messageData json.RawMessage) {
	log.InfoNoContext("Received comp level starting event", "Message", messageData)
}

type AwardsPostedEvent struct {
	EventKey  string          `json:"event_key"`
	TeamKey   string          `json:"team_key"`
	EventName string          `json:"event_name"`
	Awards    []swagger.Award `json:"awards"`
}

func (h *Handler) HandleAwardsPostedEvent(messageData json.RawMessage) {
	log.InfoNoContext("Received awards posted event", "Message", messageData)
}

type EventScheduleUpdatedEvent struct {
	EventKey       string `json:"event_key"`
	EventName      string `json:"event_name"`
	FirstMatchTime int64  `json:"first_match_time"`
}

func (h *Handler) HandleEventScheduleUpdatedEvent(messageData json.RawMessage) {
	log.InfoNoContext("Received event schedule updated event", "Message", messageData)
}

type PingEvent struct {
	Title       string `json:"title"`
	Description string `json:"desc"`
}

func (h *Handler) HandlePingEvent(messageData json.RawMessage) {
	log.InfoNoContext("Received ping event", "Message", messageData)
}

type BroadcastEvent struct {
	Title       string `json:"title"`
	Description string `json:"desc"`
	Url         string `json:"url"`
}

func (h *Handler) HandleBroadcastEvent(messageData json.RawMessage) {
	log.InfoNoContext("Received broadcast event", "Message", messageData)
}

type VerificationEvent struct {
	VerificationKey string `json:"verification_key"`
}

func (h *Handler) HandleVerificationEvent(messageData json.RawMessage) {
	log.InfoNoContext("Received Verification Event", "Message", messageData)

	var event VerificationEvent
	err := json.Unmarshal(messageData, &event)
	if err != nil {
		log.WarnNoContext("Failed to decode verification event", "Error", err, "Message", messageData)
		return
	}

	h.TbaVerificationCode = event.VerificationKey

	// Only create the file if it doesn't already exist
	_, err = os.Stat(utils.GetWebhookFilePath())
	if os.IsNotExist(err) {
		err = os.WriteFile(utils.GetWebhookFilePath(), []byte(h.TbaVerificationCode), 0644)
		if err != nil {
			log.WarnNoContext("Failed to write tba webhook file body", "Error", err)
		}
	} else if err != nil {
		log.WarnNoContext("Failed to check if webhook file exists", "Error", err)
	}
}
