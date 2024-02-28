package database

import "database/sql"

type DatabaseDriver struct {
    Username string
    Password string
    Ip string
    DbName string
    Connection *sql.DB //TODO Give this a type
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
        return nil
    }
    driver.Connection = db

    return &driver
}

func (driver *DatabaseDriver) createConnectionString() string {
    return "postgresql://" + driver.Username + ":" + driver.Password + "@" + driver.Ip + "/" + driver.DbName + "?sslmode=disable"
}

//TODO Add a type
func (driver *DatabaseDriver) RunQuery(query string) *sql.Rows {
    rows, err := driver.Connection.Query(query)
    if err != nil {
        return nil
    }

    return rows
}

func (driver *DatabaseDriver) RunExec(query string) {
    driver.Connection.Exec(query)
}
