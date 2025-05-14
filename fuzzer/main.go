package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"server/assert"
	"server/swagger"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var validCompLevels []string = []string {
    "q",
    "qf",
    "sf",
    "f",
}

func RegisterDatabaseConnection(username string, password string, ip string, dbName string) *sql.DB{
    slog.Info("Setting up DB connection", "User", username, "Ip", ip, "Database Name", dbName)
    connStr := createConnectionString(username, password, ip, dbName)

    a := assert.CreateAssertWithContext("Register DB")

    db, err := sql.Open("postgres", connStr)
    a.NoError(err, "Could not open database connection")
    a.NoError(db.Ping(), "Failed to ping database")

    fmt.Println(setupRandomMatch())
    fmt.Println(setupRandomMatch())
    fmt.Println(setupRandomMatch())
    fmt.Println(setupRandomMatch())
    fmt.Println(setupRandomMatch())

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
    validTeams := getValidTeams(database)
    createFuzzyMatch(validTeams)
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
        rows.Scan(&team)
        validTeams = append(validTeams, team)
    }

    return validTeams
}

func createFuzzyMatch(validTeams []string) swagger.Match {
    match := setupRandomMatch()
    match.Alliances.Red = createMatchAlliance(validTeams)
    match.Alliances.Blue = createMatchAlliance(validTeams)

    return match
}

func createMatchAlliance(validTeams []string) *swagger.MatchAlliance {
    return nil
}

const alphabet string = "abcdefghijklmnopqrstuvwxyz"

func setupRandomMatch() swagger.Match {
    match := swagger.Match{
        CompLevel: validCompLevels[rand.Intn(len(validCompLevels))],
        MatchNumber: rand.Int31n(999) + 1,
    }
    sb := strings.Builder{}
    year :=  rand.Intn(3000) + 1
    sb.WriteString(strconv.Itoa(year))
    sb.WriteString(getRandomString(rand.Intn(4) + 2, alphabet))
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
