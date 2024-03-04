package database

import (
	"database/sql"
	"log"

    _ "github.com/lib/pq"
)

type DatabaseDriver struct {
    Username string
    Password string
    Ip string
    DbName string
    Connection *sql.DB
}

func CreateDatabaseDriver(username string, password string, ip string, dbName string) *DatabaseDriver{
    driver := DatabaseDriver{
        Username: username,
        Password: password,
        Ip: ip,
        DbName: dbName,
    }

    connStr := driver.createConnectionString()

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal(err)
    }
    driver.Connection = db

    return &driver
}

func (driver *DatabaseDriver) createConnectionString() string {
    return "postgresql://" + driver.Username + ":" + driver.Password + "@" + driver.Ip + "/" + driver.DbName + "?sslmode=disable"
}

func (driver *DatabaseDriver) RunQuery(query string) *sql.Rows {
    rows, err := driver.Connection.Query(query)
    if err != nil {
        log.Fatal(err)
    }

    return rows
}

func (driver *DatabaseDriver) RunExec(query string) {
    _, err := driver.Connection.Exec(query)
    if err != nil {
        log.Fatal(err)
    }
}
