package tbaHandler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)


func getTbaTok(t *testing.T) string {
    err := godotenv.Load(filepath.Join("../", ".env"))
    if err != nil {
        t.Skipf("Skipping test: failed to load .env file %v", err)
    }
    token := os.Getenv("TBA_TOKEN")
    if token == "" {
        t.Skip("Skipping test: TBA_TOKEN not found in environment")
    }
    return token
}

func TestMatchListReq(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    matches := handler.MakeMatchListReq(t.Context(), "frc254", "2026casnv")
    assert.True(t, len(matches) > 0, "No matches were found")
    firstMatch := matches[0]
    if (firstMatch.EventKey != "2026casnv") {
        t.Fatalf("Match Key Incorrect")
    }

    if (firstMatch.ScoreBreakdown.Blue.TotalTeleopPoints == 0) {
        t.Fatalf("Score is not set")
    }
}

func TestEventListReq(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    events := handler.MakeEventListReq(t.Context(), "frc1690")
    if (len(events) == 0) {
        t.Fatalf("No events were found")
    }
}

func TestMatchReq(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    match := handler.MakeMatchReq(t.Context(), "2026casnv_qm24")
    if (match.ScoreBreakdown.Blue.TotalTeleopPoints == 0) {
        t.Fatalf("Score not set correctly")
    }
}

func TestMatchKeysRequest(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    keys := handler.MakeMatchKeysRequest(t.Context(), "frc1690", "2024isde1")
    if (len(keys) == 0) {
        t.Fatalf("No match keys found")
    }
}

func TestMatchKeysYearRequest(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    keys := handler.MakeMatchKeysYearRequest(t.Context(), "frc1690")
    if (len(keys) == 0) {
        t.Fatalf("No match keys found")
    }
}

func TestTeamEventStatusRequest(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    event := handler.MakeTeamEventStatusRequest(t.Context(), "frc1690", "2024isde1")
    if (event.LastMatchKey == "") {
        t.Fatalf("There should be a last match")
    }
}

type mockAllianceTransport struct {
	requestCount     int
	emptyResponses   int
	responseBody     string
}

func (m *mockAllianceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	m.requestCount++
	body := m.responseBody
	if m.requestCount <= m.emptyResponses {
		body = "[]"
	}
	headers := make(http.Header)
	headers.Set("Etag", "mock-etag")
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     headers,
		Request:    req,
	}, nil
}

func TestEliminationAllianceRequestRetrySuccess(t *testing.T) {
	validAlliance := `[{"name":"Alliance 1","picks":["frc254","frc1690"]},{"name":"Alliance 2","picks":["frc1114","frc2056"]}]`
	mock := &mockAllianceTransport{
		emptyResponses: 2,
		responseBody:   validAlliance,
	}

	handler := NewHandler("test-token", nil)
	handler.client = &http.Client{Transport: mock}

	start := time.Now()
	alliances := handler.MakeEliminationAllianceRequest(t.Context(), "2026test")
	elapsed := time.Since(start)

	assert.Equal(t, 3, mock.requestCount, "Expected 3 requests (2 empty + 1 success)")
	assert.Equal(t, 2, len(alliances), "Expected 2 alliances after retry")
	assert.Equal(t, "Alliance 1", alliances[0].Name)
	assert.Equal(t, "frc254", alliances[0].Picks[0])

	// Exponential backoff: 1s + 2s = 3s minimum
	assert.True(t, elapsed >= 3*time.Second, "Expected at least 3s of backoff delay")
}

func TestEliminationAllianceRequestRetryExhausted(t *testing.T) {
	mock := &mockAllianceTransport{
		emptyResponses: 10,
		responseBody:   "[]",
	}

	handler := NewHandler("test-token", nil)
	handler.client = &http.Client{Transport: mock}

	start := time.Now()
	alliances := handler.MakeEliminationAllianceRequest(t.Context(), "2026test")
	elapsed := time.Since(start)

	assert.Equal(t, 6, mock.requestCount, "Expected 6 requests (initial + 5 retries)")
	assert.Equal(t, 0, len(alliances), "Expected empty alliances after all retries exhausted")

	// Exponential backoff: 1s + 2s + 4s + 8s + 16s = 31s minimum
	assert.True(t, elapsed >= 31*time.Second, "Expected at least 31s of backoff delay")
}
