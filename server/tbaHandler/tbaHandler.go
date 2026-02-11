package tbaHandler

import (
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"server/assert"
	"server/swagger"
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
		client:   &http.Client{},
	}
	return handler
}

func (t *TbaHandler) checkCache(url string) ([]byte, string, error) {
	assert := assert.CreateAssertWithContext("Check Tba Cache")
	assert.AddContext("Url", url)

	//Dont check the cache if we dont have a database
	//This is probably because we are running a unit test
	if t.database == nil {
		return nil, "", nil
	}

	query := `Select
        etag,
        responseBody
    From TbaCache
    Where url = $1;`
	stmt, err := t.database.Prepare(query)
	assert.NoError(err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			slog.Error("Failed to close statement", "error", err)
		}
	}()

	var etag string
	var body []byte
	err = stmt.QueryRow(url).Scan(&etag, &body)

	return body, etag, err
}

func (t *TbaHandler) cacheData(url string, etag string, body []byte) {
	assert := assert.CreateAssertWithContext("Cache Tba Data")
	assert.AddContext("Url", url)
	assert.AddContext("Etag", etag)

	//Dont cache the data if we dont have a database
	//This is probably because we are running a unit test
	if t.database == nil {
		return
	}

	query := `Insert Into TbaCache (url, etag, responseBody) Values ($1, $2, $3);`
	stmt, err := t.database.Prepare(query)
	assert.NoError(err, "Failed to prepare query")
	defer func() {
		if err := stmt.Close(); err != nil {
			slog.Error("Failed to close statement", "error", err)
		}
	}()

	_, err = stmt.Exec(url, etag, body)
	if err != nil {
		slog.Error("Failed to cache tba data", "Error", err)
	}
}

func (t *TbaHandler) makeRequest(url string) []byte {
	slog.Info("Making TBA request", "Url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("Failed to construct tba request", "Error", err)
		return nil
	}

	slog.Info("Checking cache for tba data", "Url", url)
	body, etag, err := t.checkCache(url)

	if err == nil {
		slog.Info("Found cached data", "Url", url, "Etag", etag)
		req.Header.Add("If-None-Match", etag)
	} else {
		slog.Warn("Did not find cached tba data", "Url", url, "Error", err)
	}

	req.Header.Add("X-TBA-Auth-Key", t.tbaToken)
	resp, err := t.client.Do(req)
	if err != nil {
		slog.Error("Failed to run tba request", "Error", err)
		return nil
	}

	defer func() {
		err = resp.Body.Close()
		if err != nil {
			slog.Warn("Failed to close tba request", "Url", url, "Error", err)
		}
	}()

	slog.Info("Got response from tba", "Status", resp.Status)
	switch resp.StatusCode {
	case http.StatusNotModified:
		slog.Info("Got not modified from tba, using cache data", "Url", url)
		return body
	case http.StatusNotFound:
		return nil
	default:
		slog.Info("Request to Tba returned", "Url", url, "Status", resp.StatusCode)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read tba request body", "Error", err)
		return nil
	}

	// TODO It looks like we might not be caching anything anymore.
	// Need to figure out why
	t.cacheData(url, resp.Header["Etag"][0], body)

	return body
}

// Make functions to make tba requests
func (t *TbaHandler) MakeMatchListReq(teamId string, eventId string) []swagger.Match {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches"
	jsonData := t.makeRequest(url)
	var matches []swagger.Match
	err := json.Unmarshal(jsonData, &matches)

	if err != nil {
		slog.Error("Failed to parse match list from tba", "Message Data", jsonData, "Team", teamId, "Event", eventId, "Error", err)
		return nil
	}

	return matches
}

func (t *TbaHandler) MakeEventListReq(teamId string) []string {
	url := BASE_URL + "team/" + teamId + "/events/2025/keys"
	var events []string
	jsonData := t.makeRequest(url)
	err := json.Unmarshal(jsonData, &events)

	if err != nil {
		slog.Error("Failed to parse event list from tba", "Message Data", jsonData, "Team", teamId, "Error", err)
		return nil
	}

	return events
}

func (t *TbaHandler) MakeMatchReq(matchId string) swagger.Match {
	url := BASE_URL + "match/" + matchId
	var match swagger.Match
	jsonData := t.makeRequest(url)
	err := json.Unmarshal(jsonData, &match)

	if err != nil {
		slog.Error("Failed to parse match from tba", "Message Data", jsonData, "Match", matchId, "Error", err)
		return swagger.Match{}
	}

	return match
}

func (t *TbaHandler) MakeMatchKeysRequest(teamId string, eventId string) []string {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches/keys"
	var keys []string
	jsonData := t.makeRequest(url)
	err := json.Unmarshal(jsonData, &keys)

	if err != nil {
		slog.Error("Failed to parse match key list from tba", "Message Data", jsonData, "Team", teamId, "Event", eventId, "Error", err)
		return nil
	}

	return keys
}

func (t *TbaHandler) MakeEventMatchKeysRequest(eventId string) []string {
	url := BASE_URL + "event/" + eventId + "/matches/keys"
	var keys []string
	jsonData := t.makeRequest(url)
	err := json.Unmarshal(jsonData, &keys)

	if err != nil {
		slog.Error("Failed to parse event match key list from tba", "Message Data", jsonData, "Event", eventId, "Error", err)
		return nil
	}

	return keys
}

func (t *TbaHandler) MakeMatchKeysYearRequest(teamId string) []string {
	url := BASE_URL + "team/" + teamId + "/matches/2024/keys"
	var matches []string
	jsonData := t.makeRequest(url)
	err := json.Unmarshal(jsonData, &matches)

	if err != nil {
		slog.Error("Failed to parse match key year list from tba", "Message Data", jsonData, "Team", teamId, "Error", err)
		return nil
	}

	return matches
}

func (t *TbaHandler) MakeTeamEventStatusRequest(teamId string, eventId string) swagger.TeamEventStatus {
	url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/status"
	var event swagger.TeamEventStatus
	jsonData := t.makeRequest(url)
	err := json.Unmarshal(jsonData, &event)

	if err != nil {
		slog.Error("Failed to parse event status from tba", "Message Data", jsonData, "Team", teamId, "Event", eventId, "Error", err)
		return swagger.TeamEventStatus{}
	}

	return event
}

func (t *TbaHandler) MakeTeamsAtEventRequest(eventId string) []swagger.Team {
	url := BASE_URL + "event/" + eventId + "/teams/simple"
	var teams []swagger.Team
	jsonData := t.makeRequest(url)
	err := json.Unmarshal(jsonData, &teams)

	if err != nil {
		slog.Error("Failed to parse teams at event list from tba", "Message Data", jsonData, "Event", eventId, "Error", err)
		return nil
	}

	return teams
}

func (t *TbaHandler) MakeEliminationAllianceRequest(eventId string) []swagger.EliminationAlliance {
	url := BASE_URL + "event/" + eventId + "/alliances"
	var alliances []swagger.EliminationAlliance
	jsonData := t.makeRequest(url)
	slog.Info(string(jsonData))
	err := json.Unmarshal(jsonData, &alliances)

	if err != nil {
		slog.Error("Failed to parse elimination alliances from tba", "Message Data", jsonData, "Event", eventId, "Error", err)
		return nil
	}

	return alliances
}
