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
    handler    tbaHandler.TBAInterface
    draftStore model.DraftStore
    draftId    int
    eventSet   map[string]struct{}
}

func NewPickValidator(handler tbaHandler.TBAInterface, draftStore model.DraftStore, draftId int) PickValidator {
    validEvents := utils.Events()
    eventSet := make(map[string]struct{}, len(validEvents))
    for _, event := range validEvents {
        eventSet[event] = struct{}{}
    }

    return PickValidator{
        handler:    handler,
        draftStore: draftStore,
        draftId:    draftId,
        eventSet:   eventSet,
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

	events, err := p.handler.MakeEventListReq(ctx, pick.Pick.String)
	if err != nil {
		return err
	}

	log.Debug(ctx, "Checking if team is in a valid event", "teamEvents", events)
	for _, event := range events {
		if _, ok := p.eventSet[event]; ok {
			log.Debug(ctx, "Checked if team is a valid pick", "team", pick.Pick.String, "picked", picked, "validEvent", true)
			return nil
		}
	}

	log.Debug(ctx, "Checked if team is a valid pick", "team", pick.Pick.String, "picked", picked, "validEvent", false)
	return errors.New("team not at event")
}
