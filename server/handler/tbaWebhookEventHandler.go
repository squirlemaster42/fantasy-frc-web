package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"server/swagger"

	"github.com/labstack/echo/v4"
)

type TbaWebsocketEvent struct {
    MessageType string `json:"message_type"`
    MessageData string `json:"message_data"`
}

func validMAC(message []byte, messageMAC []byte, key []byte) bool {
	mac := hmac.New(sha256.New, key)
	mac.Write(message)
	expectedMAC := mac.Sum(nil)
    slog.Info("Validating HMAC", "HMAC", messageMAC, "Expected HMAC", hex.EncodeToString(expectedMAC))
	return hmac.Equal(messageMAC, []byte(hex.EncodeToString(expectedMAC)))
}

func (h *Handler) ConsumeTbaWebsocket(c echo.Context) error {
    slog.Info("Received webhook message")
    body, err := io.ReadAll(c.Request().Body)
    if err != nil {
        slog.Error("Failed to read request body", "Error", err)
        return nil
    }

    messageMac := c.Request().Header.Get("X-TBA-HMAC")
    valid := validMAC(body, []byte(messageMac), []byte(h.TbaWekhookSecret))

    if !valid {
        slog.Warn("Webhook event authentication failed", "Message", string(body))
        return nil
    }

    var event TbaWebsocketEvent
    err = json.NewDecoder(c.Request().Body).Decode(&event)
    if err != nil {
        slog.Error("Failed to decode webhook message", "Error", err, "Message", string(body))
        return nil
    }

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
    case "verification":
        h.HandleVerificationEvent(event.MessageData)
    default:
        slog.Warn("Unknown websocket event detected", "MessageType", event.MessageType, "Message", event.MessageData)
    }

    return nil
}

type MatchScoreNofification struct {
    EventKey string `json:"event_key"`
    MatchKey string `json:"match_key"`
    TeamKey string `json:"team_key"`
    EventName string `json:"event_name"`
    Match swagger.Match `json:"match"`
}

func (h *Handler) HandleMatchScoreEvent(messageData string) {
    slog.Info("Received match score event", "Message", messageData)
    var scoreNotification MatchScoreNofification
    err := json.Unmarshal([]byte(messageData), &scoreNotification)
    if err != nil {
        slog.Error("Failed to decode webhook message", "Error", err, "Message", messageData)
        return
    }

    h.Scorer.AddMatchToScore(scoreNotification.Match)
}

type AllianceSelectionNotification struct {
    EventKey string `json:"event_key"`
    TeamKey string `json:"team_key"`
    EventName string `json:"event_name"`
    Event swagger.Event `json:"event"`

}

func (h *Handler) HandleAllianceSelectionEvent(messageData string) {
    slog.Info("Received alliance selection event", "Message", messageData)
    var notification AllianceSelectionNotification
    err := json.Unmarshal([]byte(messageData), &notification)
    if err != nil {
        slog.Error("Failed to decode webhook message", "Error", err, "Message", messageData)
    }

    h.Scorer.ScoreAllianceSelection(notification.EventKey)
}

type UpcomingMatchEvent struct {
    EventKey string `json:"event_key"`
    MatchKey string `json:"match_key"`
    TeamKey string `json:"team_key"`
    EventName string `json:"event_name"`
    TeamKeys []string `json:"team_keys"`
    SchedulesTime int64 `json:"scheduled_time"`
    PredictedTime int64 `json:"predicted_time"`
    Webcast swagger.Webcast `json:"webcast"`
}

func (h *Handler) HandleUpcomingMatchEvent(messageData string) {
    slog.Info("Received upcoming match event", "Message", messageData)
}

type MatchVideoNofification struct {
    EventKey string `json:"event_key"`
    MatchKey string `json:"match_key"`
    TeamKey string `json:"team_key"`
    EventName string `json:"event_name"`
    Match swagger.Match `json:"match"`
}

func (h *Handler) HandleMatchVideoEvent(messageData string) {
    slog.Info("Received match video event", "Message", messageData)
}

type CompLevelStartingEvent struct {
    EventKey string `json:"event_key"`
    EventName string `json:"event_name"`
    CompLevel string `json:"comp_level"`
    ScheduledTime string `json:"scheduled_time"`
}

func (h *Handler) HandleCompLevelStartingEvent(messageData string) {
    slog.Info("Received comp level starting event", "Message", messageData)
}

type AwardsPostedEvent struct {
    EventKey string `json:"event_key"`
    TeamKey string `json:"team_key"`
    EventName string `json:"event_name"`
    Awards []swagger.Award `json:"awards"`
}

func (h *Handler) HandleAwardsPostedEvent(messageData string) {
    slog.Info("Received awards posted event", "Message", messageData)
}

type EventScheduleUpdatedEvent struct {
    EventKey string `json:"event_key"`
    EventName string `json:"event_name"`
    FirstMatchTime int64 `json:"first_match_time"`
}

func (h *Handler) HandleEventScheduleUpdatedEvent(messageData string) {
    slog.Info("Received event schedule updated event", "Message", messageData)
}

type PingEvent struct {
    Title string `json:"title"`
    Description string `json:"desc"`
}

func (h *Handler) HandlePingEvent(messageData string) {
    slog.Info("Received ping event", "Message", messageData)
}

type BroadcastEvent struct {
    Title string `json:"title"`
    Description string `json:"desc"`
    Url string `json:"url"`
}

func (h *Handler) HandleBroadcastEvent(messageData string) {
    slog.Info("Received broadcast event", "Message", messageData)
}

type VerificationEvent struct {
    VerificationKey string `json:"verification_key"`
}

func (h *Handler) HandleVerificationEvent(messageData string) {
    slog.Info("Received Verification Event", "Message", messageData)
}
