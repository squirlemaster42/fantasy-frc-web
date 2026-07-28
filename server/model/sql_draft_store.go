package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type SQLDraftStore struct {
	database *sql.DB
}

func NewSQLDraftStore(database *sql.DB) *SQLDraftStore {
	return &SQLDraftStore{database: database}
}

func (s *SQLDraftStore) GetDraft(ctx context.Context, draftId int) (DraftModel, error) {
	return getDraft(ctx, s.database, draftId)
}

func (s *SQLDraftStore) GetDraftsByName(ctx context.Context, searchString string) ([]DraftModel, error) {
	return getDraftsByName(ctx, s.database, searchString)
}

func (s *SQLDraftStore) GetDraftScore(ctx context.Context, draftId int) ([]DraftPlayer, error) {
	return getDraftScore(ctx, s.database, draftId)
}

func (s *SQLDraftStore) GetDraftPickRows(ctx context.Context, teamKeys []string) ([]DraftPickRow, error) {
	return getDraftPickRows(ctx, s.database, teamKeys)
}

func (s *SQLDraftStore) GetDraftsForUser(ctx context.Context, userUuid uuid.UUID) ([]DraftModel, error) {
	return getDraftsForUser(ctx, s.database, userUuid)
}

func (s *SQLDraftStore) CreateDraft(ctx context.Context, draft *DraftModel) (int, error) {
	return createDraft(ctx, s.database, draft)
}

func (s *SQLDraftStore) GetInvites(ctx context.Context, userUuid uuid.UUID) ([]DraftInvite, error) {
	return getInvites(ctx, s.database, userUuid)
}

func (s *SQLDraftStore) GetInvite(ctx context.Context, inviteId int) (DraftInvite, error) {
	return getInvite(ctx, s.database, inviteId)
}

func (s *SQLDraftStore) GetNumPlayersInInvitedDraft(ctx context.Context, inviteId int) (int, error) {
	return getNumPlayersInInvitedDraft(ctx, s.database, inviteId)
}

func (s *SQLDraftStore) CancelOutstandingInvites(ctx context.Context, draftId int) error {
	return cancelOutstandingInvites(ctx, s.database, draftId)
}

func (s *SQLDraftStore) AcceptInvite(ctx context.Context, inviteId int) (int, uuid.UUID, error) {
	return acceptInvite(ctx, s.database, inviteId)
}

func (s *SQLDraftStore) AddPlayerToDraft(ctx context.Context, draftId int, player uuid.UUID) error {
	return addPlayerToDraft(ctx, s.database, draftId, player)
}

func (s *SQLDraftStore) InvitePlayer(ctx context.Context, draftId int, invitingUserUuid uuid.UUID, invitedUserUuid uuid.UUID) (int, error) {
	return invitePlayer(ctx, s.database, draftId, invitingUserUuid, invitedUserUuid)
}

func (s *SQLDraftStore) GetDraftPlayerId(ctx context.Context, draftId int, userUuid uuid.UUID) (int, error) {
	return getDraftPlayerId(ctx, s.database, draftId, userUuid)
}

func (s *SQLDraftStore) ShouldSkipPick(ctx context.Context, draftPlayerId int) (bool, error) {
	return shouldSkipPick(ctx, s.database, draftPlayerId)
}

func (s *SQLDraftStore) MarkShouldSkipPick(ctx context.Context, draftPlayerId int, shouldSkip bool) error {
	return markShouldSkipPick(ctx, s.database, draftPlayerId, shouldSkip)
}

func (s *SQLDraftStore) UpdateDraftStatus(ctx context.Context, draftId int, status DraftState) error {
	return updateDraftStatus(ctx, s.database, draftId, status)
}

func (s *SQLDraftStore) UpdateDraft(ctx context.Context, draft *DraftModel) error {
	return updateDraft(ctx, s.database, draft)
}

func (s *SQLDraftStore) GetPicks(ctx context.Context, draft int) ([]Pick, error) {
	return getPicks(ctx, s.database, draft)
}

func (s *SQLDraftStore) GetDraftPlayerUser(ctx context.Context, draftPlayerId int) (User, error) {
	return getDraftPlayerUser(ctx, s.database, draftPlayerId)
}

func (s *SQLDraftStore) MakePickAvailable(ctx context.Context, draftPlayerId int, availableTime time.Time, expirationTime time.Time) (int, error) {
	return makePickAvailable(ctx, s.database, draftPlayerId, availableTime, expirationTime)
}

func (s *SQLDraftStore) MakePick(ctx context.Context, pick Pick) error {
	return makePick(ctx, s.database, pick)
}

func (s *SQLDraftStore) NextPick(ctx context.Context, draftId int) (DraftPlayer, error) {
	return nextPick(ctx, s.database, draftId)
}

func (s *SQLDraftStore) GetCurrentPick(ctx context.Context, draftId int) (Pick, error) {
	return getCurrentPick(ctx, s.database, draftId)
}

func (s *SQLDraftStore) SkipPick(ctx context.Context, pickId int) error {
	return skipPick(ctx, s.database, pickId)
}

func (s *SQLDraftStore) UpdatePickExpirationTime(ctx context.Context, pickId int, expirationTime time.Time) error {
	return updatePickExpirationTime(ctx, s.database, pickId, expirationTime)
}

func (s *SQLDraftStore) GetPreviousPick(ctx context.Context, draftId int, currentPickId int) (Pick, error) {
	return getPreviousPick(ctx, s.database, draftId, currentPickId)
}

func (s *SQLDraftStore) DeletePick(ctx context.Context, pickId int) error {
	return deletePick(ctx, s.database, pickId)
}

func (s *SQLDraftStore) ResetPick(ctx context.Context, pickId int, expirationTime time.Time) error {
	return resetPick(ctx, s.database, pickId, expirationTime)
}

func (s *SQLDraftStore) GetDraftsInStatus(ctx context.Context, status DraftState) ([]int, error) {
	return getDraftsInStatus(ctx, s.database, status)
}

func (s *SQLDraftStore) RandomizePickOrder(ctx context.Context, draftId int) error {
	return randomizePickOrder(ctx, s.database, draftId)
}

func (s *SQLDraftStore) HasBeenPicked(ctx context.Context, draftId int, team string) (bool, error) {
	return hasBeenPicked(ctx, s.database, draftId, team)
}

func (s *SQLDraftStore) CancelInvite(ctx context.Context, inviteId int) error {
	return cancelInvite(ctx, s.database, inviteId)
}

func (s *SQLDraftStore) UninvitePlayer(ctx context.Context, draftId int, ownerUuid uuid.UUID, inviteId int) error {
	return uninvitePlayer(ctx, s.database, draftId, ownerUuid, inviteId)
}

func (s *SQLDraftStore) GetOutstandingInvitesForDraft(ctx context.Context, draftId int) ([]DraftInvite, error) {
	return getOutstandingInvitesForDraft(ctx, s.database, draftId)
}

func (s *SQLDraftStore) GetOverallLeaderboard(ctx context.Context, page int, perPage int) (LeaderboardPage, error) {
	return getOverallLeaderboard(ctx, s.database, page, perPage)
}

func (s *SQLDraftStore) SkipAndMakeNextPickAvailable(ctx context.Context, currentPickId int, nextDraftPlayerId int, availableTime time.Time, expirationTime time.Time) (int, error) {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if err := skipPick(ctx, tx, currentPickId); err != nil {
		return 0, err
	}
	pickId, err := makePickAvailable(ctx, tx, nextDraftPlayerId, availableTime, expirationTime)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return pickId, nil
}

func (s *SQLDraftStore) TransferOwnership(ctx context.Context, draftId int, newOwnerUuid uuid.UUID) error {
	return transferOwnership(ctx, s.database, draftId, newOwnerUuid)
}

func (s *SQLDraftStore) UndoLastPick(ctx context.Context, currentPickId int, previousPickId int, previousPickExpirationTime time.Time) error {
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := deletePick(ctx, tx, currentPickId); err != nil {
		return err
	}
	if err := resetPick(ctx, tx, previousPickId, previousPickExpirationTime); err != nil {
		return err
	}

	return tx.Commit()
}
