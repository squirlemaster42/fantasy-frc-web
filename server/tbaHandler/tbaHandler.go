package tbaHandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"server/assert"
	"server/log"
	"server/metrics"
	"server/swagger"
	"time"

	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	BASE_URL = "https://www.thebluealliance.com/api/v3/"
)

type TBAHandler struct {
	tbaToken string
	database *sql.DB
	client   *http.Client
}

func NewHandler(tbaToken string, database *sql.DB) *TBAHandler {
	handler := &TBAHandler{
		tbaToken: tbaToken,
		database: database,
		client: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
	return handler
}

func (t *TBAHandler) checkCache(ctx context.Context, url string) ([]byte, string, error) {
	assert := assert.CreateAssertWithContext("Check Tba Cache")
	assert.AddContext("Url", url)

	// Dont check the cache if we dont have a database
	// This is probably because we are running a unit test
	if t.database == nil {
		return nil, "", nil
	}

	query := `Select
        etag,
        responseBody
    From TbaCache
    Where url = $1;`
	stmt, err := t.database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "checkCache: Failed to close statement", "error", err)
		}
	}()

	var etag string
	var body []byte
	err = stmt.QueryRowContext(ctx, url).Scan(&etag, &body)

	return body, etag, err
}

func (t *TBAHandler) cacheData(ctx context.Context, url string, etag string, body []byte) {
	assert := assert.CreateAssertWithContext("Cache Tba Data")
	assert.AddContext("url", url)
	assert.AddContext("Etag", etag)

	// Dont cache the data if we dont have a database
	// This is probably because we are running a unit test
	if t.database == nil {
		return
	}

	query := `Insert Into TbaCache (url, etag, responseBody) Values ($1, $2, $3)
		On Conflict (url) Do Update Set etag = excluded.etag, responseBody = excluded.responseBody;`
	stmt, err := t.database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Error(ctx, "cacheData: Failed to close statement", "error", err)
		}
	}()

	_, err = stmt.ExecContext(ctx, url, etag, body)
	if err != nil {
		log.Error(ctx, "Failed to cache tba data", "error", err)
	}
}

// makeRequest makes a request to The Blue Alliance API.
// url: The full URL to request
// endpoint: The endpoint template for metrics (e.g., "/team/{team}/event/{event}/matches")
func (t *TBAHandler) makeRequest(ctx context.Context, url string, endpoint string) []byte {
	log.Debug(ctx, "Making TBA request", "url", url, "endpoint", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error(ctx, "Failed to construct tba request", "error", err)
		return nil
	}

	log.PropagateCorrelationID(ctx, req)

	log.Debug(ctx, "Checking cache for tba data", "url", url)
	body, etag, err := t.checkCache(ctx, url)

	if err == nil {
		log.Debug(ctx, "Found cached data", "url", url, "etag", etag)
		req.Header.Add("If-None-Match", etag)
		metrics.RecordTbaCacheHit("hit")
	} else {
		log.Debug(ctx, "Did not find cached tba data", "url", url, "error", err)
		metrics.RecordTbaCacheHit("miss")
	}

	req.Header.Add("X-TBA-Auth-Key", t.tbaToken)
	start := time.Now()
	resp, err := t.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Error(ctx, "Failed to run tba request", "error", err)
		metrics.RecordTbaRequest(endpoint, 0, duration)
		return nil
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Error(ctx, "Failed to close tba request", "url", url, "error", err)
		}
	}()

	log.Debug(ctx, "Got response from tba", "statusCode", resp.Status)
	switch resp.StatusCode {
	case http.StatusNotModified:
		log.Debug(ctx, "Got not modified from tba, using cache data", "url", url)
		metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
		metrics.RecordTbaCacheHit("not_modified")
		return body
	case http.StatusNotFound:
		log.Debug(ctx, "TBA returned 404", "url", url)
		metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
		return nil
	default:
		if resp.StatusCode >= http.StatusInternalServerError {
			log.Error(ctx, "TBA returned server error", "url", url, "statusCode", resp.StatusCode)
			metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
			return nil
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			log.Warn(ctx, "TBA returned rate limit", "url", url, "statusCode", resp.StatusCode)
			metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
			return nil
		}
		log.Debug(ctx, "Request to Tba returned", "url", url, "statusCode", resp.StatusCode)
		metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
		metrics.RecordTbaCacheHit("miss")
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Error(ctx, "Failed to read tba request body", "error", err)
		return nil
	}

	etag = resp.Header.Get("Etag")
	if etag != "" {
		t.cacheData(ctx, url, etag, body)
	}

	return body
}

// MakeMatchListReq requests the list of matches for a team at an event from The Blue Alliance.
func (t *TBAHandler) MakeMatchListReq(ctx context.Context, teamId string, eventId string) []swagger.Match {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches"
	endpoint := "/team/{team}/event/{event}/matches"
	jsonData := t.makeRequest(ctx, url, endpoint)
	var matches []swagger.Match
	err := json.Unmarshal(jsonData, &matches)

	if err != nil {
		log.Error(ctx, "Failed to parse match list from tba", "messageData", jsonData, "team", teamId, "event", eventId, "error", err)
		return nil
	}

	return matches
}

// MakeEventListReq requests the list of events for a team from The Blue Alliance.
func (t *TBAHandler) MakeEventListReq(ctx context.Context, teamId string) []string {
	url := BASE_URL + "team/" + teamId + "/events/2026/keys"
	endpoint := "/team/{team}/events/{year}/keys"
	var events []string
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &events)

	if err != nil {
		log.Error(ctx, "Failed to parse event list from tba", "messageData", jsonData, "team", teamId, "error", err)
		return nil
	}

	return events
}

// MakeMatchReq requests a single match from The Blue Alliance.
func (t *TBAHandler) MakeMatchReq(ctx context.Context, matchId string) swagger.Match {
	url := BASE_URL + "match/" + matchId
	endpoint := "/match/{match}"
	var match swagger.Match
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &match)

	if err != nil {
		log.Error(ctx, "Failed to parse match from tba", "messageData", jsonData, "match", matchId, "error", err)
		return swagger.Match{}
	}

	return match
}

// MakeMatchKeysRequest requests the match keys for a team at an event from The Blue Alliance.
func (t *TBAHandler) MakeMatchKeysRequest(ctx context.Context, teamId string, eventId string) []string {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches/keys"
	endpoint := "/team/{team}/event/{event}/matches/keys"
	var keys []string
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &keys)

	if err != nil {
		log.Error(ctx, "Failed to parse match key list from tba", "messageData", jsonData, "team", teamId, "event", eventId, "error", err)
		return nil
	}

	return keys
}

// MakeEventMatchKeysRequest requests the match keys for an event from The Blue Alliance.
func (t *TBAHandler) MakeEventMatchKeysRequest(ctx context.Context, eventId string) []string {
	url := BASE_URL + "event/" + eventId + "/matches/keys"
	endpoint := "/event/{event}/matches/keys"
	var keys []string
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &keys)

	if err != nil {
		log.Error(ctx, "Failed to parse event match key list from tba", "messageData", jsonData, "event", eventId, "error", err)
		return nil
	}

	return keys
}

// MakeMatchKeysYearRequest requests the match keys for a team in a specific year from The Blue Alliance.
func (t *TBAHandler) MakeMatchKeysYearRequest(ctx context.Context, teamId string) []string {
	url := BASE_URL + "team/" + teamId + "/matches/2024/keys"
	endpoint := "/team/{team}/matches/{year}/keys"
	var matches []string
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &matches)

	if err != nil {
		log.Error(ctx, "Failed to parse match key year list from tba", "messageData", jsonData, "team", teamId, "error", err)
		return nil
	}

	return matches
}

// MakeTeamEventStatusRequest requests the team event status from The Blue Alliance.
func (t *TBAHandler) MakeTeamEventStatusRequest(ctx context.Context, teamId string, eventId string) swagger.TeamEventStatus {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/status"
	endpoint := "/team/{team}/event/{event}/status"
	var event swagger.TeamEventStatus
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &event)

	if err != nil {
		log.Error(ctx, "Failed to parse event status from tba", "messageData", jsonData, "team", teamId, "event", eventId, "error", err)
		return swagger.TeamEventStatus{}
	}

	return event
}

// MakeTeamsAtEventRequest requests the teams at an event from The Blue Alliance.
func (t *TBAHandler) MakeTeamsAtEventRequest(ctx context.Context, eventId string) []swagger.Team {
	url := BASE_URL + "event/" + eventId + "/teams/simple"
	endpoint := "/event/{event}/teams/simple"
	var teams []swagger.Team
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &teams)

	if err != nil {
		log.Error(ctx, "Failed to parse teams at event list from tba", "messageData", jsonData, "event", eventId, "error", err)
		return nil
	}

	return teams
}

// MakeEliminationAllianceRequest requests the elimination alliances for an event from The Blue Alliance.
// Retries with exponential backoff when TBA returns an empty alliance list (up to 5 retries).
func (t *TBAHandler) MakeEliminationAllianceRequest(ctx context.Context, eventId string) []swagger.EliminationAlliance {
	url := BASE_URL + "event/" + eventId + "/alliances"
	endpoint := "/event/{event}/alliances"

	const maxRetries = 5
	var alliances []swagger.EliminationAlliance

	for attempt := 0; attempt <= maxRetries; attempt++ {
		jsonData := t.makeRequest(ctx, url, endpoint)
		err := json.Unmarshal(jsonData, &alliances)

		if err != nil {
			log.Error(ctx, "Failed to parse elimination alliances from tba", "messageData", jsonData, "event", eventId, "error", err)
			return nil
		}

		if len(alliances) > 0 {
			return alliances
		}

		if attempt < maxRetries {
			backoff := time.Duration(1<<attempt) * time.Second
			log.Debug(ctx, "TBA returned empty alliances, retrying", "event", eventId, "attempt", attempt+1, "backoff", backoff)
			time.Sleep(backoff)
		}
	}

	log.Warn(ctx, "TBA returned empty alliances after all retries", "event", eventId, "attempts", maxRetries+1)
	return alliances
}

// MakeTeamAvatarRequest requests the team avatar/media from The Blue Alliance.
func (t *TBAHandler) MakeTeamAvatarRequest(ctx context.Context, teamId string) (string, error) {
	url := fmt.Sprintf("%steam/%s/media/%d", BASE_URL, teamId, time.Now().Year())
	endpoint := "/team/{team}/media/{year}"
	var media []swagger.TeamMedia
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &media)

	if err != nil {
		return "", err
	}

	for _, m := range media {
		if m.Type == "avatar" {
			return m.Details.Base64Image, nil
		}
	}

	return "", errors.New("Failed to find avatar in response: " + string(jsonData))
}
