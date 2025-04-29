package draft

import (
	"database/sql"
	"errors"
	"server/model"
	"server/picking"
	"server/tbaHandler"
	"sync"
)

type DraftManager struct {
    drafts map[int]*Draft
    database *sql.DB
    tbaHandler *tbaHandler.TbaHandler
}

func (dm *DraftManager) GetDraft(draftId int, reload bool) (*Draft, error) {
    draft, ok := dm.drafts[draftId]
    if ok && !reload {
        return draft, nil
    } else if reload {
        draftModel, err := model.GetDraft(dm.database, draftId)
        draft.model = &draftModel
        return draft, err
    } else {
        //Load draft model
        draftModel, err := model.GetDraft(dm.database, draftId)

        draft := Draft {
            draftId: draftId,
            pickManager: picking.NewPickManager(draftId, dm.database, dm.tbaHandler),
            model: &draftModel,
        }
        dm.drafts[draftId] = &draft
        return &draft, err
    }
}

type Draft struct {
    draftId int
    model *model.DraftModel
    pickManager *picking.PickManager
    stateLock sync.Mutex
}

func (d *Draft) ExecuteDraftStateTransition(requestedState model.DraftState, database *sql.DB) error {
    d.stateLock.Lock()
    defer d.stateLock.Unlock()

    if d.model.Status == requestedState {
        return errors.New("Draft is already in requested state")
    }

    switch requestedState {
    case model.FILLING:
        //Drafts should start in this status so we should not allow them to
        //transistion into this status
        return nil
    case model.WAITING_TO_START:
        //Drafts enter this status when the number of players who have accepted the draft
        //is equal to the number of players needed for the draft
        return nil
    case model.PICKING:
        //The draft enters this status when the owner indicates it is time to start picking
        return nil
    case model.TEAMS_PLAYING:
        //The draft enters this status when all players have picked their teams
        return nil
    case model.COMPLETE:
        //The draft enters this status after scoring has been completed
        return nil
    default:
        return errors.New("Invalid requested draft state")
    }
}
