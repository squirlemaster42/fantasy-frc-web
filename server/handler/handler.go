package handler

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"server/background"
	"server/draft"
	"server/scorer"
	"server/swagger"
	"server/tbaHandler"

	"github.com/labstack/echo/v4"
)

type Handler struct {
    Database *sql.DB
    TbaHandler tbaHandler.TbaHandler
    DraftManager *draft.DraftManager
    DraftDaemon *background.DraftDaemon
    Scorer *scorer.Scorer
}

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
        break
    case "match_score":
        h.ScoreMatch(event.MessageData)
        break
    case "match_video":
        break
    case "starting_comp_level":
        break
    case "alliance_selection":
        h.ScoreAllianceSelection(event.MessageData)
        break
    case "awards_posted":
        break
    case "schedule_updated":
        break
    case "ping":
        break
    case "broadcast":
        break
    case "verification":
        break
    default:
        slog.Warn("Unknown websocket event detected", "MessageType", event.MessageType)
        break
    }

    return nil
}

type ScoreNofification struct {
    EventKey string `json:"event_key"`
    MatchKey string `json:"match_key"`
    TeamKey string `json:"team_key"`
    EventName string `json:"event_name"`
    Match swagger.Match `json:"match"`
}

func (h *Handler) ScoreMatch(messageData string) {
    var scoreNotification ScoreNofification
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

func (h *Handler) ScoreAllianceSelection(messageData string) {
    var notification AllianceSelectionNotification
    err := json.Unmarshal([]byte(messageData), &notification)
    if err != nil {
        slog.Error("Failed to decode webhook message", "Error", err, "Message", messageData)
    }

    h.ScoreAllianceSelection(notification.EventKey)
}
