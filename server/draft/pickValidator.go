package draft

import (
	"context"
	"errors"
	"server/cache"
	"server/log"
	"server/model"
)

type PickValidator struct {
	cache      cache.EventTeamCache
	draftStore model.DraftStore
	draftId    int
}

func NewPickValidator(cache cache.EventTeamCache, draftStore model.DraftStore, draftId int) PickValidator {
	return PickValidator{
		cache:      cache,
		draftStore: draftStore,
		draftId:    draftId,
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

	if p.cache == nil {
		log.Error(ctx, "Pick validation cache is not configured")
		return errors.New("pick validation cache is not available")
	}

	valid, err := p.cache.IsTeamAtEvent(ctx, pick.Pick.String)
	if err != nil {
		log.Warn(ctx, "Failed to validate team against event cache", "team", pick.Pick.String, "error", err)
		return errors.New("unable to validate team eligibility")
	}

	log.Debug(ctx, "Checked if team is a valid pick", "team", pick.Pick.String, "picked", picked, "validEvent", valid)
	if !valid {
		return errors.New("team not at event")
	}

	return nil
}
