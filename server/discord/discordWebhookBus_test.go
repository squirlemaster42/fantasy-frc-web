package discord

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBus(t *testing.T) {
	bus := NewBus()
	defer bus.Stop()

	assert.NotNil(t, bus)
	assert.NotNil(t, bus.client)
	assert.NotNil(t, bus.preMatchCh)
	assert.NotNil(t, bus.stopCh)
}

func TestPostPreMatchNotification_Success(t *testing.T) {
	received := make(chan DiscordWebhook, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var webhook DiscordWebhook
		err = json.Unmarshal(body, &webhook)
		assert.NoError(t, err)

		received <- webhook
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	bus := NewBus()
	defer bus.Stop()

	event := PreMatchDiscordEvent{
		EventName:     "Test Event",
		PredictedTime: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		IdsToTeams: map[string][]string{
			"<@123>": {"frc254", "frc118"},
		},
		Webhook: server.URL,
	}

	err := bus.PostPreMatchNotification(event)
	assert.NoError(t, err)

	select {
	case webhook := <-received:
		assert.Equal(t, "Match Notifier", webhook.Username)
		assert.Contains(t, webhook.Content, "Test Event")
		assert.Contains(t, webhook.Content, "254")
		assert.Contains(t, webhook.Content, "118")
		assert.Equal(t, []string{"users"}, webhook.AllowedMentions.Parse)
	case <-time.After(2 * time.Second):
		t.Fatal("expected webhook to be received")
	}
}

func TestPostPreMatchNotification_QueueFull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	bus := NewBus()
	defer bus.Stop()

	// Fill the channel to capacity
	for i := 0; i < cap(bus.preMatchCh); i++ {
		bus.preMatchCh <- PreMatchDiscordEvent{EventName: "fill", Webhook: server.URL}
	}

	err := bus.PostPreMatchNotification(PreMatchDiscordEvent{EventName: "overflow", Webhook: server.URL})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "queue is full")
}

func TestPostPreMatchNotification_HandlesNon2xxResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer server.Close()

	bus := NewBus()
	defer bus.Stop()

	event := PreMatchDiscordEvent{
		EventName:     "Test Event",
		PredictedTime: time.Now(),
		Webhook:       server.URL,
	}

	err := bus.PostPreMatchNotification(event)
	assert.NoError(t, err)

	// Give the worker time to process and log the non-2xx response
	time.Sleep(100 * time.Millisecond)
}

func TestPostPickNotification_DraftComplete(t *testing.T) {
	received := make(chan DiscordWebhook, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var webhook DiscordWebhook
		err = json.Unmarshal(body, &webhook)
		assert.NoError(t, err)

		received <- webhook
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	bus := NewBus()
	defer bus.Stop()

	event := NextPickDiscordEvent{
		PreviousPickedTeam: "frc254",
		PreviousPickName:   "Alice",
		Webhook:            server.URL,
		DraftComplete:      true,
	}

	err := bus.PostPickNotification(event)
	assert.NoError(t, err)

	select {
	case webhook := <-received:
		assert.Equal(t, "Pick Notifier", webhook.Username)
		assert.Contains(t, webhook.Content, "Alice")
		assert.Contains(t, webhook.Content, "254")
		assert.Contains(t, webhook.Content, "draft is complete")
	case <-time.After(time.Second):
		t.Fatal("expected webhook to be received")
	}
}

func TestPostPickNotification_NextPickWithMention(t *testing.T) {
	received := make(chan DiscordWebhook, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var webhook DiscordWebhook
		err = json.Unmarshal(body, &webhook)
		assert.NoError(t, err)

		received <- webhook
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	bus := NewBus()
	defer bus.Stop()

	event := NextPickDiscordEvent{
		PreviousPickedTeam: "frc254",
		PreviousPickName:   "Alice",
		PreviousPickDiscordId: sql.NullString{String: "12345678901234567", Valid: true},
		NextPickName:          "Bob",
		NextPickDiscordId:     sql.NullString{String: "98765432109876543", Valid: true},
		Webhook:               server.URL,
		ExpirationTime:        time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	err := bus.PostPickNotification(event)
	assert.NoError(t, err)

	select {
	case webhook := <-received:
		assert.Contains(t, webhook.Content, "<@12345678901234567>")
		assert.Contains(t, webhook.Content, "<@98765432109876543>")
		assert.Equal(t, []string{"98765432109876543"}, webhook.AllowedMentions.Users)
	case <-time.After(time.Second):
		t.Fatal("expected webhook to be received")
	}
}

func TestPostPickNotification_InvalidDiscordIdIgnored(t *testing.T) {
	received := make(chan DiscordWebhook, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var webhook DiscordWebhook
		err = json.Unmarshal(body, &webhook)
		assert.NoError(t, err)

		received <- webhook
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	bus := NewBus()
	defer bus.Stop()

	event := NextPickDiscordEvent{
		PreviousPickedTeam:    "frc254",
		PreviousPickName:      "Alice",
		PreviousPickDiscordId: sql.NullString{String: "not-a-number", Valid: true},
		NextPickName:          "Bob",
		NextPickDiscordId:     sql.NullString{String: "123", Valid: true},
		Webhook:               server.URL,
		ExpirationTime:        time.Now(),
	}

	err := bus.PostPickNotification(event)
	assert.NoError(t, err)

	select {
	case webhook := <-received:
		assert.NotContains(t, webhook.Content, "<@not-a-number>")
		assert.NotContains(t, webhook.Content, "<@123>")
		assert.Contains(t, webhook.Content, "Alice")
		assert.Contains(t, webhook.Content, "Bob")
	case <-time.After(time.Second):
		t.Fatal("expected webhook to be received")
	}
}

func TestPostPickNotification_Non2xxResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid webhook"))
	}))
	defer server.Close()

	bus := NewBus()
	defer bus.Stop()

	event := NextPickDiscordEvent{
		PreviousPickedTeam: "frc254",
		PreviousPickName:   "Alice",
		Webhook:            server.URL,
		ExpirationTime:     time.Now(),
	}

	err := bus.PostPickNotification(event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid webhook")
}

func TestPostPickNotification_ConsecutivePicksMessage(t *testing.T) {
	received := make(chan DiscordWebhook, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)

		var webhook DiscordWebhook
		err = json.Unmarshal(body, &webhook)
		assert.NoError(t, err)

		received <- webhook
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	bus := NewBus()
	defer bus.Stop()

	event := NextPickDiscordEvent{
		PreviousPickedTeam: "frc254",
		PreviousPickName:   "Alice",
		NextPickName:       "Alice",
		Webhook:            server.URL,
		ExpirationTime:     time.Now(),
	}

	err := bus.PostPickNotification(event)
	assert.NoError(t, err)

	select {
	case webhook := <-received:
		assert.Contains(t, webhook.Content, "it is your turn again")
	case <-time.After(time.Second):
		t.Fatal("expected webhook to be received")
	}
}
