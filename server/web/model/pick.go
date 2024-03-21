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

func GetNextPlayerPickOrder(draftId int, dbDriver *database.DatabaseDriver) (int, int) {
    query := `Select PlayerOrder, PickOrder From Picks
    Inner Join (Select Max(Id) As Id From Picks Where draftId = $1) m On m.Id = Picks.Id;`
    stmt, err := dbDriver.Connection.Prepare(query)

    if err != nil {
        log.Println(err)
    }

    var playerOrder int
    var pickOrder int
    err = stmt.QueryRow(draftId).Scan(&playerOrder, &pickOrder)

    return 0, 0
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

func GetPick(playerId int, draftId int, dbDriver *database.DatabaseDriver) string {
    query := `Select pickedTeam From Picks Where playerId = $1 AND draftId = $2`
    stmt, err := dbDriver.Connection.Prepare(query)

    var pick string
    if err != nil {
        log.Println(err)
    }

    err = stmt.QueryRow(playerId, draftId).Scan(&pick)

    if err != nil {
        log.Println(err)
    }

    return pick
}
