package model

import (
	"log"
	"server/database"
)

type Pick struct {
    PlayerId int
    PickOrder int
    PlayerOrder int
    DraftId int
    PickedTeam int
}

func MakePick(playerId int, pickOrder int, playerOrder int, draftId int, pickedTeam string, dbDriver *database.DatabaseDriver) {
    query := `INSERT INTO Picks (playerId, pickOrder, playerOrder, draftId, pickedTeam)
    VALUES ($1, $2, $3, $4, $5)
    ON CONFLICT (playerId, pickOrder, playerOrder, draftId)
    DO UPDATE SET
    pickedTeam = EXCLUDED.pickedTeam;`

    stmt, err := dbDriver.Connection.Prepare(query)

    if err != nil {
        log.Println(err)
        return
    }

    _, err = stmt.Exec(playerId, pickOrder, playerOrder, draftId, pickedTeam)

    if err != nil {
        log.Println(err)
    }
}
