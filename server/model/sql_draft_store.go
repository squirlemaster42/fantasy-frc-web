package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"uuid"

	"server/database"
)

type SQLDraftStore struct {
	database.SQLStore
	isTx bool
}

func NewSQLDraftStore(db *sql.DB) *SQLDraftStore {
	return &SQLDraftStore{SQLStore: database.NewSQLStore(db), isTx: false}
}

func (s *SQLDraftStore) RunInTransaction(ctx context.Context, fn func(db database.DBTX) error) error {
	if s.isTx {
		return fn(s.DB)
	}
	sqlDB, ok := s.DB.(*sql.DB)
	if !ok {
		return errors.New("runInTransaction called on a store without an underlying *sql.DB")
	}
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (s *SQLDraftStore) WithTx(tx database.DBTX) DraftStore {
	return &SQLDraftStore{SQLStore: database.NewSQLStore(tx), isTx: true}
}

func (s *SQLDraftStore) GetDraft(ctx context.Context, draftId int) (DraftModel, error) {
	return getDraft(ctx, s.DB, draftId)
}

func (s *SQLDraftStore) GetDraftsByName(ctx context.Context, searchString string) ([]DraftModel, error) {
	return getDraftsByName(ctx, s.DB, searchString)
}

func (s *SQLDraftStore) GetDraftScore(ctx context.Context, draftId int) ([]DraftPlayer, error) {
	return getDraftScore(ctx, s.DB, draftId)
}

func (s *SQLDraftStore) GetDraftPickRows(ctx context.Context, teamKeys []string) ([]DraftPickRow, error) {
	return getDraftPickRows(ctx, s.DB, teamKeys)
}

func (s *SQLDraftStore) SearchDrafts(ctx context.Context, search DraftSearchQuery) ([]DraftModel, error) {
	return searchDrafts(ctx, s.DB, search)
}

func (s *SQLDraftStore) CreateDraft(ctx context.Context, draft *DraftModel) (int, error) {
	var draftId int
	err := s.RunInTransaction(ctx, func(tx database.DBTX) error {
		var err error
		draftId, err = createDraft(ctx, tx, draft)
		return err
	})
	if err != nil {
		return 0, err
	}
	return draftId, nil
}

func (s *SQLDraftStore) GetInvites(ctx context.Context, userUuid uuid.UUID) ([]DraftInvite, error) {
	return getInvites(ctx, s.DB, userUuid)
}

func (s *SQLDraftStore) GetInvite(ctx context.Context, inviteId int) (DraftInvite, error) {
	return getInvite(ctx, s.DB, inviteId)
}

func (s *SQLDraftStore) GetNumPlayersInInvitedDraft(ctx context.Context, inviteId int) (int, error) {
	return getNumPlayersInInvitedDraft(ctx, s.DB, inviteId)
}

func (s *SQLDraftStore) GetNumPlayersInDraft(ctx context.Context, draftId int) (int, error) {
	return getNumPlayersInDraft(ctx, s.DB, draftId)
}

func (s *SQLDraftStore) LockDraft(ctx context.Context, draftId int) error {
	return lockDraft(ctx, s.DB, draftId)
}

func (s *SQLDraftStore) CancelOutstandingInvites(ctx context.Context, draftId int) error {
	return cancelOutstandingInvites(ctx, s.DB, draftId)
}

func (s *SQLDraftStore) AcceptInvite(ctx context.Context, inviteId int) (int, uuid.UUID, error) {
	return acceptInvite(ctx, s.DB, inviteId)
}

func (s *SQLDraftStore) AddPlayerToDraft(ctx context.Context, draftId int, player uuid.UUID) error {
	return addPlayerToDraft(ctx, s.DB, draftId, player)
}

func (s *SQLDraftStore) InvitePlayer(ctx context.Context, draftId int, invitingUserUuid uuid.UUID, invitedUserUuid uuid.UUID) (int, error) {
	return invitePlayer(ctx, s.DB, draftId, invitingUserUuid, invitedUserUuid)
}

func (s *SQLDraftStore) GetDraftPlayerId(ctx context.Context, draftId int, userUuid uuid.UUID) (int, error) {
	return getDraftPlayerId(ctx, s.DB, draftId, userUuid)
}

func (s *SQLDraftStore) ShouldSkipPick(ctx context.Context, draftPlayerId int) (bool, error) {
	return shouldSkipPick(ctx, s.DB, draftPlayerId)
}

func (s *SQLDraftStore) MarkShouldSkipPick(ctx context.Context, draftPlayerId int, shouldSkip bool) error {
	return markShouldSkipPick(ctx, s.DB, draftPlayerId, shouldSkip)
}

func (s *SQLDraftStore) UpdateDraftStatus(ctx context.Context, draftId int, status DraftState) error {
	return updateDraftStatus(ctx, s.DB, draftId, status)
}

func (s *SQLDraftStore) UpdateDraft(ctx context.Context, draft *DraftModel) error {
	return updateDraft(ctx, s.DB, draft)
}

func (s *SQLDraftStore) GetPicks(ctx context.Context, draft int) ([]Pick, error) {
	return getPicks(ctx, s.DB, draft)
}

func (s *SQLDraftStore) GetDraftPlayerUser(ctx context.Context, draftPlayerId int) (User, error) {
	return getDraftPlayerUser(ctx, s.DB, draftPlayerId)
}

func (s *SQLDraftStore) MakePickAvailable(ctx context.Context, draftPlayerId int, availableTime time.Time, expirationTime time.Time) (int, error) {
	return makePickAvailable(ctx, s.DB, draftPlayerId, availableTime, expirationTime)
}

func (s *SQLDraftStore) MakePick(ctx context.Context, pick Pick) error {
	return makePick(ctx, s.DB, pick)
}

func (s *SQLDraftStore) NextPick(ctx context.Context, draftId int) (DraftPlayer, error) {
	return nextPick(ctx, s.DB, draftId)
}

func (s *SQLDraftStore) GetCurrentPick(ctx context.Context, draftId int) (Pick, error) {
	return getCurrentPick(ctx, s.DB, draftId)
}

func (s *SQLDraftStore) SkipPick(ctx context.Context, pickId int) error {
	return skipPick(ctx, s.DB, pickId)
}

func (s *SQLDraftStore) UpdatePickExpirationTime(ctx context.Context, pickId int, expirationTime time.Time) error {
	return updatePickExpirationTime(ctx, s.DB, pickId, expirationTime)
}

func (s *SQLDraftStore) GetPreviousPick(ctx context.Context, draftId int, currentPickId int) (Pick, error) {
	return getPreviousPick(ctx, s.DB, draftId, currentPickId)
}

func (s *SQLDraftStore) DeletePick(ctx context.Context, pickId int) error {
	return deletePick(ctx, s.DB, pickId)
}

func (s *SQLDraftStore) ResetPick(ctx context.Context, pickId int, expirationTime time.Time) error {
	return resetPick(ctx, s.DB, pickId, expirationTime)
}

func (s *SQLDraftStore) GetDraftsInStatus(ctx context.Context, status DraftState) ([]int, error) {
	return getDraftsInStatus(ctx, s.DB, status)
}

func (s *SQLDraftStore) RandomizePickOrder(ctx context.Context, draftId int) error {
	return randomizePickOrder(ctx, s.DB, draftId)
}

func (s *SQLDraftStore) InitializeDraftPlayerClocks(ctx context.Context, draftId int, seconds int16) error {
	return initializeDraftPlayerClocks(ctx, s.DB, draftId, seconds)
}

func (s *SQLDraftStore) SetDraftPlayerRemainingTime(ctx context.Context, draftPlayerId int, seconds int) error {
	return setDraftPlayerRemainingTime(ctx, s.DB, draftPlayerId, seconds)
}

func (s *SQLDraftStore) HasBeenPicked(ctx context.Context, draftId int, team string) (bool, error) {
	return hasBeenPicked(ctx, s.DB, draftId, team)
}

func (s *SQLDraftStore) CancelInvite(ctx context.Context, inviteId int) error {
	return cancelInvite(ctx, s.DB, inviteId)
}

func (s *SQLDraftStore) UninvitePlayer(ctx context.Context, draftId int, ownerUuid uuid.UUID, inviteId int) error {
	return uninvitePlayer(ctx, s.DB, draftId, ownerUuid, inviteId)
}

func (s *SQLDraftStore) GetOutstandingInvitesForDraft(ctx context.Context, draftId int) ([]DraftInvite, error) {
	return getOutstandingInvitesForDraft(ctx, s.DB, draftId)
}

func (s *SQLDraftStore) GetOverallLeaderboard(ctx context.Context, page int, perPage int) (LeaderboardPage, error) {
	return getOverallLeaderboard(ctx, s.DB, page, perPage)
}

func (s *SQLDraftStore) TransferOwnership(ctx context.Context, draftId int, newOwnerUuid uuid.UUID) error {
	return transferOwnership(ctx, s.DB, draftId, newOwnerUuid)
}

func (s *SQLDraftStore) GetDraftsForUser(ctx context.Context, userUuid uuid.UUID) ([]DraftModel, error) {
	return getDraftsForUser(ctx, s.DB, userUuid)
}

func (s *SQLDraftStore) GetUserDraftNotificationPreferences(ctx context.Context, userUuid uuid.UUID) (map[int]DraftNotificationPreference, error) {
	return getUserDraftNotificationPreferences(ctx, s.DB, userUuid)
}

func (s *SQLDraftStore) UpdateUserDraftNotificationPreference(ctx context.Context, userUuid uuid.UUID, draftId int, upcomingMatch bool, pickTurn bool) error {
	return updateUserDraftNotificationPreference(ctx, s.DB, userUuid, draftId, upcomingMatch, pickTurn)
}
