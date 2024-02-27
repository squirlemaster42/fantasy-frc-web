package database

import "database/sql"

type DatabaseDriver struct {
    username string
    password string
    ip string
    dbName string
    connection *sql.DB //TODO Give this a type
}

func CreateDatabaseDriver(username string, password string, ip string, dbName string) *DatabaseDriver{
    driver := DatabaseDriver{
        username: username, password: password,
        ip: ip,
        dbName: dbName,
    }

    connStr := driver.createConnectionString()

    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil
    }
    driver.connection = db

    return &driver
}

func (driver *DatabaseDriver) createConnectionString() string {
    return "postgresql://" + driver.username + ":" + driver.password + "@" + driver.ip + "/" + driver.dbName + "?sslmode=disable"
}

//TODO Add a type
func (driver *DatabaseDriver) runQuery(query string) any {
    rows, err := driver.connection.Query(query)
    if err != nil {
        return nil
    }

    return rows
}

func (driver *DatabaseDriver) runExec(query string) {
    driver.connection.Exec(query)
}
