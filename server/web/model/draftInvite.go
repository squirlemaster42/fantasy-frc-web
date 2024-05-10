package model

import "server/database"

func InvitePlayerToDraft(draftId int, playerId int, invitingPlayer int, dbDriver *database.DatabaseDriver) error {
    query := `INSERT INTO DraftInvites (draftId, invitedPlayer, invitingPlayer, accepted) Values ($1, $2, $3, $4);`
    stmt, err := dbDriver.Connection.Prepare(query)

    if err != nil {
        return err
    }

    _, err = stmt.Exec(draftId, playerId, invitingPlayer, false)

    return err
}

func AcceptInvite(draftId int, playerId int, dbDriver *database.DatabaseDriver) error {
    query := `UPDATE DraftInvites SET accepted = $1 WHERE draftId = $2 AND invitedPlayer = $3`
    stmt, err := dbDriver.Connection.Prepare(query)

    if err != nil {
        return err
    }

    _, err = stmt.Exec(true, draftId, playerId)

    return err
}

func DeclineInvite(draftId int, playerId int, dbDriver *database.DatabaseDriver) error {
    query := `Delete FROM DraftInvites WHERE draftId = $1 AND invitedPlayer = $2`
    stmt, err := dbDriver.Connection.Prepare(query)

    if err != nil {
        return err
    }

    _, err = stmt.Exec(draftId, playerId)

    return err
}
