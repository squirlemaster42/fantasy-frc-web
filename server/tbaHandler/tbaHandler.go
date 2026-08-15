package tbaHandler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	db "server/database"
	"server/log"
	"server/metrics"
	"server/swagger"
	"server/utils"
	"time"

	otelhttp "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type TBAInterface interface {
	MakeEventListReq(ctx context.Context, teamId string) ([]string, error)
	MakeMatchReq(ctx context.Context, matchId string) (swagger.Match, error)
	MakeEventMatchKeysRequest(ctx context.Context, eventId string) ([]string, error)
	MakeTeamsAtEventRequest(ctx context.Context, eventId string) ([]swagger.Team, error)
	MakeEliminationAllianceRequest(ctx context.Context, eventId string) ([]swagger.EliminationAlliance, error)
	MakeTeamAvatarRequest(ctx context.Context, teamId string) (string, error)
}

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
	stmt, err := db.Prepare(ctx, t.database, query)
	if err != nil {
		return nil, "", err
	}
	defer db.CloseStatement(ctx, stmt, "checkCache")

	var etag string
	var body []byte
	err = stmt.QueryRowContext(ctx, url).Scan(&etag, &body)

	return body, etag, err
}

func (t *TBAHandler) cacheData(ctx context.Context, url string, etag string, body []byte) {
	// Dont cache the data if we dont have a database
	// This is probably because we are running a unit test
	if t.database == nil {
		return
	}

	query := `Insert Into TbaCache (url, etag, responseBody) Values ($1, $2, $3)
		On Conflict (url) Do Update Set etag = excluded.etag, responseBody = excluded.responseBody;`
	stmt, err := db.Prepare(ctx, t.database, query)
	if err != nil {
		log.Error(ctx, "cacheData: Failed to prepare statement", "error", err)
		return
	}
	defer db.CloseStatement(ctx, stmt, "cacheData")

	_, err = stmt.ExecContext(ctx, url, etag, body)
	if err != nil {
		log.Error(ctx, "Failed to cache tba data", "error", err)
	}
}

// makeRequest makes a request to The Blue Alliance API.
// url: The full URL to request
// endpoint: The endpoint template for metrics (e.g., "/team/{team}/event/{event}/matches")
func (t *TBAHandler) makeRequest(ctx context.Context, url string, endpoint string) ([]byte, error) {
	log.Debug(ctx, "Making TBA request", "url", url, "endpoint", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to construct tba request: %w", err)
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
		metrics.RecordTbaRequest(endpoint, 0, duration)
		return nil, fmt.Errorf("failed to run tba request: %w", err)
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
		return body, nil
	case http.StatusNotFound:
		log.Debug(ctx, "TBA returned 404", "url", url)
		metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
		return nil, nil
	default:
		if resp.StatusCode >= http.StatusInternalServerError {
			log.Error(ctx, "TBA returned server error", "url", url, "statusCode", resp.StatusCode)
			metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
			return nil, fmt.Errorf("tba returned server error: %d", resp.StatusCode)
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			log.Warn(ctx, "TBA returned rate limit", "url", url, "statusCode", resp.StatusCode)
			metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
			return nil, fmt.Errorf("tba returned rate limit: %d", resp.StatusCode)
		}
		if resp.StatusCode != http.StatusOK {
			metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
			return nil, fmt.Errorf("tba returned unexpected status: %d", resp.StatusCode)
		}
		log.Debug(ctx, "Request to Tba returned", "url", url, "statusCode", resp.StatusCode)
		metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
		metrics.RecordTbaCacheHit("miss")
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read tba request body: %w", err)
	}

	etag = resp.Header.Get("Etag")
	if etag != "" {
		t.cacheData(ctx, url, etag, body)
	}

	return body, nil
}

// MakeMatchListReq requests the list of matches for a team at an event from The Blue Alliance.
func (t *TBAHandler) MakeMatchListReq(ctx context.Context, teamId string, eventId string) ([]swagger.Match, error) {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches"
	endpoint := "/team/{team}/event/{event}/matches"
	jsonData, err := t.makeRequest(ctx, url, endpoint)
	if err != nil {
		return nil, err
	}
	var matches []swagger.Match
	err = json.Unmarshal(jsonData, &matches)
	if err != nil {
		return nil, fmt.Errorf("failed to parse match list from tba: %w", err)
	}
	return matches, nil
}

// MakeEventListReq requests the list of events for a team from The Blue Alliance.
func (t *TBAHandler) MakeEventListReq(ctx context.Context, teamId string) ([]string, error) {
	url := BASE_URL + "team/" + teamId + "/events/" + strconv.Itoa(utils.TbaSeasonYear) + "/keys"
	endpoint := "/team/{team}/events/{year}/keys"
	jsonData, err := t.makeRequest(ctx, url, endpoint)
	if err != nil {
		return nil, err
	}
	var events []string
	err = json.Unmarshal(jsonData, &events)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event list from tba: %w", err)
	}
	return events, nil
}

// MakeMatchReq requests a single match from The Blue Alliance.
func (t *TBAHandler) MakeMatchReq(ctx context.Context, matchId string) (swagger.Match, error) {
	url := BASE_URL + "match/" + matchId
	endpoint := "/match/{match}"
	jsonData, err := t.makeRequest(ctx, url, endpoint)
	if err != nil {
		return swagger.Match{}, err
	}
	var match swagger.Match
	err = json.Unmarshal(jsonData, &match)
	if err != nil {
		return swagger.Match{}, fmt.Errorf("failed to parse match from tba: %w", err)
	}
	return match, nil
}

// MakeMatchKeysRequest requests the match keys for a team at an event from The Blue Alliance.
func (t *TBAHandler) MakeMatchKeysRequest(ctx context.Context, teamId string, eventId string) ([]string, error) {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches/keys"
	endpoint := "/team/{team}/event/{event}/matches/keys"
	jsonData, err := t.makeRequest(ctx, url, endpoint)
	if err != nil {
		return nil, err
	}
	var keys []string
	err = json.Unmarshal(jsonData, &keys)
	if err != nil {
		return nil, fmt.Errorf("failed to parse match key list from tba: %w", err)
	}
	return keys, nil
}

// MakeEventMatchKeysRequest requests the match keys for an event from The Blue Alliance.
func (t *TBAHandler) MakeEventMatchKeysRequest(ctx context.Context, eventId string) ([]string, error) {
	url := BASE_URL + "event/" + eventId + "/matches/keys"
	endpoint := "/event/{event}/matches/keys"
	jsonData, err := t.makeRequest(ctx, url, endpoint)
	if err != nil {
		return nil, err
	}
	var keys []string
	err = json.Unmarshal(jsonData, &keys)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event match key list from tba: %w", err)
	}
	return keys, nil
}

// MakeMatchKeysYearRequest requests the match keys for a team in a specific year from The Blue Alliance.
func (t *TBAHandler) MakeMatchKeysYearRequest(ctx context.Context, teamId string) ([]string, error) {
	url := BASE_URL + "team/" + teamId + "/matches/" + strconv.Itoa(utils.TbaHistoricMatchYear) + "/keys"
	endpoint := "/team/{team}/matches/{year}/keys"
	jsonData, err := t.makeRequest(ctx, url, endpoint)
	if err != nil {
		return nil, err
	}
	var matches []string
	err = json.Unmarshal(jsonData, &matches)
	if err != nil {
		return nil, fmt.Errorf("failed to parse match key year list from tba: %w", err)
	}
	return matches, nil
}

// MakeTeamEventStatusRequest requests the team event status from The Blue Alliance.
func (t *TBAHandler) MakeTeamEventStatusRequest(ctx context.Context, teamId string, eventId string) (swagger.TeamEventStatus, error) {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/status"
	endpoint := "/team/{team}/event/{event}/status"
	jsonData, err := t.makeRequest(ctx, url, endpoint)
	if err != nil {
		return swagger.TeamEventStatus{}, err
	}
	var event swagger.TeamEventStatus
	err = json.Unmarshal(jsonData, &event)
	if err != nil {
		return swagger.TeamEventStatus{}, fmt.Errorf("failed to parse event status from tba: %w", err)
	}
	return event, nil
}

// MakeTeamsAtEventRequest requests the teams at an event from The Blue Alliance.
func (t *TBAHandler) MakeTeamsAtEventRequest(ctx context.Context, eventId string) ([]swagger.Team, error) {
	url := BASE_URL + "event/" + eventId + "/teams/simple"
	endpoint := "/event/{event}/teams/simple"
	jsonData, err := t.makeRequest(ctx, url, endpoint)
	if err != nil {
		return nil, err
	}
	var teams []swagger.Team
	err = json.Unmarshal(jsonData, &teams)
	if err != nil {
		return nil, fmt.Errorf("failed to parse teams at event list from tba: %w", err)
	}
	return teams, nil
}

// MakeEliminationAllianceRequest requests the elimination alliances for an event from The Blue Alliance.
// Retries with exponential backoff when TBA returns an empty alliance list (up to 5 retries).
func (t *TBAHandler) MakeEliminationAllianceRequest(ctx context.Context, eventId string) ([]swagger.EliminationAlliance, error) {
	url := BASE_URL + "event/" + eventId + "/alliances"
	endpoint := "/event/{event}/alliances"

	const maxRetries = 5
	var alliances []swagger.EliminationAlliance

	for attempt := 0; attempt <= maxRetries; attempt++ {
		jsonData, err := t.makeRequest(ctx, url, endpoint)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(jsonData, &alliances)
		if err != nil {
			return nil, fmt.Errorf("failed to parse elimination alliances from tba: %w", err)
		}

		if len(alliances) > 0 {
			return alliances, nil
		}

		if attempt < maxRetries {
			backoff := time.Duration(1<<attempt) * time.Second
			log.Debug(ctx, "TBA returned empty alliances, retrying", "event", eventId, "attempt", attempt+1, "backoff", backoff)
			time.Sleep(backoff)
		}
	}

	log.Warn(ctx, "TBA returned empty alliances after all retries", "event", eventId, "attempts", maxRetries+1)
	return alliances, nil
}

// MakeTeamAvatarRequest requests the team avatar/media from The Blue Alliance.
func (t *TBAHandler) MakeTeamAvatarRequest(ctx context.Context, teamId string) (string, error) {
	url := fmt.Sprintf("%steam/%s/media/%d", BASE_URL, teamId, time.Now().Year())
	endpoint := "/team/{team}/media/{year}"
	jsonData, err := t.makeRequest(ctx, url, endpoint)
	if err != nil {
		return "", err
	}
	var media []swagger.TeamMedia
	err = json.Unmarshal(jsonData, &media)
	if err != nil {
		return "", fmt.Errorf("failed to parse team media from tba: %w", err)
	}

	for _, m := range media {
		if m.Type == "avatar" {
			return m.Details.Base64Image, nil
		}
	}

	return "", errors.New("failed to find avatar in response")
}
