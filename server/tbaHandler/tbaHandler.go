package tbaHandler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"server/logging"
    "server/swagger"
)

const (
    BASE_URL = "https://www.thebluealliance.com/api/v3/"
)

type TbaHandler struct {
    tbaToken string
    logger *logging.Logger
}

func NewHandler(tbaToken string, logger *logging.Logger) *TbaHandler {
    handler := &TbaHandler{ tbaToken: tbaToken, logger: logger,
    }
    return handler
}

func (t *TbaHandler) makeRequest(url string) []byte {
    t.logger.Log(fmt.Sprintf("Making TBA request to %s", url))
    client := &http.Client{}

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil
    }

    req.Header.Add("X-TBA-Auth-Key", t.tbaToken)
    resp, err := client.Do(req)
    if err != nil {
        return nil
    }

    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil
    }

    return body
}

//Make functions to make tba requests
func (t *TbaHandler) MakeMatchListReq(teamId string, eventId string) []swagger.Match {
    url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches"
    jsonData := t.makeRequest(url)
    var matches []swagger.Match
    json.Unmarshal(jsonData, &matches)
    return matches
}

func (t *TbaHandler) MakeEventListReq(teamId string) []string {
    url := BASE_URL + "team/" + teamId + "/events/2024/keys"
    var events []string
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &events)
    return events
}

func (t *TbaHandler) MakeMatchReq(matchId string) swagger.Match {
    url := BASE_URL + "match/" + matchId
    var match swagger.Match
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &match)
    return match
}

func (t *TbaHandler) MakeMatchKeysRequest(teamId string, eventId string) []string {
    url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches/keys"
    var keys []string
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &keys)
    return keys
}

func (t *TbaHandler) MakeEventMatchKeysRequest(eventId string) []string {
    url := BASE_URL + "event/" + eventId + "/matches/keys"
    var keys []string
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &keys)
    return keys
}

func (t *TbaHandler) MakeMatchKeysYearRequest(teamId string) []string {
    url := BASE_URL + "team/" + teamId + "/matches/2024/keys"
    var matches []string
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &matches)
    return matches
}

func (t *TbaHandler) MakeTeamEventStatusRequest(teamId string, eventId string) swagger.TeamEventStatus {
    url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/status"
    var event swagger.TeamEventStatus
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &event)
    return event
}

func (t *TbaHandler) MakeTeamsAtEventRequest(eventId string) []swagger.Team {
    url := BASE_URL + "event/" + eventId + "/teams/simple"
    var teams []swagger.Team
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &teams)
    return teams
}
