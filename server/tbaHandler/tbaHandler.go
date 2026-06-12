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

type TbaHandler struct {
	tbaToken string
	database *sql.DB
	client   *http.Client
}

func NewHandler(tbaToken string, database *sql.DB) *TbaHandler {
	handler := &TbaHandler{
		tbaToken: tbaToken,
		database: database,
		client: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
	return handler
}

func (t *TbaHandler) checkCache(ctx context.Context, url string) ([]byte, string, error) {
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
			log.Warn(ctx, "checkCache: Failed to close statement", "error", err)
		}
	}()

	var etag string
	var body []byte
	err = stmt.QueryRowContext(ctx, url).Scan(&etag, &body)

	return body, etag, err
}

func (t *TbaHandler) cacheData(ctx context.Context, url string, etag string, body []byte) {
	assert := assert.CreateAssertWithContext("Cache Tba Data")
	assert.AddContext("Url", url)
	assert.AddContext("Etag", etag)

	// Dont cache the data if we dont have a database
	// This is probably because we are running a unit test
	if t.database == nil {
		return
	}

	query := `Insert Into TbaCache (url, etag, responseBody) Values ($1, $2, $3);`
	stmt, err := t.database.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "cacheData: Failed to close statement", "error", err)
		}
	}()

	_, err = stmt.ExecContext(ctx, url, etag, body)
	if err != nil {
		log.Error(ctx, "Failed to cache tba data", "Error", err)
	}
}

// makeRequest makes a request to The Blue Alliance API.
// url: The full URL to request
// endpoint: The endpoint template for metrics (e.g., "/team/{team}/event/{event}/matches")
func (t *TbaHandler) makeRequest(ctx context.Context, url string, endpoint string) []byte {
	log.Debug(ctx, "Making TBA request", "Url", url, "Endpoint", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error(ctx, "Failed to construct tba request", "Error", err)
		return nil
	}

	log.Debug(ctx, "Checking cache for tba data", "Url", url)
	body, etag, err := t.checkCache(ctx, url)

	if err == nil {
		log.Debug(ctx, "Found cached data", "Url", url, "Etag", etag)
		req.Header.Add("If-None-Match", etag)
		metrics.RecordTbaCacheHit("hit")
	} else {
		log.Warn(ctx, "Did not find cached tba data", "Url", url, "Error", err)
		metrics.RecordTbaCacheHit("miss")
	}

	req.Header.Add("X-TBA-Auth-Key", t.tbaToken)
	start := time.Now()
	resp, err := t.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Error(ctx, "Failed to run tba request", "Error", err)
		metrics.RecordTbaRequest(endpoint, 0, duration)
		return nil
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Warn(ctx, "Failed to close tba request", "Url", url, "Error", err)
		}
	}()

	log.Debug(ctx, "Got response from tba", "Status", resp.Status)
	switch resp.StatusCode {
	case http.StatusNotModified:
		log.Debug(ctx, "Got not modified from tba, using cache data", "Url", url)
		metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
		metrics.RecordTbaCacheHit("not_modified")
		return body
	case http.StatusNotFound:
		metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
		return nil
	default:
		log.Debug(ctx, "Request to Tba returned", "Url", url, "Status", resp.StatusCode)
		metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
		metrics.RecordTbaCacheHit("miss")
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Error(ctx, "Failed to read tba request body", "Error", err)
		return nil
	}

	t.cacheData(ctx, url, resp.Header["Etag"][0], body)

	return body
}

// MakeMatchListReq requests the list of matches for a team at an event from The Blue Alliance.
func (t *TbaHandler) MakeMatchListReq(ctx context.Context, teamId string, eventId string) []swagger.Match {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches"
	endpoint := "/team/{team}/event/{event}/matches"
	jsonData := t.makeRequest(ctx, url, endpoint)
	var matches []swagger.Match
	err := json.Unmarshal(jsonData, &matches)

	if err != nil {
		log.Error(ctx, "Failed to parse match list from tba", "Message Data", jsonData, "Team", teamId, "Event", eventId, "Error", err)
		return nil
	}

	return matches
}

// MakeEventListReq requests the list of events for a team from The Blue Alliance.
func (t *TbaHandler) MakeEventListReq(ctx context.Context, teamId string) []string {
	url := BASE_URL + "team/" + teamId + "/events/2026/keys"
	endpoint := "/team/{team}/events/{year}/keys"
	var events []string
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &events)

	if err != nil {
		log.Error(ctx, "Failed to parse event list from tba", "Message Data", jsonData, "Team", teamId, "Error", err)
		return nil
	}

	return events
}

// MakeMatchReq requests a single match from The Blue Alliance.
func (t *TbaHandler) MakeMatchReq(ctx context.Context, matchId string) swagger.Match {
	url := BASE_URL + "match/" + matchId
	endpoint := "/match/{match}"
	var match swagger.Match
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &match)

	if err != nil {
		log.Error(ctx, "Failed to parse match from tba", "Message Data", jsonData, "Match", matchId, "Error", err)
		return swagger.Match{}
	}

	return match
}

// MakeMatchKeysRequest requests the match keys for a team at an event from The Blue Alliance.
func (t *TbaHandler) MakeMatchKeysRequest(ctx context.Context, teamId string, eventId string) []string {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches/keys"
	endpoint := "/team/{team}/event/{event}/matches/keys"
	var keys []string
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &keys)

	if err != nil {
		log.Error(ctx, "Failed to parse match key list from tba", "Message Data", jsonData, "Team", teamId, "Event", eventId, "Error", err)
		return nil
	}

	return keys
}

// MakeEventMatchKeysRequest requests the match keys for an event from The Blue Alliance.
func (t *TbaHandler) MakeEventMatchKeysRequest(ctx context.Context, eventId string) []string {
	url := BASE_URL + "event/" + eventId + "/matches/keys"
	endpoint := "/event/{event}/matches/keys"
	var keys []string
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &keys)

	if err != nil {
		log.Error(ctx, "Failed to parse event match key list from tba", "Message Data", jsonData, "Event", eventId, "Error", err)
		return nil
	}

	return keys
}

// MakeMatchKeysYearRequest requests the match keys for a team in a specific year from The Blue Alliance.
func (t *TbaHandler) MakeMatchKeysYearRequest(ctx context.Context, teamId string) []string {
	url := BASE_URL + "team/" + teamId + "/matches/2024/keys"
	endpoint := "/team/{team}/matches/{year}/keys"
	var matches []string
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &matches)

	if err != nil {
		log.Error(ctx, "Failed to parse match key year list from tba", "Message Data", jsonData, "Team", teamId, "Error", err)
		return nil
	}

	return matches
}

// MakeTeamEventStatusRequest requests the team event status from The Blue Alliance.
func (t *TbaHandler) MakeTeamEventStatusRequest(ctx context.Context, teamId string, eventId string) swagger.TeamEventStatus {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/status"
	endpoint := "/team/{team}/event/{event}/status"
	var event swagger.TeamEventStatus
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &event)

	if err != nil {
		log.Error(ctx, "Failed to parse event status from tba", "Message Data", jsonData, "Team", teamId, "Event", eventId, "Error", err)
		return swagger.TeamEventStatus{}
	}

	return event
}

// MakeTeamsAtEventRequest requests the teams at an event from The Blue Alliance.
func (t *TbaHandler) MakeTeamsAtEventRequest(ctx context.Context, eventId string) []swagger.Team {
	url := BASE_URL + "event/" + eventId + "/teams/simple"
	endpoint := "/event/{event}/teams/simple"
	var teams []swagger.Team
	jsonData := t.makeRequest(ctx, url, endpoint)
	err := json.Unmarshal(jsonData, &teams)

	if err != nil {
		log.Error(ctx, "Failed to parse teams at event list from tba", "Message Data", jsonData, "Event", eventId, "Error", err)
		return nil
	}

	return teams
}

// MakeEliminationAllianceRequest requests the elimination alliances for an event from The Blue Alliance.
// Retries with exponential backoff when TBA returns an empty alliance list (up to 5 retries).
func (t *TbaHandler) MakeEliminationAllianceRequest(ctx context.Context, eventId string) []swagger.EliminationAlliance {
	url := BASE_URL + "event/" + eventId + "/alliances"
	endpoint := "/event/{event}/alliances"

	const maxRetries = 5
	var alliances []swagger.EliminationAlliance

	for attempt := 0; attempt <= maxRetries; attempt++ {
		jsonData := t.makeRequest(ctx, url, endpoint)
		err := json.Unmarshal(jsonData, &alliances)

		if err != nil {
			log.Error(ctx, "Failed to parse elimination alliances from tba", "Message Data", jsonData, "Event", eventId, "Error", err)
			return nil
		}

		if len(alliances) > 0 {
			return alliances
		}

		if attempt < maxRetries {
			backoff := time.Duration(1<<attempt) * time.Second
			log.Info(ctx, "TBA returned empty alliances, retrying", "Event", eventId, "Attempt", attempt+1, "Backoff", backoff)
			time.Sleep(backoff)
		}
	}

	log.Warn(ctx, "TBA returned empty alliances after all retries", "Event", eventId, "Attempts", maxRetries+1)
	return alliances
}

// MakeTeamAvatarRequest requests the team avatar/media from The Blue Alliance.
func (t *TbaHandler) MakeTeamAvatarRequest(ctx context.Context, teamId string) (string, error) {
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
