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
