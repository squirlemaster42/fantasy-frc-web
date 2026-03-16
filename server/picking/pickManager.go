package picking

import (
	"database/sql"
	"errors"
	"server/log"
	"server/model"
	"server/tbaHandler"
	"server/utils"
	"sync"
	"time"
)

type PickManager struct {
	draftId    int
	lock       sync.Mutex
	listeners  []*PickListener
	database   *sql.DB
	tbaHandler *tbaHandler.TbaHandler
}

type PickEvent struct {
	Success bool
	Err     error
	Pick    model.Pick
	DraftId int
}

type PickListener interface {
	ReceivePickEvent(pickEvent PickEvent)
}

func NewPickManager(draftId int, database *sql.DB, tbaHandler *tbaHandler.TbaHandler) *PickManager {
	return &PickManager{
		draftId:    draftId,
		database:   database,
		tbaHandler: tbaHandler,
	}
}

// TODO Make sure that all GetCurrentPick calls are going through this
func (p *PickManager) GetCurrentPick(draftId int) (model.Pick, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	return model.GetCurrentPick(p.database, draftId)
}

func (p *PickManager) UndoLastPick() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Get the current pick
	currentPick, err := model.GetCurrentPick(p.database, p.draftId)
	if currentPick.Id == 0 || err != nil {
		return errors.New("no current pick found for this draft")
	}

	// Get the previous pick
	previousPick, err := model.GetPreviousPick(p.database, p.draftId, currentPick.Id)
	if err != nil {
		return errors.New("cannot undo pick: this is the first pick of the draft")
	}

	// Delete the current pick
	err = model.DeletePick(p.database, currentPick.Id)
	if err != nil {
		log.ErrorNoContext("Failed to delete current pick", "Pick Id", currentPick.Id, "Error", err)
		return errors.New("failed to delete current pick")
	}

	// Set the expiration time to 3 hours from now
	newExpirationTime := time.Now().Add(3 * time.Hour)

	// Reset the previous pick (null out pick and pickTime, and set new expiration)
	err = model.ResetPick(p.database, previousPick.Id, newExpirationTime)
	if err != nil {
		log.ErrorNoContext("Failed to reset previous pick", "Pick Id", previousPick.Id, "Error", err)
		return errors.New("failed to reset previous pick")
	}
	return nil
}

func (p *PickManager) SkipCurrentPick() error {
	p.lock.Lock()
	defer p.lock.Unlock()
	curPick, err := model.GetCurrentPick(p.database, p.draftId)
	if err != nil {
		return err
	}
	nextPickPlayer := model.NextPick(p.database, p.draftId)
	model.SkipPick(p.database, curPick.Id)
	model.MakePickAvailable(p.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now()))

	for _, listener := range p.listeners {
		(*listener).ReceivePickEvent(PickEvent{
			Pick:    model.Pick{},
			Success: true,
			Err:     nil,
			DraftId: p.draftId,
		})
	}
	return nil
}

// TODO Deal with error on all callers
// Return error if pick is not able to be made
func (p *PickManager) MakePick(pick model.Pick) (bool, error) {
	// TODO There is a bug on the last pick when watching the page with
	// websockets that causes a new row to show up after pick 64 is made

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
		err := model.MakePick(p.database, pick)
		if err != nil {
			return false, err
		}
		nextPickPlayer := model.NextPick(p.database, p.draftId)

		//Make the next pick available if we havn't aleady made all picks
		picks, err := model.GetPicks(p.database, p.draftId)

		if err != nil {
			log.WarnNoContext("Failed to get picks", "Draft Id", p.draftId, "Error", err)
			return false, err
		}

		log.InfoNoContext("Checking if we should make another pick available", "Num picks", len(picks))
		if len(picks) < 64 {
			log.InfoNoContext("Making next pick available", "Draft Id", p.draftId)
			model.MakePickAvailable(p.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now()))
		} else {
			// Set draft to the teams playing state
			// This isnt entirely correct becuase it doesnt account for skips
			// But I dont care about that for this year
			pickingComplete = true
		}
	}

	for _, listener := range p.listeners {
		(*listener).ReceivePickEvent(PickEvent{
			Pick:    pick,
			Success: valid,
			Err:     err,
			DraftId: p.draftId,
		})
	}

	return pickingComplete, err
}

func (p *PickManager) AddListener(listener PickListener) {
	log.InfoNoContext("Added pick listener", "Listener", listener)
	p.listeners = append(p.listeners, &listener)
}
