package main

import (
	"database/sql"
	"log/slog"
	"os"
	"server/assert"
    "server/swagger"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func RegisterDatabaseConnection(username string, password string, ip string, dbName string) *sql.DB{
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
    validTeams := getValidTeams(database)
    createFuzzyMatch(validTeams)
}

func getValidTeams(database *sql.DB) map[string]bool {
    assert := assert.CreateAssertWithContext("Get Valid Teams")

    query := `Select
        tbaid
    From Teams;`
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    rows, err := stmt.Query()
    assert.NoError(err, "Failed to query valid teams")

    validTeams := make(map[string]bool)
    for rows.Next() {
        var team string
        rows.Scan(&team)
        validTeams[team] = true
    }

    return validTeams
}

func createFuzzyMatch(validTeams map[string]bool) swagger.Match {
    return swagger.Match{}
}
