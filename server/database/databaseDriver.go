package database

import (
	"database/sql"
	"server/assert"

	_ "github.com/lib/pq"
)

func RegisterDatabaseConnection(username string, password string, ip string, dbName string) *sql.DB{
    connStr := createConnectionString(username, password, ip, dbName)

    a := assert.CreateAssertWithContext("Register DB")

    db, err := sql.Open("postgres", connStr)
    a.NoError(err, "Could not open database connection")
    a.NoError(db.Ping(), "Failed to ping database")

    return db
}

func checkDbVersion(database *sql.DB) bool {
    return false;
}

func createConnectionString(username string, password string, ip string, dbName string) string {
    return "postgresql://" + username + ":" + password + "@" + ip + "/" + dbName + "?sslmode=disable"
}
