package handler

import (
	"encoding/json"
	"log/slog"
	"server/swagger"

	"github.com/labstack/echo/v4"
)

type TbaWebsocketEvent struct {
    MessageType string `json:"message_type"`
    MessageData string `json:"message_data"`
}

func (h *Handler) ConsumeTbaWebsocket(c echo.Context) error {
    //TODO we need to authenticate the requset
    var event TbaWebsocketEvent
    err := json.NewDecoder(c.Request().Body).Decode(&event)
    if err != nil {
        //TODO it would be nice to get the request body in here
        slog.Error("Failed to decode webhook message", "Error", err)
    }

    switch event.MessageType {
    case "upcoming_match":
        h.HandleUpcomingMatchEvent(event.MessageData)
        break
    case "match_score":
        h.HandleMatchScoreEvent(event.MessageData)
        break
    case "match_video":
        h.HandleMatchVideoEvent(event.MessageData)
        break
    case "starting_comp_level":
        h.HandleCompLevelStartingEvent(event.MessageData)
        break
    case "alliance_selection":
        h.HandleAllianceSelectionEvent(event.MessageData)
        break
    case "awards_posted":
        h.HandleAwardsPostedEvent(event.MessageData)
        break
    case "schedule_updated":
        h.HandleEventScheduleUpdatedEvent(event.MessageData)
        break
    case "ping":
        h.HandlePingEvent(event.MessageData)
        break
    case "broadcast":
        h.HandleBroadcastEvent(event.MessageData)
        break
    case "verification":
        h.HandleVerificationEvent(event.MessageData)
        break
    default:
        slog.Warn("Unknown websocket event detected", "MessageType", event.MessageType)
        break
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

}


type MatchVideoNofification struct {
    EventKey string `json:"event_key"`
    MatchKey string `json:"match_key"`
    TeamKey string `json:"team_key"`
    EventName string `json:"event_name"`
    Match swagger.Match `json:"match"`
}

func (h *Handler) HandleMatchVideoEvent(messageData string) {

}

type CompLevelStartingEvent struct {
    EventKey string `json:"event_key"`
    EventName string `json:"event_name"`
    CompLevel string `json:"comp_level"`
    ScheduledTime string `json:"scheduled_time"`
}

func (h *Handler) HandleCompLevelStartingEvent(messageData string) {

}

type AwardsPostedEvent struct {
    EventKey string `json:"event_key"`
    TeamKey string `json:"team_key"`
    EventName string `json:"event_name"`
    Awards []swagger.Award `json:"awards"`
}

func (h *Handler) HandleAwardsPostedEvent(messageData string) {

}

type EventScheduleUpdatedEvent struct {
    EventKey string `json:"event_key"`
    EventName string `json:"event_name"`
    FirstMatchTime int64 `json:"first_match_time"`
}

func (h *Handler) HandleEventScheduleUpdatedEvent(messageData string) {

}

type PingEvent struct {
    Title string `json:"title"`
    Description string `json:"desc"`
}

func (h *Handler) HandlePingEvent(messageData string) {

}

type BroadcastEvent struct {
    Title string `json:"title"`
    Description string `json:"desc"`
    Url string `json:"url"`
}

func (h *Handler) HandleBroadcastEvent(messageData string) {

}

type VerificationEvent struct {
    VerificationKey string `json:"verification_key"`
}

func (h *Handler) HandleVerificationEvent(messageData string) {

}
