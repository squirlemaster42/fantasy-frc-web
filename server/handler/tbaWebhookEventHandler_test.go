package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"server/discord"
	"server/model"
	"server/model/mocks"
	"server/scorer"
	"server/swagger"
)

func computeTbaHMAC(t *testing.T, secret string, body []byte) string {
	t.Helper()
	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write(body)
	require.NoError(t, err)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestValidMAC(t *testing.T) {
	secret := "shhh"
	body := []byte("hello webhook")
	mac := computeTbaHMAC(t, secret, body)

	assert.True(t, validMAC(body, mac, []byte(secret)))
	assert.False(t, validMAC(body, mac, []byte("wrong")))
	assert.False(t, validMAC(body, "not-hex", []byte(secret)))
}

func TestConsumeTbaWebhook_InvalidHMAC(t *testing.T) {
	_, c, rec := setupTestContext(t, http.MethodPost, "/tba/webhook", `{"message_type":"ping"}`, "")
	c.Request().Header.Set("X-TBA-HMAC", "bad-mac")

	h := &Handler{Config: ConfigGroup{TbaWebhookSecret: "secret"}}
	err := h.ConsumeTbaWebhook(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestConsumeTbaWebhook_MalformedJSON(t *testing.T) {
	body := []byte(`not json`)
	_, c, rec := setupTestContext(t, http.MethodPost, "/tba/webhook", string(body), "")
	c.Request().Header.Set("X-TBA-HMAC", computeTbaHMAC(t, "secret", body))

	h := &Handler{Config: ConfigGroup{TbaWebhookSecret: "secret"}}
	err := h.ConsumeTbaWebhook(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestConsumeTbaWebhook_PayloadTooLarge(t *testing.T) {
	largeBody := strings.Repeat("a", int(TbaWebhookMaxBodyBytes())+1)
	_, c, rec := setupTestContext(t, http.MethodPost, "/tba/webhook", largeBody, "")

	h := &Handler{Config: ConfigGroup{TbaWebhookSecret: "secret"}}
	err := h.ConsumeTbaWebhook(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestConsumeTbaWebhook_Verification(t *testing.T) {
	payload := map[string]any{
		"message_type": "verification",
		"message_data": map[string]string{"verification_key": "verify-me"},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	_, c, rec := setupTestContext(t, http.MethodPost, "/tba/webhook", string(body), "")
	c.Request().Header.Set("X-TBA-HMAC", computeTbaHMAC(t, "secret", body))

	// Ensure the webhook secret file exists so the handler does not try to create it.
	secretPath := "./webhookSecret.txt" //nolint:gosec // test fixture path, not a credential
	require.NoError(t, os.WriteFile(secretPath, []byte("existing"), 0600))
	defer func() { _ = os.Remove(secretPath) }()

	h := &Handler{Config: ConfigGroup{TbaWebhookSecret: "secret"}}
	err = h.ConsumeTbaWebhook(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "verify-me", h.Config.TbaVerificationCode)
}

func TestConsumeTbaWebhook_Ping(t *testing.T) {
	payload := map[string]any{
		"message_type": "ping",
		"message_data": map[string]string{"title": "ping", "desc": "pong"},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	_, c, rec := setupTestContext(t, http.MethodPost, "/tba/webhook", string(body), "")
	c.Request().Header.Set("X-TBA-HMAC", computeTbaHMAC(t, "secret", body))

	h := &Handler{Config: ConfigGroup{TbaWebhookSecret: "secret"}}
	err = h.ConsumeTbaWebhook(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestConsumeTbaWebhook_MatchScore(t *testing.T) {
	payload := map[string]any{
		"message_type": "match_score",
		"message_data": map[string]any{
			"event_key":  "2026arc",
			"match_key":  "2026arc_qm1",
			"team_key":   "frc254",
			"event_name": "Test Event",
			"match":      swagger.Match{Key: "2026arc_qm1"},
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	_, c, rec := setupTestContext(t, http.MethodPost, "/tba/webhook", string(body), "")
	c.Request().Header.Set("X-TBA-HMAC", computeTbaHMAC(t, "secret", body))

	h := &Handler{
		Config:   ConfigGroup{TbaWebhookSecret: "secret"},
		Services: ServiceGroup{Scorer: scorer.NewScorer(nil, nil, nil, nil, nil, nil)},
	}
	err = h.ConsumeTbaWebhook(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestConsumeTbaWebhook_AllianceSelection(t *testing.T) {
	payload := map[string]any{
		"message_type": "alliance_selection",
		"message_data": map[string]any{
			"event_key":  "2026arc",
			"team_key":   "frc254",
			"event_name": "Test Event",
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	_, c, rec := setupTestContext(t, http.MethodPost, "/tba/webhook", string(body), "")
	c.Request().Header.Set("X-TBA-HMAC", computeTbaHMAC(t, "secret", body))

	tbaHandler := &mockAdminTBAHandler{teams: map[string][]swagger.Team{}}
	h := &Handler{
		Config:   ConfigGroup{TbaWebhookSecret: "secret"},
		Services: ServiceGroup{Scorer: scorer.NewScorer(tbaHandler, nil, nil, nil, nil, nil)},
	}
	err = h.ConsumeTbaWebhook(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestConsumeTbaWebhook_UpcomingMatch(t *testing.T) {
	payload := map[string]any{
		"message_type": "upcoming_match",
		"message_data": map[string]any{
			"event_key":      "2026arc",
			"match_key":      "2026arc_qm1",
			"team_key":       "frc254",
			"event_name":     "Test Event",
			"team_keys":      []string{"frc1", "frc2", "frc3", "frc4", "frc5", "frc6"},
			"scheduled_time": time.Now().Unix(),
			"predicted_time": time.Now().Unix(),
		},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	done := make(chan struct{})
	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraftPickRows", mock.Anything, []string{"frc1", "frc2", "frc3", "frc4", "frc5", "frc6"}).Run(func(args mock.Arguments) {
		close(done)
	}).Return([]model.DraftPickRow{}, nil).Once()

	_, c, rec := setupTestContext(t, http.MethodPost, "/tba/webhook", string(body), "")
	c.Request().Header.Set("X-TBA-HMAC", computeTbaHMAC(t, "secret", body))

	h := &Handler{
		Config:   ConfigGroup{TbaWebhookSecret: "secret"},
		Stores:   StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{DiscordWebhookBus: &mockDiscordNotifier{}},
	}
	err = h.ConsumeTbaWebhook(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for upcoming match handler")
	}
}

func TestConsumeTbaWebhook_UnknownEvent(t *testing.T) {
	payload := map[string]any{
		"message_type": "custom_event",
		"message_data": map[string]string{},
	}
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	_, c, rec := setupTestContext(t, http.MethodPost, "/tba/webhook", string(body), "")
	c.Request().Header.Set("X-TBA-HMAC", computeTbaHMAC(t, "secret", body))

	h := &Handler{Config: ConfigGroup{TbaWebhookSecret: "secret"}}
	err = h.ConsumeTbaWebhook(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHandleMatchScoreEvent(t *testing.T) {
	s := scorer.NewScorer(nil, nil, nil, nil, nil, nil)
	h := &Handler{Services: ServiceGroup{Scorer: s}}

	payload := map[string]any{
		"event_key":  "2026arc",
		"match_key":  "2026arc_qm1",
		"team_key":   "frc254",
		"event_name": "Test Event",
		"match":      swagger.Match{Key: "2026arc_qm1"},
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	// Should not panic or error.
	h.HandleMatchScoreEvent(context.Background(), data)
}

func TestHandleMatchScoreEvent_InvalidJSON(t *testing.T) {
	s := scorer.NewScorer(nil, nil, nil, nil, nil, nil)
	h := &Handler{Services: ServiceGroup{Scorer: s}}

	h.HandleMatchScoreEvent(context.Background(), []byte(`not json`))
}

func TestHandleAllianceSelectionEvent(t *testing.T) {
	tbaHandler := &mockAdminTBAHandler{teams: map[string][]swagger.Team{}}
	s := scorer.NewScorer(tbaHandler, nil, nil, nil, nil, nil)
	h := &Handler{Services: ServiceGroup{Scorer: s}}

	payload := map[string]any{
		"event_key":  "2026arc",
		"team_key":   "frc254",
		"event_name": "Test Event",
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	h.HandleAllianceSelectionEvent(context.Background(), data)
}

func TestHandleAllianceSelectionEvent_InvalidJSON(t *testing.T) {
	h := &Handler{Services: ServiceGroup{Scorer: scorer.NewScorer(nil, nil, nil, nil, nil, nil)}}
	h.HandleAllianceSelectionEvent(context.Background(), []byte(`not json`))
}

type mockDiscordNotifier struct {
	preMatchEvent *discord.PreMatchDiscordEvent
}

func (m *mockDiscordNotifier) PostPreMatchNotification(event discord.PreMatchDiscordEvent) error {
	m.preMatchEvent = &event
	return nil
}

func (m *mockDiscordNotifier) PostPickNotification(_ context.Context, _ discord.NextPickDiscordEvent) error {
	return nil
}

func (m *mockDiscordNotifier) Stop() {}

func TestHandleUpcomingMatchEvent(t *testing.T) {
	mockDraftStore := mocks.NewMockDraftStore(t)
	mockDraftStore.On("GetDraftPickRows", mock.Anything, []string{"frc1", "frc2", "frc3", "frc4", "frc5", "frc6"}).Return([]model.DraftPickRow{
		{
			DraftId:            1,
			DraftName:          "Test Draft",
			Username:           "user",
			Pick:               "frc1",
			DiscordId:          sql.NullString{String: "12345", Valid: true},
			Webhook:            sql.NullString{String: "https://discord.com/webhook", Valid: true},
			WantsUpcomingMatch: true,
		},
	}, nil).Once()

	notifier := &mockDiscordNotifier{}
	h := &Handler{
		Stores:   StorageGroup{DraftStore: mockDraftStore},
		Services: ServiceGroup{DiscordWebhookBus: notifier},
	}

	payload := map[string]any{
		"event_key":      "2026arc",
		"match_key":      "2026arc_qm1",
		"team_key":       "frc1",
		"event_name":     "Test Event",
		"team_keys":      []string{"frc1", "frc2", "frc3", "frc4", "frc5", "frc6"},
		"scheduled_time": time.Now().Unix(),
		"predicted_time": time.Now().Unix(),
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	h.HandleUpcomingMatchEvent(context.Background(), data)

	assert.NotNil(t, notifier.preMatchEvent)
	assert.Equal(t, "Test Event", notifier.preMatchEvent.EventName)
	assert.Contains(t, notifier.preMatchEvent.IdsToTeams["user"], "frc1")
}

func TestHandleUpcomingMatchEvent_WrongTeamCount(t *testing.T) {
	h := &Handler{}

	payload := map[string]any{
		"event_key":      "2026arc",
		"match_key":      "2026arc_qm1",
		"team_key":       "frc1",
		"event_name":     "Test Event",
		"team_keys":      []string{"frc1", "frc2"},
		"scheduled_time": time.Now().Unix(),
		"predicted_time": time.Now().Unix(),
	}
	data, err := json.Marshal(payload)
	require.NoError(t, err)

	h.HandleUpcomingMatchEvent(context.Background(), data)
}

func TestHandleMatchVideoEvent(t *testing.T) {
	h := &Handler{}
	data, err := json.Marshal(map[string]string{"event_key": "2026arc", "match_key": "2026arc_qm1"})
	require.NoError(t, err)
	h.HandleMatchVideoEvent(context.Background(), data)
}

func TestHandleCompLevelStartingEvent(t *testing.T) {
	h := &Handler{}
	data, err := json.Marshal(map[string]string{"event_key": "2026arc", "comp_level": "qf"})
	require.NoError(t, err)
	h.HandleCompLevelStartingEvent(context.Background(), data)
}

func TestHandleAwardsPostedEvent(t *testing.T) {
	h := &Handler{}
	data, err := json.Marshal(map[string]any{"event_key": "2026arc", "awards": []swagger.Award{}})
	require.NoError(t, err)
	h.HandleAwardsPostedEvent(context.Background(), data)
}

func TestHandleEventScheduleUpdatedEvent(t *testing.T) {
	h := &Handler{}
	data, err := json.Marshal(map[string]any{"event_key": "2026arc", "first_match_time": time.Now().Unix()})
	require.NoError(t, err)
	h.HandleEventScheduleUpdatedEvent(context.Background(), data)
}

func TestHandlePingEvent(t *testing.T) {
	h := &Handler{}
	data, err := json.Marshal(map[string]string{"title": "ping", "desc": "pong"})
	require.NoError(t, err)
	h.HandlePingEvent(context.Background(), data)
}

func TestHandleBroadcastEvent(t *testing.T) {
	h := &Handler{}
	data, err := json.Marshal(map[string]string{"title": "broadcast", "desc": "hello", "url": "https://example.com"})
	require.NoError(t, err)
	h.HandleBroadcastEvent(context.Background(), data)
}

func TestHandleVerificationEvent_InvalidJSON(t *testing.T) {
	h := &Handler{Config: ConfigGroup{}}
	h.HandleVerificationEvent(context.Background(), []byte(`not json`))
	assert.Empty(t, h.Config.TbaVerificationCode)
}

func TestHandleVerificationEvent_Success(t *testing.T) {
	secretPath := "./webhookSecret.txt" //nolint:gosec // test fixture path, not a credential
	require.NoError(t, os.WriteFile(secretPath, []byte("existing"), 0600))
	defer func() { _ = os.Remove(secretPath) }()

	h := &Handler{Config: ConfigGroup{}}
	data, err := json.Marshal(map[string]string{"verification_key": "verify-me"})
	require.NoError(t, err)
	h.HandleVerificationEvent(context.Background(), data)

	assert.Equal(t, "verify-me", h.Config.TbaVerificationCode)
}

func TestHandleVerificationEvent_WritesFileWhenMissing(t *testing.T) {
	secretPath := "./webhookSecret.txt" //nolint:gosec // test fixture path, not a credential
	_ = os.Remove(secretPath)
	defer func() { _ = os.Remove(secretPath) }()

	h := &Handler{Config: ConfigGroup{}}
	data, err := json.Marshal(map[string]string{"verification_key": "fresh-key"})
	require.NoError(t, err)
	h.HandleVerificationEvent(context.Background(), data)

	assert.Equal(t, "fresh-key", h.Config.TbaVerificationCode)
	written, err := os.ReadFile(secretPath)
	require.NoError(t, err)
	assert.Equal(t, "fresh-key", string(written))
}
