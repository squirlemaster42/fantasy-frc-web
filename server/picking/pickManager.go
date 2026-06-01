package picking

import (
	"context"
	"errors"
	"server/discord"
	"server/log"
	"server/metrics"
	"server/model"
	"server/tbaHandler"
	"server/utils"
	"sync"
	"time"
)

type PickManager struct {
	draftId      int
	pickLock     sync.Mutex
	listenerLock sync.RWMutex
	listeners    []PickListener
	draftStore   model.DraftStore
	teamStore    model.TeamStore
	discordStore model.DiscordStore
	tbaHandler   *tbaHandler.TbaHandler
	discordBus   *discord.DiscordWebhookBus
}

type PickEvent struct {
	Success bool
	Err     error
	Pick    model.Pick
	DraftId int
}

type PickListener interface {
	ReceivePickEvent(pickEvent PickEvent) error
}

func NewPickManager(draftId int, draftStore model.DraftStore, teamStore model.TeamStore, discordStore model.DiscordStore, tbaHandler *tbaHandler.TbaHandler, discordBus *discord.DiscordWebhookBus) *PickManager {
	return &PickManager{
		draftId:      draftId,
		draftStore:   draftStore,
		teamStore:    teamStore,
		discordStore: discordStore,
		tbaHandler:   tbaHandler,
		discordBus:   discordBus,
	}
}

func (p *PickManager) GetCurrentPick(draftId int) (model.Pick, error) {
	p.pickLock.Lock()
	defer p.pickLock.Unlock()
	return p.draftStore.GetCurrentPick(context.TODO(), draftId)
}

func (p *PickManager) UndoLastPick() error {
	p.pickLock.Lock()
	defer p.pickLock.Unlock()

	// Get the current pick
	currentPick, err := p.draftStore.GetCurrentPick(context.TODO(), p.draftId)
	if currentPick.Id == 0 || err != nil {
		return errors.New("no current pick found for this draft")
	}

	// Get the previous pick
	previousPick, err := p.draftStore.GetPreviousPick(context.TODO(), p.draftId, currentPick.Id)
	if err != nil {
		return errors.New("cannot undo pick: this is the first pick of the draft")
	}

	// Delete the current pick
	err = p.draftStore.DeletePick(context.TODO(), currentPick.Id)
	if err != nil {
		log.Error(context.TODO(), "Failed to delete current pick", "Pick Id", currentPick.Id, "Error", err)
		return errors.New("failed to delete current pick")
	}

	// Set the expiration time to 3 hours from now
	newExpirationTime := time.Now().Add(3 * time.Hour)

	// Reset the previous pick (null out pick and pickTime, and set new expiration)
	err = p.draftStore.ResetPick(context.TODO(), previousPick.Id, newExpirationTime)
	if err != nil {
		log.Error(context.TODO(), "Failed to reset previous pick", "Pick Id", previousPick.Id, "Error", err)
		return errors.New("failed to reset previous pick")
	}
	return nil
}

func (p *PickManager) SkipCurrentPick() error {
	p.pickLock.Lock()
	curPick, err := p.draftStore.GetCurrentPick(context.TODO(), p.draftId)
	if err != nil {
		p.pickLock.Unlock()
		return err
	}
	nextPickPlayer, err := p.draftStore.NextPick(context.TODO(), p.draftId)
	if err != nil {
		return err
	}
	err = p.draftStore.SkipPick(context.TODO(), curPick.Id)
	if err != nil {
		log.Warn(context.Background(), "Failed to skip current pick", "Current pick", curPick.Id, "Error", err)
		return err
	}
	_, err = p.draftStore.MakePickAvailable(context.TODO(), nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(context.TODO(), time.Now(), utils.PICK_TIME))
	if err != nil {
		log.Warn(context.Background(), "Failed to make pick available when skipping current pick", "Current pick", curPick.Id, "Error", err)
		return err
	}

	event := PickEvent{
		Pick:    model.Pick{},
		Success: true,
		Err:     nil,
		DraftId: p.draftId,
	}
	p.pickLock.Unlock()

	go p.NotifyListeners(event)
	return nil
}

func (p *PickManager) MakePick(ctx context.Context, pick model.Pick) (bool, error) {
	p.pickLock.Lock()
	defer p.pickLock.Unlock()

	pickingComplete := false

	var err error
	if !pick.Pick.Valid {
		err = errors.New("no team entered")
		return false, err
	}

	// Check that we are still trying to make the current pick
	currentPick, err := p.draftStore.GetCurrentPick(ctx, p.draftId)
	if currentPick.Id != pick.Id {
		log.Warn(ctx, "Pick attempt made against pick that is not the current pick", "Current Pick", currentPick.Id, "Attempted Pick", pick.Id)
		return false, errors.New("attempting to make pick that is not the current pick")
	}

	if err == nil {
		_, err = model.ValidPick(ctx, p.draftStore, p.teamStore, p.tbaHandler, pick.Pick.String, p.draftId)
	}

	if err == nil {
		//If we have not found any errors indicating that the pick is invalid, make the pick
		err := p.draftStore.MakePick(ctx, pick)
		if err != nil {
			return false, err
		}
		nextPickPlayer, err := p.draftStore.NextPick(ctx, p.draftId)
		if err != nil {
			log.Warn(ctx, "failed to get next pick", "Pick Id", pick.Id, "Errors", err)
			return false, err
		}

		//Make the next pick available if we havn't aleady made all picks
		picks, err := p.draftStore.GetPicks(ctx, p.draftId)

		if err != nil {
			log.Warn(ctx, "Failed to get picks", "Draft Id", p.draftId, "Error", err)
			return false, err
		}

		log.Info(ctx, "Checking if we should make another pick available", "Num picks", len(picks))
		if len(picks) < 64 {
			log.Info(ctx, "Making next pick available", "Draft Id", p.draftId)
			expirationTime := utils.GetPickExpirationTime(ctx, time.Now(), utils.PICK_TIME)
			_, err = p.draftStore.MakePickAvailable(ctx, nextPickPlayer.Id, time.Now(), expirationTime)
			if err != nil {
				log.Warn(ctx, "Failed to make pick available", "Draft Player Id", pick.Player, "Error", err)
				return false, err
			}

			currPickDiscordId, err := p.discordStore.GetPlayerDiscordId(ctx, currentPick.Player)
			if err != nil {
				log.Warn(ctx, "Could not get current pick draft player id", "Draft Player Id", pick.Player, "Error", err)
				return false, err
			}

			currPickUser, err := p.draftStore.GetDraftPlayerUser(ctx, currentPick.Player)
			if err != nil {
				log.Warn(ctx, "Could not get current pick draft player name", "Draft Player Id", pick.Player, "Error", err)
				return false, err
			}
			currPickName := currPickUser.Username

			nextPickDiscordId, err := p.discordStore.GetPlayerDiscordId(ctx, nextPickPlayer.Id)
			if err != nil {
				log.Warn(ctx, "Could not get next pick draft player id", "Draft Player Id", nextPickPlayer.Id, "Error", err)
				return false, err
			}

			nextPickUser, err := p.draftStore.GetDraftPlayerUser(ctx, nextPickPlayer.Id)
			if err != nil {
				log.Warn(ctx, "Could not get next pick draft player name", "Draft Player Id", nextPickPlayer.Id, "Error", err)
				return false, err
			}
			nextPickName := nextPickUser.Username

		draftWebhook, err := p.discordStore.GetDraftWebhook(ctx, p.draftId)
		if err != nil || draftWebhook == "" {
			log.Warn(ctx, "No discord webhook configured, skipping discord notification", "Draft Id", p.draftId, "Error", err)
		} else {
			event := discord.NextPickDiscordEvent{
				PreviousPickedTeam:    pick.Pick.String,
				PreviousPickName:      currPickName,
				PreviousPickDiscordId: currPickDiscordId,
				NextPickName:          nextPickName,
				NextPickDiscordId:     nextPickDiscordId,
				Webhook:               draftWebhook,
				ExpirationTime:        expirationTime,
			}
			go func() {
				err = p.discordBus.PostPickNotification(event)
				if err != nil {
					log.Warn(ctx, "Failed to post discord webhook", "Error", err)
				}
			}()
		}
		} else {
			log.Info(ctx, "Draft Complete", "Draft Id", p.draftId)
			// Set draft to the teams playing state
			// This isnt entirely correct becuase it doesnt account for skips
			// But I dont care about that for this year
			pickingComplete = true
		}
	}

	return pickingComplete, err
}

func (p *PickManager) AddListener(listener PickListener) {
	log.Info(context.TODO(), "Added pick listener", "Listener", listener)
	p.listenerLock.Lock()
	p.listeners = append(p.listeners, listener)
	p.listenerLock.Unlock()
	metrics.IncrementWebSocketListener()
}

func (p *PickManager) RemoveListener(listener PickListener) {
	p.listenerLock.Lock()
	defer p.listenerLock.Unlock()

	for i, l := range p.listeners {
		if l == listener {
			p.listeners[i] = p.listeners[len(p.listeners)-1]
			p.listeners = p.listeners[:len(p.listeners)-1]
			log.Info(context.TODO(), "Removed pick listener", "Listener", listener)
			metrics.DecrementWebSocketListener()
			return
		}
	}
	log.Warn(context.TODO(), "Failed to remove pick listener, not found", "Listener", listener)
}

func (p *PickManager) NotifyListeners(pickEvent PickEvent) {
	log.DebugNoContext("Started notifying pick listeners", "Draft Id", pickEvent.DraftId, "Pick", pickEvent.Pick.Pick.String, "Num Listeners", len(p.listeners))

	p.listenerLock.RLock()
	listenersCopy := make([]PickListener, len(p.listeners))
	copy(listenersCopy, p.listeners)
	p.listenerLock.RUnlock()

	var wg sync.WaitGroup
	for _, listener := range listenersCopy {
		wg.Add(1)
		go func(l PickListener) {
			defer wg.Done()
			log.DebugNoContext("Notifying pick listener", "Draft Id", pickEvent.DraftId, "Pick", pickEvent.Pick.Pick.String)
			if err := l.ReceivePickEvent(pickEvent); err != nil {
				log.Warn(context.TODO(), "Removing dead listener", "Listener", l, "Error", err)
				p.RemoveListener(l)
			}
		}(listener)
	}
	wg.Wait()
	log.DebugNoContext("Finished notifying pick listeners", "Draft Id", pickEvent.DraftId)
}
