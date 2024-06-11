package model

import (
	"database/sql"
	"server/assert"
)

//TODO The primary key here is not good. I should fix that.
type User struct {
    Id int
    Username string
    Password string
}

func RegisterUser(database *sql.DB, username string, password string) {
    query := `INSERT INTO Users (username, password) Value($1, $2);`
    assert := assert.CreateAssertWithContext("Register User")
    assert.AddContext("Username", username)
    assert.AddContext("Password", password)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(username, password)
    assert.NoError(err, "Failed to register user")
}

func UsernameTaken(database *sql.DB, username string) bool {
    query := `Select count(Id) From Users Where username = $1;`
    assert := assert.CreateAssertWithContext("Username Taken")
    assert.AddContext("Username", username)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var count int
    err = stmt.QueryRow(username).Scan(&count)
    assert.NoError(err, "Failed to check username taken")
    return count > 0
}

//All crypto should happen before this since this just communicates with the DB
//TODO Should the crypto be rolled into this?
func ValidateLogin(database *sql.DB, username string, password string) bool {
    query := `Select password From Users Where username = $1;`
    assert := assert.CreateAssertWithContext("Validate Login")
    assert.AddContext("Username", username)
    assert.AddContext("Password", password)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    var dbPassword string
    err = stmt.QueryRow(username).Scan(&dbPassword)
    assert.NoError(err, "Failed to validate login")
    return dbPassword == password
}

//The old password logic should happen before this
//TODO Should we move more logic here?
func UpdatePassword(database *sql.DB, username string, newPassword string) {
    query := `Update Users Set password = $1 Where username = $2;`
    assert := assert.CreateAssertWithContext("Update Password")
    assert.AddContext("Username", username)
    assert.AddContext("New Password", newPassword)
    stmt, err := database.Prepare(query)
    assert.NoError(err, "Failed to prepare statement")
    _, err = stmt.Exec(newPassword, username)
    assert.NoError(err, "Failed to Update Password")
}
