package model

import "database/sql"

//TODO The primary key here is not good. I should fix that.
type User struct {
    Id int
    Username string
    Password string
}

func RegisterUser(database *sql.DB, user User) error {
    return nil
}

func UsernameTaken(database *sql.DB, username string) bool {
    return false
}

func ValidateLogin(database *sql.DB, username string, password string) bool {
    return false
}

func UpdatePassword(database *sql.DB, username string, oldPassword string, newPassword string) bool {
    return false
}
