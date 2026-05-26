package draft

import (
	"context"
	"errors"
	"server/log"
	"server/model"
	"server/tbaHandler"
	"server/utils"
)

type PickValidator struct {
    handler *tbaHandler.TbaHandler
    draftStore model.DraftStore
	draftId int
}

func NewPickValidator(handler *tbaHandler.TbaHandler, draftStore model.DraftStore, draftId int) PickValidator {
    return PickValidator{
        handler: handler,
        draftStore: draftStore,
		draftId: draftId,
    }
}

func (p *PickValidator) ValidatePick(ctx context.Context, pick model.Pick) error {
	if !pick.Pick.Valid {
		return errors.New("no team entered")
	}

	if pick.Pick.String == "" {
		return errors.New("no team entered")
	}

	picked, err := p.draftStore.HasBeenPicked(ctx, p.draftId, pick.Pick.String)
	if err != nil {
		return err
	}

	if picked {
		return errors.New("team already picked")
	}

	events := p.handler.MakeEventListReq(ctx, pick.Pick.String)
	draftEvents := utils.Events()

	validEvent := false
	//Looping here should always be faster because of the small lists
	log.Info(ctx, "Checking is team is in a valid event", "Team Events", events, "Draft Events", draftEvents)
	for _, event := range events {
		for _, draftEvent := range draftEvents {
			if event == draftEvent {
				validEvent = true
				break
			}
		}

		if validEvent {
			break
		}
	}

	log.Info(ctx, "Checked if team is a valid pick", "Team", pick.Pick.String, "Picked", picked, "Valid Event", validEvent)
	if !validEvent {
		return errors.New("team not at event")
	}
	return nil
}
