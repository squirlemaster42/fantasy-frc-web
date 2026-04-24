package handler

import (
	"encoding/json"
	"net/http"
	"server/log"
	"time"

	"github.com/labstack/echo/v4"
)

var lastLogsRequest time.Time
var logsRequestCount int

func (h *Handler) HandleLogs(c echo.Context) error {
	now := time.Now()
	if now.Sub(lastLogsRequest) < time.Minute {
		logsRequestCount++
		if logsRequestCount > 10 {
			return c.NoContent(http.StatusTooManyRequests)
		}
	} else {
		logsRequestCount = 0
	}
	lastLogsRequest = now

	log.Info(c.Request().Context(), "logs endpoint accessed", "ip", c.RealIP())

	lastSeenStr := c.QueryParam("last_seen")
	var lastSeen time.Time

	if lastSeenStr != "" {
		parsed, err := time.Parse(time.RFC3339, lastSeenStr)
		if err != nil {
			return c.NoContent(http.StatusBadRequest)
		}
		lastSeen = parsed
		if time.Since(lastSeen) > time.Hour {
			return c.NoContent(http.StatusBadRequest)
		}
	}

	buffer := log.GetBuffer()
	var entries []log.LogEntry
	if lastSeen.IsZero() {
		entries = buffer.GetAll()
	} else {
		entries = buffer.GetSince(lastSeen)
	}

	if len(entries) > 100 {
		entries = entries[len(entries)-100:]
	}

	c.Response().Header().Set("Content-Type", "application/x-ndjson")
	for _, e := range entries {
		entry := map[string]any{
			"time":  e.Time.Format(time.RFC3339),
			"level": e.Level.String(),
			"msg":   e.Msg,
		}
		for k, v := range e.Attrs {
			entry[k] = v
		}

		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		c.Response().Write(data)
		c.Response().Write([]byte("\n"))
	}

	return nil
}
