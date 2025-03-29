package tbaHandler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"server/swagger"
	"sync"

	"github.com/labstack/echo/v4"
)

type WebhookClient struct {
    mu sync.Mutex
    running bool
    messages chan swagger.Match
    server echo.Echo
}

func NewWebhookClient () *WebhookClient {
    client := &WebhookClient{
        running: false,
        messages: make(chan swagger.Match, 10),
        server: *echo.New(),
    }
    client.server.POST("/recieveTbaWebhook", client.HandleWebhook)
    return client
}

func (w *WebhookClient) Start() error {
    w.mu.Lock()
    defer w.mu.Unlock()
    if !w.running {
        w.running = true
        err := w.server.Start(":4000")
        slog.Error("Failed to start webhook client", "Error", err)
        return nil
    } else {
        return errors.New("Cannot start a draft daemon that has already been started")
    }
}

func (w *WebhookClient) HandleWebhook(c echo.Context) error {
    _, err := io.ReadAll(c.Request().Body)
    if err != nil {
        slog.Error("Unable to read websocket body", "Error", err)
    }

    w.messages <- swagger.Match{}

    //TODO we need to verify the web request
    //TODO is there any content we need here?
    return c.String(http.StatusOK, "")
}

func (w *WebhookClient) RequestMessage() swagger.Match {
    return <- w.messages
}

func (w *WebhookClient) IsRunning() bool {
    return w.running
}

func (w *WebhookClient) Stop() error {
    w.running = false
    return nil
}
