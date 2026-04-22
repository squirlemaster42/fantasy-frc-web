package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"server/assert"
	"server/swagger"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var validCompLevels []string = []string{
	"q",
	"qf",
	"sf",
	"f",
}

func RegisterDatabaseConnection(username string, password string, ip string, dbName string) *sql.DB {
	slog.Info("Setting up DB connection", "User", username, "Ip", ip, "Database Name", dbName)
	connStr := createConnectionString(username, password, ip, dbName)

	a := assert.CreateAssertWithContext("Register DB")

	db, err := sql.Open("postgres", connStr)
	a.NoError(err, "Could not open database connection")
	a.NoError(db.Ping(), "Failed to ping database")

	return db
}

func createConnectionString(username string, password string, ip string, dbName string) string {
	return "postgresql://" + username + ":" + password + "@" + ip + "/" + dbName + "?sslmode=disable"
}

func main() {
	assert := assert.CreateAssertWithContext("Main")
	err := godotenv.Load()
	assert.NoError(err, "Failed to load env vars")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbUsername := os.Getenv("DB_USERNAME")
	dbIp := os.Getenv("DB_IP")
	dbName := os.Getenv("DB_NAME")
	slog.Info("Extracted Env Vars")
	database := RegisterDatabaseConnection(dbUsername, dbPassword, dbIp, dbName)

	secret := os.Getenv("TBA_WEBHOOK_SECRET")
	targetUrl := "http://localhost:8080/tbaWebhook"
	validTeams := getValidTeams(database)
	for range 1000 {
		match := createFuzzyMatch(validTeams)
		sendUpcomingMatch(match, targetUrl, secret)
		sendMatchScore(match, targetUrl, secret)
		// discord ratelimits, this could be shorter if necessary but 1 second works
		time.Sleep(time.Second)
	}
}

type WebhookMessage struct {
	MessageType string          `json:"message_type"`
	MessageData json.RawMessage `json:"message_data"`
}

func sendFuzzyTBAPostRequest(body []byte, targetUrl string, secret string) {
	mac := hmac.New(sha256.New, []byte(secret))
	_, err := mac.Write(body)
	assert.NoErrorCF(err, "Failed to write data to mac")
	macToSend := mac.Sum(nil)

	req, err := http.NewRequest("POST", targetUrl, bytes.NewBuffer(body))
	assert.NoErrorCF(err, "Failed to create post request")
	slog.Info("Created request")
	slog.Info("Adding hmac to msg", "HMAC", hex.EncodeToString(macToSend))
	req.Header.Set("X-TBA-HMAC", hex.EncodeToString(macToSend))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	slog.Info("Made request")
	assert.NoErrorCF(err, "Failed to make post request")

	err = resp.Body.Close()
	assert.NoErrorCF(err, "Failed to close response body")
}

func sendUpcomingMatch(match swagger.Match, targetUrl string, secret string) {
	slog.Info("Starting to send upcoming match notification")

	messageData := UpcomingMatchEvent{
		EventKey:      match.EventKey,
		MatchKey:      match.Key,
		TeamKey:       match.Alliances.Red.TeamKeys[0],
		EventName:     match.EventKey,
		TeamKeys:      append(match.Alliances.Red.TeamKeys, match.Alliances.Blue.TeamKeys...),
		ScheduledTime: match.ActualTime,
		PredictedTime: match.PredictedTime,
	}
	messageJson, err := json.Marshal(messageData)
	assert.NoErrorCF(err, "Failed to marshal upcoming match message data")
	notification := WebhookMessage{
		MessageType: "upcoming_match",
		MessageData: messageJson,
	}

	serialized, err := json.Marshal(notification)
	assert.NoErrorCF(err, "Failed to marshal upcoming match message data")
	sendFuzzyTBAPostRequest(serialized, targetUrl, secret)
	slog.Info("Sent upcoming match notification", "MatchKey", match.Key)
}

func sendMatchScore(match swagger.Match, targetUrl string, secret string) {
	slog.Info("Starting to send score notification")
	messageData := MatchScoreNofification{
		EventKey:  match.EventKey,
		MatchKey:  match.Key,
		TeamKey:   match.Alliances.Red.TeamKeys[0],
		EventName: match.EventKey,
		Match:     match,
	}

	serializedNotification, err := json.Marshal(messageData)
	assert.NoErrorCF(err, "Failed to marshal score message data")
	scoreNotification := WebhookMessage{
		MessageType: "match_score",
		MessageData: serializedNotification,
	}

	serialized, err := json.Marshal(scoreNotification)
	assert.NoErrorCF(err, "Failed to marshal score notification")
	sendFuzzyTBAPostRequest(serialized, targetUrl, secret)
	slog.Info("Sent match score", "MatchKey", match.Key)
}

type MatchScoreNofification struct {
	EventKey  string        `json:"event_key"`
	MatchKey  string        `json:"match_key"`
	TeamKey   string        `json:"team_key"`
	EventName string        `json:"event_name"`
	Match     swagger.Match `json:"match"`
}

type UpcomingMatchEvent struct {
	EventKey      string          `json:"event_key"`
	MatchKey      string          `json:"match_key"`
	TeamKey       string          `json:"team_key"`
	EventName     string          `json:"event_name"`
	TeamKeys      []string        `json:"team_keys"`
	ScheduledTime int64           `json:"scheduled_time"`
	PredictedTime int64           `json:"predicted_time"`
	Webcast       swagger.Webcast `json:"webcast"`
}

func getValidTeams(database *sql.DB) []string {
	assert := assert.CreateAssertWithContext("Get Valid Teams")

	query := `Select
        tbaid
    From Teams;`
	stmt, err := database.Prepare(query)
	assert.NoError(err, "Failed to prepare statement")
	rows, err := stmt.Query()
	assert.NoError(err, "Failed to query valid teams")

	var validTeams []string
	for rows.Next() {
		var team string
		err := rows.Scan(&team)
		assert.NoError(err, "Failed to get team")
		validTeams = append(validTeams, team)
	}

	return validTeams
}

func createFuzzyMatch(validTeams []string) swagger.Match {
	match := setupRandomMatch()
	match.Alliances = &swagger.MatchSimpleAlliances{}

	match.Alliances.Red = createMatchAlliance(validTeams)
	match.Alliances.Blue = createMatchAlliance(validTeams)

	if match.Alliances.Red.Score > match.Alliances.Blue.Score {
		match.WinningAlliance = "red"
	} else if match.Alliances.Red.Score < match.Alliances.Blue.Score {
		match.WinningAlliance = "blue"
	} else {
		match.WinningAlliance = ""
	}

	match.Time = rand.Int63n(100000000000)
	match.ActualTime = rand.Int63n(10000) + match.Time
	match.PredictedTime = rand.Int63n(10000) + match.Time
	match.PostResultTime = rand.Int63n(10000) + match.Time

	match.ScoreBreakdown = &swagger.OneOfMatchScoreBreakdown{
		MatchScoreBreakdown2026: swagger.MatchScoreBreakdown2026{
			Red:  createScoreBreakdown(match.Alliances.Red.Score),
			Blue: createScoreBreakdown(match.Alliances.Blue.Score),
		},
	}

	return match
}

func createScoreBreakdown(totalScore int32) *swagger.MatchScoreBreakdown2026Alliance {
	scoreBreakdown := swagger.MatchScoreBreakdown2026Alliance{
		AdjustPoints: totalScore,
	}

	//For now we only care about the rps, maybe ill add more some day
	if rand.Intn(2) == 1 {
		scoreBreakdown.EnergizedAchieved = true
	} else {
		scoreBreakdown.EnergizedAchieved = false
	}
	if scoreBreakdown.EnergizedAchieved && rand.Intn(2) == 1 {
		scoreBreakdown.SuperchargedAchieved = true
	} else {
		scoreBreakdown.SuperchargedAchieved = false
	}
	if rand.Intn(2) == 1 {
		scoreBreakdown.TraversalAchieved = true
	} else {
		scoreBreakdown.TraversalAchieved = false
	}

	return &scoreBreakdown
}

func createMatchAlliance(validTeams []string) *swagger.MatchAlliance {
	alliance := swagger.MatchAlliance{
		Score:             rand.Int31n(500),
		TeamKeys:          []string{},
		SurrogateTeamKeys: []string{},
		DqTeamKeys:        []string{},
	}

	for range 3 {
		alliance.TeamKeys = append(alliance.TeamKeys, validTeams[rand.Intn(len(validTeams))])
	}

	for _, team := range alliance.TeamKeys {
		if rand.Intn(50) == 0 {
			alliance.SurrogateTeamKeys = append(alliance.SurrogateTeamKeys, team)
		}
		if rand.Intn(50) == 0 {
			alliance.DqTeamKeys = append(alliance.DqTeamKeys, team)
		}
	}

	return &alliance
}

const alphabet string = "abcdefghijklmnopqrstuvwxyz"

func setupRandomMatch() swagger.Match {
	match := swagger.Match{
		CompLevel:   validCompLevels[rand.Intn(len(validCompLevels))],
		MatchNumber: rand.Int31n(999) + 1,
		EventKey:    getRandomString(rand.Intn(4)+2, alphabet),
	}
	sb := strings.Builder{}
	year := rand.Intn(3000) + 1
	sb.WriteString(strconv.Itoa(year))
	sb.WriteString(match.EventKey)
	sb.WriteString("_")
	sb.WriteString(match.CompLevel)
	if match.CompLevel != "q" {
		match.SetNumber = rand.Int31n(3) + 1
		sb.WriteString(strconv.Itoa(int(match.SetNumber)))
	}
	sb.WriteString("m")
	sb.WriteString(strconv.Itoa(int(match.MatchNumber)))
	match.Key = sb.String()
	return match
}

func getRandomString(length int, charset string) string {
	assert := assert.CreateAssertWithContext("Get Random String")
	assert.AddContext("Length", length)
	assert.AddContext("Charset", charset)
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
