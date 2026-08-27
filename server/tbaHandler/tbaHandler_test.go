package tbaHandler

import (
	"database/sql"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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
    matches, err := handler.MakeMatchListReq(t.Context(), "frc254", "2026casnv")
    assert.NoError(t, err)
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
    events, err := handler.MakeEventListReq(t.Context(), "frc1690")
    if err != nil {
        t.Fatalf("Failed to get event list: %v", err)
    }
    if (len(events) == 0) {
        t.Fatalf("No events were found")
    }
}

func TestMatchReq(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    match, err := handler.MakeMatchReq(t.Context(), "2026casnv_qm24")
    if err != nil {
        t.Fatalf("Failed to get match: %v", err)
    }
    if (match.ScoreBreakdown.Blue.TotalTeleopPoints == 0) {
        t.Fatalf("Score not set correctly")
    }
}

func TestMatchKeysRequest(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    keys, err := handler.MakeMatchKeysRequest(t.Context(), "frc1690", "2024isde1")
    if err != nil {
        t.Fatalf("Failed to get match keys: %v", err)
    }
    if (len(keys) == 0) {
        t.Fatalf("No match keys found")
    }
}

func TestMatchKeysYearRequest(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    keys, err := handler.MakeMatchKeysYearRequest(t.Context(), "frc1690")
    if err != nil {
        t.Fatalf("Failed to get match keys by year: %v", err)
    }
    if (len(keys) == 0) {
        t.Fatalf("No match keys found")
    }
}

func TestTeamEventStatusRequest(t *testing.T) {
    tbaTok := getTbaTok(t)
    assert.True(t, len(tbaTok) > 0, "TBA Token was not loaded correctly")
    handler := NewHandler(tbaTok, nil)
    event, err := handler.MakeTeamEventStatusRequest(t.Context(), "frc1690", "2024isde1")
    if err != nil {
        t.Fatalf("Failed to get team event status: %v", err)
    }
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
	alliances, err := handler.MakeEliminationAllianceRequest(t.Context(), "2026test")
	elapsed := time.Since(start)

	assert.NoError(t, err)
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
	alliances, err := handler.MakeEliminationAllianceRequest(t.Context(), "2026test")
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 6, mock.requestCount, "Expected 6 requests (initial + 5 retries)")
	assert.Equal(t, 0, len(alliances), "Expected empty alliances after all retries exhausted")

	// Exponential backoff: 1s + 2s + 4s + 8s + 16s = 31s minimum
	assert.True(t, elapsed >= 31*time.Second, "Expected at least 31s of backoff delay")
}

type mockTransport struct {
	responses []mockResponse
	index     int
}

type mockResponse struct {
	statusCode int
	body       string
	etag       string
	err        error
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.index >= len(m.responses) {
		return nil, errors.New("no more mocked responses")
	}
	r := m.responses[m.index]
	m.index++
	if r.err != nil {
		return nil, r.err
	}
	headers := make(http.Header)
	if r.etag != "" {
		headers.Set("Etag", r.etag)
	}
	return &http.Response{
		StatusCode: r.statusCode,
		Body:       io.NopCloser(strings.NewReader(r.body)),
		Header:     headers,
		Request:    req,
	}, nil
}

func TestMakeRequest_CacheMiss200(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	url := "https://www.thebluealliance.com/api/v3/test"
	etag := "test-etag"
	body := `[{"key":"value"}]`

	mock.ExpectPrepare(`Select etag, responseBody From TbaCache Where url = \$1;`).
		ExpectQuery().WithArgs(url).WillReturnError(sql.ErrNoRows)
	mock.ExpectPrepare(`Insert Into TbaCache \(url, etag, responseBody\) Values \(\$1, \$2, \$3\) On Conflict \(url\) Do Update Set etag = excluded\.etag, responseBody = excluded\.responseBody;`).
		ExpectExec().WithArgs(url, etag, []byte(body)).WillReturnResult(sqlmock.NewResult(1, 1))

	handler := NewHandler("test-token", db)
	handler.client = &http.Client{Transport: &mockTransport{responses: []mockResponse{{statusCode: http.StatusOK, body: body, etag: etag}}}}

	resp, err := handler.makeRequest(t.Context(), url, "/test")
	assert.NoError(t, err)
	assert.Equal(t, []byte(body), resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMakeRequest_CacheHitNotModified(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	url := "https://www.thebluealliance.com/api/v3/test"
	etag := "test-etag"
	cachedBody := []byte(`[{"cached":true}]`)

	mock.ExpectPrepare(`Select etag, responseBody From TbaCache Where url = \$1;`).
		ExpectQuery().WithArgs(url).WillReturnRows(sqlmock.NewRows([]string{"etag", "responseBody"}).AddRow(etag, cachedBody))

	handler := NewHandler("test-token", db)
	handler.client = &http.Client{Transport: &mockTransport{responses: []mockResponse{{statusCode: http.StatusNotModified, body: ""}}}}

	resp, err := handler.makeRequest(t.Context(), url, "/test")
	assert.NoError(t, err)
	assert.Equal(t, cachedBody, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMakeRequest_404ReturnsNil(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	url := "https://www.thebluealliance.com/api/v3/test"

	mock.ExpectPrepare(`Select etag, responseBody From TbaCache Where url = \$1;`).
		ExpectQuery().WithArgs(url).WillReturnError(sql.ErrNoRows)

	handler := NewHandler("test-token", db)
	handler.client = &http.Client{Transport: &mockTransport{responses: []mockResponse{{statusCode: http.StatusNotFound, body: "not found"}}}}

	resp, err := handler.makeRequest(t.Context(), url, "/test")
	assert.NoError(t, err)
	assert.Nil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMakeRequest_500ReturnsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	url := "https://www.thebluealliance.com/api/v3/test"

	mock.ExpectPrepare(`Select etag, responseBody From TbaCache Where url = \$1;`).
		ExpectQuery().WithArgs(url).WillReturnError(sql.ErrNoRows)

	handler := NewHandler("test-token", db)
	handler.client = &http.Client{Transport: &mockTransport{responses: []mockResponse{{statusCode: http.StatusInternalServerError, body: "server error"}}}}

	resp, err := handler.makeRequest(t.Context(), url, "/test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server error")
	assert.Nil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMakeRequest_429ReturnsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	url := "https://www.thebluealliance.com/api/v3/test"

	mock.ExpectPrepare(`Select etag, responseBody From TbaCache Where url = \$1;`).
		ExpectQuery().WithArgs(url).WillReturnError(sql.ErrNoRows)

	handler := NewHandler("test-token", db)
	handler.client = &http.Client{Transport: &mockTransport{responses: []mockResponse{{statusCode: http.StatusTooManyRequests, body: "rate limited"}}}}

	resp, err := handler.makeRequest(t.Context(), url, "/test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rate limit")
	assert.Nil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMakeRequest_NetworkError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	url := "https://www.thebluealliance.com/api/v3/test"

	mock.ExpectPrepare(`Select etag, responseBody From TbaCache Where url = \$1;`).
		ExpectQuery().WithArgs(url).WillReturnError(sql.ErrNoRows)

	handler := NewHandler("test-token", db)
	handler.client = &http.Client{Transport: &mockTransport{responses: []mockResponse{{err: errors.New("connection refused")}}}}

	resp, err := handler.makeRequest(t.Context(), url, "/test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
	assert.Nil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMakeRequest_UnexpectedStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer func() { _ = db.Close() }()

	url := "https://www.thebluealliance.com/api/v3/test"

	mock.ExpectPrepare(`Select etag, responseBody From TbaCache Where url = \$1;`).
		ExpectQuery().WithArgs(url).WillReturnError(sql.ErrNoRows)

	handler := NewHandler("test-token", db)
	handler.client = &http.Client{Transport: &mockTransport{responses: []mockResponse{{statusCode: http.StatusBadRequest, body: "bad request"}}}}

	resp, err := handler.makeRequest(t.Context(), url, "/test")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status")
	assert.Nil(t, resp)
	assert.NoError(t, mock.ExpectationsWereMet())
}
