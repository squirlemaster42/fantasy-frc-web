package handler

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"server/background"
	"server/cache"
	"server/draft"
	"server/log"
	"server/model"
	"server/tbaHandler"
	"server/utils"
)

// AdminService encapsulates the business logic for admin console operations.
// It uses typed parameters instead of the old CLI-style argument strings.
type AdminService struct {
	TBAHandler     tbaHandler.TBAInterface
	DraftStore     model.DraftStore
	TeamStore      model.TeamStore
	DraftActorMap  *draft.DraftActorMap
	DraftDaemon    *background.DraftDaemon
	EventTeamCache cache.EventTeamCache
}

// Ping returns a simple liveness response.
func (s *AdminService) Ping(_ context.Context) (string, error) {
	return "Pong", nil
}

// PopulateTeams fetches teams from TBA for all configured events and creates
// any team records that do not already exist.
func (s *AdminService) PopulateTeams(ctx context.Context) (string, error) {
	log.Info(ctx, "Populating Teams")
	count := 0

	for _, event := range utils.Events() {
		log.Debug(ctx, "Creating teams for event", "event", event)
		teams, err := s.TBAHandler.MakeTeamsAtEventRequest(ctx, event)
		if err != nil {
			log.Error(ctx, "Failed to get teams for event", "event", event, "error", err)
			continue
		}
		for _, t := range teams {
			log.Debug(ctx, "Checking if team is needed", "team", t.Key, "event", event)
			team, err := s.TeamStore.GetTeam(ctx, t.Key)
			if err != nil {
				log.Error(ctx, "Failed to get team", "team", t.Key, "error", err)
				continue
			}
			if team == nil {
				log.Debug(ctx, "Creating team", "team", t.Key, "event", event)
				if err := s.TeamStore.CreateTeam(ctx, t.Key, ""); err != nil {
					log.Error(ctx, "Failed to create team", "team", t.Key, "error", err)
					continue
				}
				count++
			}
		}
	}

	log.Info(ctx, "Finished populating teams", "count", count)
	return fmt.Sprintf("Successfully populated %d teams", count), nil
}

// ListDrafts returns a formatted list of drafts, optionally filtered by name.
func (s *AdminService) ListDrafts(ctx context.Context, search string) (string, error) {
	drafts, err := s.DraftStore.GetDraftsByName(ctx, search)
	if err != nil {
		return "", fmt.Errorf("failed to list drafts: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("Id    |  Name\n")
	sb.WriteString("-------------\n")
	for _, d := range drafts {
		fmt.Fprintf(&sb, "%4d  | %s\n", d.Id, d.DisplayName)
	}
	return sb.String(), nil
}

// StartDraft transitions a draft from filling to picking.
func (s *AdminService) StartDraft(ctx context.Context, draftId int) (string, error) {
	draftActor, err := s.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Error(ctx, "Failed to get draft actor", "draftId", draftId, "error", err)
		return "", fmt.Errorf("draft ID does not match a valid draft")
	}

	draftState := draftActor.GetDraftState()
	if !model.CanStartDraft(draftState) {
		return "", fmt.Errorf("draft must have between %d and %d accepted players to start", model.MinDraftPlayers, model.MaxDraftPlayers)
	}

	if err := s.DraftStore.CancelOutstandingInvites(ctx, draftId); err != nil {
		log.Error(ctx, "Failed to cancel outstanding invites", "draftId", draftId, "error", err)
		return "", fmt.Errorf("failed to cancel outstanding invites")
	}

	if err := draft.ExecuteDraftStateTransition(ctx, draftActor, model.PICKING); err != nil {
		log.Error(ctx, "Failed to execute draft state transition", "draftId", draftId, "error", err)
		return "", err
	}

	if s.DraftDaemon != nil {
		if err := s.DraftDaemon.AddDraft(ctx, draftId); err != nil {
			log.Warn(ctx, "Failed to add draft to daemon", "draftId", draftId, "error", err)
		}
	}

	return "Draft Started", nil
}

// ViewWebhookKey returns the contents of the configured webhook key file.
func (s *AdminService) ViewWebhookKey(_ context.Context) (string, error) {
	file, err := os.Open(utils.GetWebhookFilePath())
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() { _ = file.Close() }()

	body, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(body), nil
}

// SkipPick skips the current pick for the given draft.
func (s *AdminService) SkipPick(ctx context.Context, draftId int) (string, error) {
	log.Info(ctx, "Skipping pick", "draftId", draftId)

	draftActor, err := s.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Error(ctx, "Failed to get draft actor", "draftId", draftId, "error", err)
		return "", fmt.Errorf("draft ID does not match a valid draft")
	}

	skipped := draft.SkipCurrentPick(ctx, draftActor, draftId, draftActor.GetDraftState().CurrentPick.Id)
	if !skipped {
		return "", fmt.Errorf("did not get confirmation of skip; verify draft state")
	}
	return "Player was skipped", nil
}

// ModifyPickTime adjusts the current pick expiration time by the given duration.
func (s *AdminService) ModifyPickTime(ctx context.Context, draftId int, delta time.Duration) (string, error) {
	log.Info(ctx, "Modifying pick time", "draftId", draftId, "delta", delta)

	draftActor, err := s.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Error(ctx, "Failed to get draft actor", "draftId", draftId, "error", err)
		return "", fmt.Errorf("draft ID does not match a valid draft")
	}

	if err := draft.ModifyCurrentPickExpirationTime(ctx, draftActor, delta); err != nil {
		log.Warn(ctx, "Update draft pick expiration time failed", "draftId", draftId, "delta", delta, "error", err)
		return "", err
	}

	return "Successfully updated pick expiration time", nil
}

// AdminPick makes a pick on behalf of the current player in a draft.
func (s *AdminService) AdminPick(ctx context.Context, draftId int, teamNumber string) (string, error) {
	log.Info(ctx, "Making admin pick", "draftId", draftId, "team", teamNumber)

	tbaId := teamPrefix + teamNumber

	draftActor, err := s.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Error(ctx, "Failed to get draft actor", "draftId", draftId, "error", err)
		return "", err
	}

	draftState := draftActor.GetDraftState()
	if draftState.CurrentPick.Id == 0 {
		return "", fmt.Errorf("no current pick found for this draft")
	}

	pickStruct := model.Pick{
		Id:     draftState.CurrentPick.Id,
		Player: draftState.CurrentPick.Player,
		Pick: sql.NullString{
			Valid:  true,
			String: tbaId,
		},
		PickTime: sql.NullTime{
			Valid: true,
			Time:  time.Now().UTC(),
		},
	}

	if err := draft.MakePick(ctx, draftActor, pickStruct); err != nil {
		log.Warn(ctx, "Could not make pick", "currentPickId", draftState.CurrentPick.Id, "pick", tbaId, "error", err)
		return "", err
	}

	return fmt.Sprintf("Successfully picked team %s", teamNumber), nil
}

// RenameDraft updates the display name of a draft.
func (s *AdminService) RenameDraft(ctx context.Context, draftId int, newName string) (string, error) {
	log.Info(ctx, "Renaming draft", "draftId", draftId, "newName", newName)

	draftActor, err := s.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		return "", fmt.Errorf("draft ID does not match a valid draft")
	}

	draftModel := draft.GetDraft(draftActor)
	oldName := draftModel.DisplayName
	draftModel.DisplayName = newName

	if err := draft.UpdateDraft(ctx, draftActor, draftModel); err != nil {
		log.Error(ctx, "Failed to update draft name", "draftId", draftId, "error", err)
		return "", fmt.Errorf("failed to update draft name")
	}

	return fmt.Sprintf("Successfully renamed draft from '%s' to '%s'", oldName, newName), nil
}

// UndoPick reverts the most recent pick in a draft.
func (s *AdminService) UndoPick(ctx context.Context, draftId int) (string, error) {
	log.Info(ctx, "Undoing pick", "draftId", draftId)

	draftActor, err := s.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		return "", fmt.Errorf("draft ID does not match a valid draft")
	}

	if err := draft.UndoLastPick(ctx, draftActor); err != nil {
		return "", err
	}

	return "Successfully undid pick", nil
}

// InvalidatePickCache refreshes the TBA event-team validation cache.
func (s *AdminService) InvalidatePickCache(ctx context.Context) (string, error) {
	if s.EventTeamCache == nil {
		return "", fmt.Errorf("pick validation cache is not configured")
	}

	log.Info(ctx, "Invalidating pick validation cache")
	if err := s.EventTeamCache.InvalidateAndRefresh(ctx); err != nil {
		log.Error(ctx, "Failed to invalidate pick validation cache", "error", err)
		return "", fmt.Errorf("failed to invalidate pick validation cache: %w", err)
	}

	return fmt.Sprintf("Pick validation cache invalidated and refreshed (last refresh: %s)", s.EventTeamCache.LastRefresh().Format(time.RFC3339)), nil
}
