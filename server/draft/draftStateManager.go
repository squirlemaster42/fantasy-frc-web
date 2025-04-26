package draft

import (
	"database/sql"
	"errors"
)

type DraftState string

const (
    FILLING DraftState = "Filling"
    WAITING_TO_START = "Waiting to Start"
    PICKING = "Picking"
    TEAMS_PLAYING = "Teams Playing"
    COMPLETE = "Complete"
)

func ExecuteDraftStateTransition(draftId int, requestedState DraftState, database *sql.DB) error {
    switch requestedState {
    case FILLING:
        return errors.New("Invalid requested draft state")
    case WAITING_TO_START:
        return errors.New("Invalid requested draft state")
    case PICKING:
        //TODO Add draft to pick manager
        return errors.New("Invalid requested draft state")
    case TEAMS_PLAYING:
        return errors.New("Invalid requested draft state")
    case COMPLETE:
        return errors.New("Invalid requested draft state")
    default:
        return errors.New("Invalid requested draft state")
    }
}
