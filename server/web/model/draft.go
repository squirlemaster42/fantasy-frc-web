package model

import (
	"fmt"
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
    defer stmt.Close()

    if err != nil {
        log.Fatalln(err)
    }

    pickQuery := `SELECT
    pa.name,
    pi.pickedTeam,
    pi.playerOrder,
    pi.pickOrder
    FROM Picks pi
    LEFT JOIN Players pa ON pa.id = pi.player
    WHERE draftId = $1
    ORDER BY pi.playerOrder, pi.pickOrder;
    `

    pickStmt, err := dbDriver.Connection.Prepare(pickQuery)
    defer pickStmt.Close()

    if err != nil {
        log.Fatalln(err)
    }

    err = stmt.QueryRow(draftId).Scan(&draftName)

    if err != nil {
        log.Fatalln(err)
    }

    rows, err := pickStmt.Query(draftId)
    defer rows.Close()

    if err != nil {
        log.Fatalln(err)
    }

    draft := Draft {
        Name: draftName,
    }

    draft.Players = make([]struct{Name string; Picks []string}, 8)

    for rows.Next() {
        var playerName string
        var pickedTeam string
        var playerOrder int
        var pickOrder int

        err = rows.Scan(&playerName, &pickedTeam, &playerOrder, &pickOrder)

        if err != nil {
            log.Fatalln(err)
        }

        draft.Players[playerOrder].Name = playerName

        if draft.Players[playerOrder].Picks == nil {
            draft.Players[playerOrder].Picks = make([]string, 8)
        }

        draft.Players[playerOrder].Picks[pickOrder] = pickedTeam[3:]
    }


    return &draft
}

func CheckIfDraftExists(draftId int, dbDriver *database.DatabaseDriver) bool {
    query := `Select Id From Drafts Where Id = $1`
    stmt, err := dbDriver.Connection.Prepare(query)

    if err != nil {
        fmt.Println(err)
        return false
    }

    var id int
    err = stmt.QueryRow(draftId).Scan(&id)

    if err != nil {
        fmt.Println(err)
        return false
    }

    return true
}

//Creates a draft where the pick order is the order of the players array
//The array uses the player ids
//Return the created draft id
func CreateDraft(draftName string, players []int, dbDriver *database.DatabaseDriver) (int, error) {
    //Create the draft
    query := `INSERT INTO Drafts (Name) Values ($1) RETURNING Id;`
    stmt, err := dbDriver.Connection.Prepare(query)

    //TODO Do we want to return friendly errors here or handle that when we make the frontend?
    if err != nil {
        return -1, err
    }

    var draftId int
    err = stmt.QueryRow(draftName).Scan(&draftId)

    if err != nil {
        return -1, err
    }

    //Add players to DraftPlayers
    for order, player := range players {
        draftPlayerQuery := `INSERT INTO DraftPlayers (draftId, player, playerOrder) VALUES ($1, $2, $3);`
        stmt, err = dbDriver.Connection.Prepare(draftPlayerQuery)
        _, err = stmt.Exec(draftId, player, order)
    }

    //Set the current player in the draft
    firstPlayer := players[0]
    query = `UPDATE Draft SET currentPlayer = $1 WHERE Id = $2`
    stmt, err = dbDriver.Connection.Prepare(query)

    if err != nil {
        return -1, err
    }

    stmt.Exec(firstPlayer, draftId)

    return draftId, nil
}
