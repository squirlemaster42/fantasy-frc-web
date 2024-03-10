package model

import (
	"server/database"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
    Id int
    Email string
    Password string
    Username string
}

func CreateUser(user User, dbHandler database.DatabaseDriver) error {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), 8)
    if err != nil {
        return err
    }

    stmt := `INSERT INTO Players (name, password) Values ($1, $2);`

    _, err = dbHandler.Connection.Exec(stmt, user.Username, string(hashedPassword))

    return err
}
