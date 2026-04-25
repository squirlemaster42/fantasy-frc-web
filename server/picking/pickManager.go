package picking

import (
	"database/sql"
	"errors"
	"server/discord"
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
	discordBus *discord.DiscordWebhookBus
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

func NewPickManager(draftId int, database *sql.DB, tbaHandler *tbaHandler.TbaHandler, discordBus *discord.DiscordWebhookBus) *PickManager {
	return &PickManager{
		draftId:    draftId,
		database:   database,
		tbaHandler: tbaHandler,
		discordBus: discordBus,
	}
}

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
	curPick, err := model.GetCurrentPick(p.database, p.draftId)
	if err != nil {
		p.lock.Unlock()
		return err
	}
	nextPickPlayer := model.NextPick(p.database, p.draftId)
	model.SkipPick(p.database, curPick.Id)
	model.MakePickAvailable(p.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now(), utils.PICK_TIME))

	event := PickEvent{
		Pick:    model.Pick{},
		Success: true,
		Err:     nil,
		DraftId: p.draftId,
	}
	p.lock.Unlock()

	go func() {
		for _, listener := range p.listeners {
			(*listener).ReceivePickEvent(event)
		}
	}()
	return nil
}

func (p *PickManager) MakePick(pick model.Pick) (bool, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	pickingComplete := false

	var err error
	if !pick.Pick.Valid {
		err = errors.New("no team entered")
		return false, err
	}

	// Check that we are still trying to make the current pick
	currentPick, err := model.GetCurrentPick(p.database, p.draftId)
	if currentPick.Id != pick.Id {
		log.WarnNoContext("Pick attempt made against pick that is not the current pick", "Current Pick", currentPick.Id, "Attempted Pick", pick.Id)
		return false, errors.New("attempting to make pick that is not the current pick")
	}

	if err == nil {
		_, err = model.ValidPick(p.database, p.tbaHandler, pick.Pick.String, p.draftId)
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
			model.MakePickAvailable(p.database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now(), utils.PICK_TIME))

			currPickDiscordId, err := model.GetPlayerDiscordId(p.database, currentPick.Player)
			if err != nil {
				log.WarnNoContext("Could not get current pick draft player id", "Draft Player Id", pick.Player, "Error", err)
				err = nil
			}

			currPickUser, err := model.GetDraftPlayerUser(p.database, currentPick.Player)
			if err != nil {
				log.WarnNoContext("Could not get current pick draft player name", "Draft Player Id", pick.Player, "Error", err)
				err = nil
			}
			currPickName := currPickUser.Username

			nextPickDiscordId, err := model.GetPlayerDiscordId(p.database, nextPickPlayer.Id)
			if err != nil {
				log.WarnNoContext("Could not get next pick draft player id", "Draft Player Id", nextPickPlayer.Id, "Error", err)
				err = nil
			}

			nextPickUser, err := model.GetDraftPlayerUser(p.database, nextPickPlayer.Id)
			if err != nil {
				log.WarnNoContext("Could not get next pick draft player name", "Draft Player Id", nextPickPlayer.Id, "Error", err)
				err = nil
			}
			nextPickName := nextPickUser.Username

			draftWebhook, err := model.GetDraftWebhook(p.database, p.draftId)
			if err != nil {
				log.WarnNoContext("Could not get draft webhook", "Draft Id", p.draftId, "Error", err)
				err = nil
			}

			event := discord.NextPickDiscordEvent{
				PreviousPickedTeam:    pick.Pick.String,
				PreviousPickName:      currPickName,
				PreviousPickDiscordId: currPickDiscordId,
				NextPickName:          nextPickName,
				NextPickDiscordId:     nextPickDiscordId,
				Webhook:               draftWebhook,
			}
			go func() {
				err = p.discordBus.PostPickNotification(event)
				if err != nil {
					log.WarnNoContext("Failed to post discord webhook", "Error", err)
				}
			}()
		} else {
			log.InfoNoContext("Draft Complete", "Draft Id", p.draftId)
			// Set draft to the teams playing state
			// This isnt entirely correct becuase it doesnt account for skips
			// But I dont care about that for this year
			pickingComplete = true
		}
	}

	return pickingComplete, err
}

func (p *PickManager) AddListener(listener PickListener) {
	log.InfoNoContext("Added pick listener", "Listener", listener)
	p.listeners = append(p.listeners, &listener)
}

func (p *PickManager) NotifyListeners(pickEvent PickEvent) {
	log.DebugNoContext("Started notifying pick listeners", "Draft Id", pickEvent.DraftId, "Pick", pickEvent.Pick.Pick.String, "Num Listeners", len(p.listeners))
	for _, listener := range p.listeners {
		log.DebugNoContext("Notifying pick listener", "Draft Id", pickEvent.DraftId, "Pick", pickEvent.Pick.Pick.String, "Num Listeners", len(p.listeners))
		(*listener).ReceivePickEvent(pickEvent)
		log.DebugNoContext("Notified pick listener", "Draft Id", pickEvent.DraftId, "Pick", pickEvent.Pick.Pick.String, "Num Listeners", len(p.listeners))
	}
	log.DebugNoContext("Finished notifying pick listeners", "Draft Id", pickEvent.DraftId, "Pick", pickEvent.Pick.Pick.String)
}
