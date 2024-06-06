package database

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type DatabaseDriver struct {
    Connection *sql.DB
}

func RegisterDatabaseConnection(username string, password string, ip string, dbName string) *sql.DB{
    connStr := createConnectionString(username, password, ip, dbName)

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal(err)
    }

    return db
}

func createConnectionString(username string, password string, ip string, dbName string) string {
    return "postgresql://" + username + ":" + password + "@" + ip + "/" + dbName + "?sslmode=disable"
}
