package tbaHandler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

const (
    BASE_URL = "https://www.thebluealliance.com/api/v3/"
)

//Make structs for different tba objects
type Match struct {
	ActualTime int `json:"actual_time"`
	Alliances  struct {
		Blue struct {
			DqTeamKeys        []string `json:"dq_team_keys"`
			Score             int      `json:"score"`
			SurrogateTeamKeys []string `json:"surrogate_team_keys"`
			TeamKeys          []string `json:"team_keys"`
		} `json:"blue"`
		Red struct {
			DqTeamKeys        []string `json:"dq_team_keys"`
			Score             int      `json:"score"`
			SurrogateTeamKeys []string `json:"surrogate_team_keys"`
			TeamKeys          []string `json:"team_keys"`
		} `json:"red"`
	} `json:"alliances"`
	CompLevel      string `json:"comp_level"`
	EventKey       string `json:"event_key"`
	Key            string `json:"key"`
	MatchNumber    int    `json:"match_number"`
	PostResultTime int    `json:"post_result_time"`
	PredictedTime  int    `json:"predicted_time"`
	ScoreBreakdown struct {
		Blue struct {
			AdjustPoints                        int    `json:"adjustPoints"`
			AutoAmpNoteCount                    int    `json:"autoAmpNoteCount"`
			AutoAmpNotePoints                   int    `json:"autoAmpNotePoints"`
			AutoLeavePoints                     int    `json:"autoLeavePoints"`
			AutoLineRobot1                      string `json:"autoLineRobot1"`
			AutoLineRobot2                      string `json:"autoLineRobot2"`
			AutoLineRobot3                      string `json:"autoLineRobot3"`
			AutoPoints                          int    `json:"autoPoints"`
			AutoSpeakerNoteCount                int    `json:"autoSpeakerNoteCount"`
			AutoSpeakerNotePoints               int    `json:"autoSpeakerNotePoints"`
			AutoTotalNotePoints                 int    `json:"autoTotalNotePoints"`
			CoopNotePlayed                      bool   `json:"coopNotePlayed"`
			CoopertitionBonusAchieved           bool   `json:"coopertitionBonusAchieved"`
			CoopertitionCriteriaMet             bool   `json:"coopertitionCriteriaMet"`
			EndGameHarmonyPoints                int    `json:"endGameHarmonyPoints"`
			EndGameNoteInTrapPoints             int    `json:"endGameNoteInTrapPoints"`
			EndGameOnStagePoints                int    `json:"endGameOnStagePoints"`
			EndGameParkPoints                   int    `json:"endGameParkPoints"`
			EndGameRobot1                       string `json:"endGameRobot1"`
			EndGameRobot2                       string `json:"endGameRobot2"`
			EndGameRobot3                       string `json:"endGameRobot3"`
			EndGameSpotLightBonusPoints         int    `json:"endGameSpotLightBonusPoints"`
			EndGameTotalStagePoints             int    `json:"endGameTotalStagePoints"`
			EnsembleBonusAchieved               bool   `json:"ensembleBonusAchieved"`
			EnsembleBonusOnStageRobotsThreshold int    `json:"ensembleBonusOnStageRobotsThreshold"`
			EnsembleBonusStagePointsThreshold   int    `json:"ensembleBonusStagePointsThreshold"`
			FoulCount                           int    `json:"foulCount"`
			FoulPoints                          int    `json:"foulPoints"`
			G206Penalty                         bool   `json:"g206Penalty"`
			G408Penalty                         bool   `json:"g408Penalty"`
			G424Penalty                         bool   `json:"g424Penalty"`
			MelodyBonusAchieved                 bool   `json:"melodyBonusAchieved"`
			MelodyBonusThreshold                int    `json:"melodyBonusThreshold"`
			MelodyBonusThresholdCoop            int    `json:"melodyBonusThresholdCoop"`
			MelodyBonusThresholdNonCoop         int    `json:"melodyBonusThresholdNonCoop"`
			MicCenterStage                      bool   `json:"micCenterStage"`
			MicStageLeft                        bool   `json:"micStageLeft"`
			MicStageRight                       bool   `json:"micStageRight"`
			Rp                                  int    `json:"rp"`
			TechFoulCount                       int    `json:"techFoulCount"`
			TeleopAmpNoteCount                  int    `json:"teleopAmpNoteCount"`
			TeleopAmpNotePoints                 int    `json:"teleopAmpNotePoints"`
			TeleopPoints                        int    `json:"teleopPoints"`
			TeleopSpeakerNoteAmplifiedCount     int    `json:"teleopSpeakerNoteAmplifiedCount"`
			TeleopSpeakerNoteAmplifiedPoints    int    `json:"teleopSpeakerNoteAmplifiedPoints"`
			TeleopSpeakerNoteCount              int    `json:"teleopSpeakerNoteCount"`
			TeleopSpeakerNotePoints             int    `json:"teleopSpeakerNotePoints"`
			TeleopTotalNotePoints               int    `json:"teleopTotalNotePoints"`
			TotalPoints                         int    `json:"totalPoints"`
			TrapCenterStage                     bool   `json:"trapCenterStage"`
			TrapStageLeft                       bool   `json:"trapStageLeft"`
			TrapStageRight                      bool   `json:"trapStageRight"`
		} `json:"blue"`
		Red struct {
			AdjustPoints                        int    `json:"adjustPoints"`
			AutoAmpNoteCount                    int    `json:"autoAmpNoteCount"`
			AutoAmpNotePoints                   int    `json:"autoAmpNotePoints"`
			AutoLeavePoints                     int    `json:"autoLeavePoints"`
			AutoLineRobot1                      string `json:"autoLineRobot1"`
			AutoLineRobot2                      string `json:"autoLineRobot2"`
			AutoLineRobot3                      string `json:"autoLineRobot3"`
			AutoPoints                          int    `json:"autoPoints"`
			AutoSpeakerNoteCount                int    `json:"autoSpeakerNoteCount"`
			AutoSpeakerNotePoints               int    `json:"autoSpeakerNotePoints"`
			AutoTotalNotePoints                 int    `json:"autoTotalNotePoints"`
			CoopNotePlayed                      bool   `json:"coopNotePlayed"`
			CoopertitionBonusAchieved           bool   `json:"coopertitionBonusAchieved"`
			CoopertitionCriteriaMet             bool   `json:"coopertitionCriteriaMet"`
			EndGameHarmonyPoints                int    `json:"endGameHarmonyPoints"`
			EndGameNoteInTrapPoints             int    `json:"endGameNoteInTrapPoints"`
			EndGameOnStagePoints                int    `json:"endGameOnStagePoints"`
			EndGameParkPoints                   int    `json:"endGameParkPoints"`
			EndGameRobot1                       string `json:"endGameRobot1"`
			EndGameRobot2                       string `json:"endGameRobot2"`
			EndGameRobot3                       string `json:"endGameRobot3"`
			EndGameSpotLightBonusPoints         int    `json:"endGameSpotLightBonusPoints"`
			EndGameTotalStagePoints             int    `json:"endGameTotalStagePoints"`
			EnsembleBonusAchieved               bool   `json:"ensembleBonusAchieved"`
			EnsembleBonusOnStageRobotsThreshold int    `json:"ensembleBonusOnStageRobotsThreshold"`
			EnsembleBonusStagePointsThreshold   int    `json:"ensembleBonusStagePointsThreshold"`
			FoulCount                           int    `json:"foulCount"`
			FoulPoints                          int    `json:"foulPoints"`
			G206Penalty                         bool   `json:"g206Penalty"`
			G408Penalty                         bool   `json:"g408Penalty"`
			G424Penalty                         bool   `json:"g424Penalty"`
			MelodyBonusAchieved                 bool   `json:"melodyBonusAchieved"`
			MelodyBonusThreshold                int    `json:"melodyBonusThreshold"`
			MelodyBonusThresholdCoop            int    `json:"melodyBonusThresholdCoop"`
			MelodyBonusThresholdNonCoop         int    `json:"melodyBonusThresholdNonCoop"`
			MicCenterStage                      bool   `json:"micCenterStage"`
			MicStageLeft                        bool   `json:"micStageLeft"`
			MicStageRight                       bool   `json:"micStageRight"`
			Rp                                  int    `json:"rp"`
			TechFoulCount                       int    `json:"techFoulCount"`
			TeleopAmpNoteCount                  int    `json:"teleopAmpNoteCount"`
			TeleopAmpNotePoints                 int    `json:"teleopAmpNotePoints"`
			TeleopPoints                        int    `json:"teleopPoints"`
			TeleopSpeakerNoteAmplifiedCount     int    `json:"teleopSpeakerNoteAmplifiedCount"`
			TeleopSpeakerNoteAmplifiedPoints    int    `json:"teleopSpeakerNoteAmplifiedPoints"`
			TeleopSpeakerNoteCount              int    `json:"teleopSpeakerNoteCount"`
			TeleopSpeakerNotePoints             int    `json:"teleopSpeakerNotePoints"`
			TeleopTotalNotePoints               int    `json:"teleopTotalNotePoints"`
			TotalPoints                         int    `json:"totalPoints"`
			TrapCenterStage                     bool   `json:"trapCenterStage"`
			TrapStageLeft                       bool   `json:"trapStageLeft"`
			TrapStageRight                      bool   `json:"trapStageRight"`
		} `json:"red"`
	} `json:"score_breakdown"`
	SetNumber       int    `json:"set_number"`
	Time            int    `json:"time"`
	Videos          []any  `json:"videos"`
	WinningAlliance string `json:"winning_alliance"`
}

type Event struct {
	Alliance          any    `json:"alliance"`
	AllianceStatusStr string `json:"alliance_status_str"`
	LastMatchKey      string `json:"last_match_key"`
	NextMatchKey      string `json:"next_match_key"`
	OverallStatusStr  string `json:"overall_status_str"`
	Playoff           any    `json:"playoff"`
	PlayoffStatusStr  string `json:"playoff_status_str"`
	Qual              struct {
		NumTeams int `json:"num_teams"`
		Ranking  struct {
			Dq            int `json:"dq"`
			MatchesPlayed int `json:"matches_played"`
			QualAverage   any `json:"qual_average"`
			Rank          int `json:"rank"`
			Record        struct {
				Losses int `json:"losses"`
				Ties   int `json:"ties"`
				Wins   int `json:"wins"`
			} `json:"record"`
			SortOrders []float64 `json:"sort_orders"`
			TeamKey    string    `json:"team_key"`
		} `json:"ranking"`
		SortOrderInfo []struct {
			Name      string `json:"name"`
			Precision int    `json:"precision"`
		} `json:"sort_order_info"`
		Status string `json:"status"`
	} `json:"qual"`
}

type Team struct {
	Key        string `json:"key"`
	TeamNumber int    `json:"team_number"`
	Nickname   string `json:"nickname"`
	Name       string `json:"name"`
	City       string `json:"city"`
	StateProv  string `json:"state_prov"`
	Country    string `json:"country"`
}

type TbaHandler struct {
    tbaToken string
}

func NewHandler(tbaToken string) *TbaHandler {
    handler := &TbaHandler{
        tbaToken: tbaToken,
    }
    return handler
}

func (t *TbaHandler) makeRequest(url string) []byte {
    slog.Info("Making TBA request", "Url", url)
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
func (t *TbaHandler) MakeMatchListReq(teamId string, eventId string) []Match {
    url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/matches"
    jsonData := t.makeRequest(url)
    var matches []Match
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

func (t *TbaHandler) MakeMatchReq(matchId string) Match {
    url := BASE_URL + "match/" + matchId
    var match Match
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

func (t *TbaHandler) MakeTeamEventStatusRequest(teamId string, eventId string) Event {
    url := BASE_URL + "team/" + teamId + "/event/" + eventId + "/status"
    var event Event
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &event)
    return event
}

func (t *TbaHandler) MakeTeamsAtEventRequest(eventId string) []Team {
    url := BASE_URL + "event/" + eventId + "/teams/simple"
    var teams []Team
    jsonData := t.makeRequest(url)
    json.Unmarshal(jsonData, &teams)
    return teams
}
