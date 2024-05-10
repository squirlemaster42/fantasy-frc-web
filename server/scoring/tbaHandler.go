package scoring

import (
	"encoding/json"
	"io"
	"net/http"
    models "server/web/model"
)

const (
    BASE_URL = "https://www.thebluealliance.com/api/v3/"
)

type TbaHandler struct {
    tbaToken string
}

func NewHandler(tbaToken string) *TbaHandler {
    handler := &TbaHandler{tbaToken: tbaToken}
    return handler
}

func (t *TbaHandler) makeRequest(url string) []byte {
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
func (t *TbaHandler) makeMatchListReq(teamId string, eventId string) []models.Match {
    url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches"
    jsonData := t.makeRequest(url)
    var matches []models.Match
    json.Unmarshal(jsonData, &matches)
    return matches
}

func (t *TbaHandler) makeEventListReq(teamId string) []string {
    url := BASE_URL + "team/" + teamId + "/events/2024/keys"
    var events []string
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &events)
    return events
}

func (t *TbaHandler) makeMatchReq(matchId string) models.Match {
    url := BASE_URL + "match/" + matchId
    var match models.Match
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &match)
    return match
}

func (t *TbaHandler) makeMatchKeysRequest(teamId string, eventId string) []string {
    url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches/keys"
    var keys []string
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &keys)
    return keys
}

func (t *TbaHandler) makeMatchKeysYearRequest(teamId string) []string {
    url := BASE_URL + "team/" + teamId + "/matches/2024/keys"
    var matches []string
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &matches)
    return matches
}

func (t *TbaHandler) makeTeamEventStatusRequest(teamId string, eventId string) models.Event {
    url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/status"
    var event models.Event
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &event)
    return event
}

func (t *TbaHandler) makeTeamsAtEventRequest(eventId string) []models.TbaTeam {
    url := BASE_URL + "event/" + eventId + "/teams/simple"
    var teams []models.TbaTeam
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &teams)
    return teams
}
