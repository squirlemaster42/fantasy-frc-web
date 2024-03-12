package model

import (
	"log"
	"server/database"

	"golang.org/x/crypto/bcrypt"
)

type Player struct {
    Id int
    Username string
    Password string
}

func CreateUser(player Player, dbHandler database.DatabaseDriver) error {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(player.Password), 8)
    if err != nil {
        return err
    }

    stmt := `INSERT INTO Players (name, password) Values ($1, $2);`

    _, err = dbHandler.Connection.Exec(stmt, player.Username, string(hashedPassword))

    return err
}

func GetPlayerById(id string, dbHandler database.DatabaseDriver) (Player, error) {
    query := `SELECT * FROM Players WHERE id=$1`

    stmt, err := dbHandler.Connection.Prepare(query)

    defer stmt.Close()

    var player Player
    err = stmt.QueryRow(id).Scan(&player.Id, &player.Username, &player.Password)
    if err != nil {
        return Player{}, err
    }

    return player, nil
}

func ValidateUserLogin(username string, password string, dbHandler database.DatabaseDriver) bool {
    var dbPassword string
    query := `SELECT password FROM Players WHERE name=$1`

    stmt, err := dbHandler.Connection.Prepare(query)
    defer stmt.Close()

    err = stmt.QueryRow(username).Scan(&dbPassword)

    if err != nil {
        log.Printf("Login attempt for %s failed because user does not exist\n", username)
        log.Println(err)
        return false
    }

    err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))

    if err != nil {
        return false
    }

    return true
}
