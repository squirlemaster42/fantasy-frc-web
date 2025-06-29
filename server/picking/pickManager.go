package picking

import (
	"database/sql"
	"errors"
	"log/slog"
	"server/model"
	"server/tbaHandler"
	"server/utils"
	"sync"
	"time"
)

type PickManager struct {
    draftId int
    lock sync.Mutex
    listeners []*PickListener
    database *sql.DB
    tbaHandler *tbaHandler.TbaHandler
}

type PickEvent struct {
    Success bool
    Err error
    Pick model.Pick
    DraftId int
}

type PickListener interface {
    ReceivePickEvent(pickEvent PickEvent)
}

func NewPickManager(draftId int, database *sql.DB, tbaHandler *tbaHandler.TbaHandler) *PickManager {
    return &PickManager{
        draftId: draftId,
        database: database,
        tbaHandler: tbaHandler,
    }
}

func (p *PickManager) SkipCurrentPick() {
    curPick := model.GetCurrentPick(p.database, p.draftId)
    nextPickPlayer := model.NextPick(p.database, p.draftId)
    model.SkipPick(p.database, curPick.Id)
    model.MakePickAvailable(p.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now()))

    for _, listener := range p.listeners {
        //TODO How do we get the error in here?
        (*listener).ReceivePickEvent(PickEvent {
            Pick: model.Pick{},
            Success: true,
            Err: nil,
            DraftId: p.draftId,
        })
    }
}

//Return error if pick is not able to be made
func (p *PickManager) MakePick(pick model.Pick) (bool, error) {
    p.lock.Lock()
    defer p.lock.Unlock()

    pickingComplete := false

    var err error
    valid := false
    if !pick.Pick.Valid {
        err = errors.New("no team entered")
    }

    if err == nil {
        valid, err = model.ValidPick(p.database, p.tbaHandler, pick.Pick.String, p.draftId)
    }

    if err == nil {
        //If we have not found any errors indicating that the pick is invalid, make the pick
        model.MakePick(p.database, pick)
        nextPickPlayer := model.NextPick(p.database, p.draftId)

        //Make the next pick available if we havn't aleady made all picks
        picks, err := model.GetPicks(p.database, p.draftId)

        if err != nil {
            slog.Warn("Failed to get picks", "Draft Id", p.draftId, "Error", err)
            return false, err
        }

        slog.Info("Checking if we should make another pick available", "Num picks", len(picks))
        if len(picks) < 64 {
            slog.Info("Making next pick available", "Draft Id", p.draftId)
            model.MakePickAvailable(p.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now()))
        } else {
            //Set draft to the teams playing state
            //This isnt entirely correct becuase it doesnt account for skips
            //But I dont care about that for this year
            pickingComplete = true
        }
    }

    for _, listener := range p.listeners {
        (*listener).ReceivePickEvent(PickEvent{
            Pick: pick,
            Success: valid,
            Err: err,
            DraftId: p.draftId,
        })
    }

    return pickingComplete, err
}

func (p *PickManager) AddListener(listener PickListener) {
    slog.Info("Added pick listener", "Listener", listener)
    p.listeners = append(p.listeners, &listener)
}
