package model

import (
	"log"
	"server/database"
)

type Draft struct {
    Name string
    Players []struct {
        Name string
        Picks []string
    }
}

func LoadDraftFromDatabase (draftId int, dbDriver *database.DatabaseDriver) *Draft {
    var draftName string
    query := `SELECT name FROM Drafts WHERE id = $1`
    stmt, err := dbDriver.Connection.Prepare(query)

    if err != nil {
        log.Fatalln(err)
    }

    defer stmt.Close()

    err = stmt.QueryRow(draftId).Scan(&draftName)

    draftMode := Draft {
        Name: draftName,
    }
    log.Println(draftMode)

    return nil
}
