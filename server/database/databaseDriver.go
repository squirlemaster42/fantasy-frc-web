package database

import (
	"database/sql"
	"os"
	"server/assert"
	"server/log"
	"strconv"
	"time"

	_ "github.com/lib/pq"
)

func RegisterDatabaseConnection(username string, password string, ip string, dbName string) *sql.DB {
	log.InfoNoContext("Setting up DB connection", "User", username, "Ip", ip, "Database Name", dbName)
	connStr := createConnectionString(username, password, ip, dbName)

	a := assert.CreateAssertWithContext("Register DB")

	db, err := sql.Open("postgres", connStr)
	a.NoError(err, "Could not open database connection")
	a.NoError(db.Ping(), "Failed to ping database")

	maxConns := getEnvInt("DB_MAX_CONNS", 25)
	maxIdleConns := getEnvInt("DB_MAX_IDLE_CONNS", 10)
	connMaxLifetime := getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 5)

	db.SetMaxOpenConns(maxConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)

	log.InfoNoContext("Database connection pool configured",
		"MaxOpenConns", maxConns,
		"MaxIdleConns", maxIdleConns,
		"ConnMaxLifetimeMinutes", connMaxLifetime,
	)

	return db
}

func createConnectionString(username string, password string, ip string, dbName string) string {
	return "postgresql://" + username + ":" + password + "@" + ip + "/" + dbName + "?sslmode=disable"
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}
