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

func (t *TbaHandler) checkCache(context context.Context, url string) ([]byte, string, error) {
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
	stmt, err := t.database.Prepare(query)
	assert.NoError(context, err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(context, "checkCache: Failed to close statement", "error", err)
		}
	}()

	var etag string
	var body []byte
	err = stmt.QueryRow(url).Scan(&etag, &body)

	return body, etag, err
}

func (t *TbaHandler) cacheData(context context.Context, url string, etag string, body []byte) {
	assert := assert.CreateAssertWithContext("Cache Tba Data")
	assert.AddContext("Url", url)
	assert.AddContext("Etag", etag)

	// Dont cache the data if we dont have a database
	// This is probably because we are running a unit test
	if t.database == nil {
		return
	}

	query := `Insert Into TbaCache (url, etag, responseBody) Values ($1, $2, $3);`
	stmt, err := t.database.Prepare(query)
	assert.NoError(context, err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(context, "cacheData: Failed to close statement", "error", err)
		}
	}()

	_, err = stmt.Exec(url, etag, body)
	if err != nil {
		log.Error(context, "Failed to cache tba data", "Error", err)
	}
}

// makeRequest makes a request to The Blue Alliance API.
// url: The full URL to request
// endpoint: The endpoint template for metrics (e.g., "/team/{team}/event/{event}/matches")
func (t *TbaHandler) makeRequest(context context.Context, url string, endpoint string) []byte {
	log.Debug(context, "Making TBA request", "Url", url, "Endpoint", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Error(context, "Failed to construct tba request", "Error", err)
		return nil
	}

	log.Debug(context, "Checking cache for tba data", "Url", url)
	body, etag, err := t.checkCache(context, url)

	if err == nil {
		log.Debug(context, "Found cached data", "Url", url, "Etag", etag)
		req.Header.Add("If-None-Match", etag)
		metrics.RecordTbaCacheHit("hit")
	} else {
		log.Warn(context, "Did not find cached tba data", "Url", url, "Error", err)
		metrics.RecordTbaCacheHit("miss")
	}

	req.Header.Add("X-TBA-Auth-Key", t.tbaToken)
	start := time.Now()
	resp, err := t.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Error(context, "Failed to run tba request", "Error", err)
		metrics.RecordTbaRequest(endpoint, 0, duration)
		return nil
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			log.Warn(context, "Failed to close tba request", "Url", url, "Error", err)
		}
	}()

	log.Debug(context, "Got response from tba", "Status", resp.Status)
	switch resp.StatusCode {
	case http.StatusNotModified:
		log.Debug(context, "Got not modified from tba, using cache data", "Url", url)
		metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
		metrics.RecordTbaCacheHit("not_modified")
		return body
	case http.StatusNotFound:
		metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
		return nil
	default:
		log.Debug(context, "Request to Tba returned", "Url", url, "Status", resp.StatusCode)
		metrics.RecordTbaRequest(endpoint, resp.StatusCode, duration)
		metrics.RecordTbaCacheHit("miss")
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Error(context, "Failed to read tba request body", "Error", err)
		return nil
	}

	t.cacheData(context, url, resp.Header["Etag"][0], body)

	return body
}

// MakeMatchListReq requests the list of matches for a team at an event from The Blue Alliance.
func (t *TbaHandler) MakeMatchListReq(context context.Context, teamId string, eventId string) []swagger.Match {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches"
	endpoint := "/team/{team}/event/{event}/matches"
	jsonData := t.makeRequest(context, url, endpoint)
	var matches []swagger.Match
	err := json.Unmarshal(jsonData, &matches)

	if err != nil {
		log.Error(context, "Failed to parse match list from tba", "Message Data", jsonData, "Team", teamId, "Event", eventId, "Error", err)
		return nil
	}

	return matches
}

// MakeEventListReq requests the list of events for a team from The Blue Alliance.
func (t *TbaHandler) MakeEventListReq(context context.Context, teamId string) []string {
	url := BASE_URL + "team/" + teamId + "/events/2026/keys"
	endpoint := "/team/{team}/events/{year}/keys"
	var events []string
	jsonData := t.makeRequest(context, url, endpoint)
	err := json.Unmarshal(jsonData, &events)

	if err != nil {
		log.Error(context, "Failed to parse event list from tba", "Message Data", jsonData, "Team", teamId, "Error", err)
		return nil
	}

	return events
}

// MakeMatchReq requests a single match from The Blue Alliance.
func (t *TbaHandler) MakeMatchReq(context context.Context, matchId string) swagger.Match {
	url := BASE_URL + "match/" + matchId
	endpoint := "/match/{match}"
	var match swagger.Match
	jsonData := t.makeRequest(context, url, endpoint)
	err := json.Unmarshal(jsonData, &match)

	if err != nil {
		log.Error(context, "Failed to parse match from tba", "Message Data", jsonData, "Match", matchId, "Error", err)
		return swagger.Match{}
	}

	return match
}

// MakeMatchKeysRequest requests the match keys for a team at an event from The Blue Alliance.
func (t *TbaHandler) MakeMatchKeysRequest(context context.Context, teamId string, eventId string) []string {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches/keys"
	endpoint := "/team/{team}/event/{event}/matches/keys"
	var keys []string
	jsonData := t.makeRequest(context, url, endpoint)
	err := json.Unmarshal(jsonData, &keys)

	if err != nil {
		log.Error(context, "Failed to parse match key list from tba", "Message Data", jsonData, "Team", teamId, "Event", eventId, "Error", err)
		return nil
	}

	return keys
}

// MakeEventMatchKeysRequest requests the match keys for an event from The Blue Alliance.
func (t *TbaHandler) MakeEventMatchKeysRequest(context context.Context, eventId string) []string {
	url := BASE_URL + "event/" + eventId + "/matches/keys"
	endpoint := "/event/{event}/matches/keys"
	var keys []string
	jsonData := t.makeRequest(context, url, endpoint)
	err := json.Unmarshal(jsonData, &keys)

	if err != nil {
		log.Error(context, "Failed to parse event match key list from tba", "Message Data", jsonData, "Event", eventId, "Error", err)
		return nil
	}

	return keys
}

// MakeMatchKeysYearRequest requests the match keys for a team in a specific year from The Blue Alliance.
func (t *TbaHandler) MakeMatchKeysYearRequest(context context.Context, teamId string) []string {
	url := BASE_URL + "team/" + teamId + "/matches/2024/keys"
	endpoint := "/team/{team}/matches/{year}/keys"
	var matches []string
	jsonData := t.makeRequest(context, url, endpoint)
	err := json.Unmarshal(jsonData, &matches)

	if err != nil {
		log.Error(context, "Failed to parse match key year list from tba", "Message Data", jsonData, "Team", teamId, "Error", err)
		return nil
	}

	return matches
}

// MakeTeamEventStatusRequest requests the team event status from The Blue Alliance.
func (t *TbaHandler) MakeTeamEventStatusRequest(context context.Context, teamId string, eventId string) swagger.TeamEventStatus {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/status"
	endpoint := "/team/{team}/event/{event}/status"
	var event swagger.TeamEventStatus
	jsonData := t.makeRequest(context, url, endpoint)
	err := json.Unmarshal(jsonData, &event)

	if err != nil {
		log.Error(context, "Failed to parse event status from tba", "Message Data", jsonData, "Team", teamId, "Event", eventId, "Error", err)
		return swagger.TeamEventStatus{}
	}

	return event
}

// MakeTeamsAtEventRequest requests the teams at an event from The Blue Alliance.
func (t *TbaHandler) MakeTeamsAtEventRequest(context context.Context, eventId string) []swagger.Team {
	url := BASE_URL + "event/" + eventId + "/teams/simple"
	endpoint := "/event/{event}/teams/simple"
	var teams []swagger.Team
	jsonData := t.makeRequest(context, url, endpoint)
	err := json.Unmarshal(jsonData, &teams)

	if err != nil {
		log.Error(context, "Failed to parse teams at event list from tba", "Message Data", jsonData, "Event", eventId, "Error", err)
		return nil
	}

	return teams
}

// MakeEliminationAllianceRequest requests the elimination alliances for an event from The Blue Alliance.
func (t *TbaHandler) MakeEliminationAllianceRequest(context context.Context, eventId string) []swagger.EliminationAlliance {
	url := BASE_URL + "event/" + eventId + "/alliances"
	endpoint := "/event/{event}/alliances"
	var alliances []swagger.EliminationAlliance
	jsonData := t.makeRequest(context, url, endpoint)
	err := json.Unmarshal(jsonData, &alliances)

	if err != nil {
		log.Error(context, "Failed to parse elimination alliances from tba", "Message Data", jsonData, "Event", eventId, "Error", err)
		return nil
	}

	return alliances
}

// MakeTeamAvatarRequest requests the team avatar/media from The Blue Alliance.
func (t *TbaHandler) MakeTeamAvatarRequest(context context.Context, teamId string) (string, error) {
	url := fmt.Sprintf("%steam/%s/media/%d", BASE_URL, teamId, time.Now().Year())
	endpoint := "/team/{team}/media/{year}"
	var media []swagger.TeamMedia
	jsonData := t.makeRequest(context, url, endpoint)
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
