package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

const (
	readTimeout  = 5 * time.Second
	writeTimeout = 10 * time.Second
)

type SQLDraftStore struct {
	db *sql.DB
}

func NewSQLDraftStore(db *sql.DB) *SQLDraftStore {
	return &SQLDraftStore{db: db}
}

func (s *SQLDraftStore) GetDraft(ctx context.Context, draftId int) (DraftModel, error) {
	ctx, cancel := WithQueryTimeout(ctx, readTimeout)
	defer cancel()
	return getDraft(ctx, s.db, draftId)
}

func (s *SQLDraftStore) GetDraftsByName(ctx context.Context, searchString string) ([]DraftModel, error) {
	return getDraftsByName(ctx, s.db, searchString)
}

func (s *SQLDraftStore) GetDraftScore(ctx context.Context, draftId int) ([]DraftPlayer, error) {
	return getDraftScore(ctx, s.db, draftId)
}

func (s *SQLDraftStore) GetDraftPickRows(ctx context.Context, teamKeys []string) ([]DraftPickRow, error) {
	return getDraftPickRows(ctx, s.db, teamKeys)
}

func (s *SQLDraftStore) GetDraftsForUser(ctx context.Context, userUuid uuid.UUID) ([]DraftModel, error) {
	return getDraftsForUser(ctx, s.db, userUuid)
}

func (s *SQLDraftStore) CreateDraft(ctx context.Context, draft *DraftModel) (int, error) {
	return createDraft(ctx, s.db, draft)
}

func (s *SQLDraftStore) GetInvites(ctx context.Context, userUuid uuid.UUID) ([]DraftInvite, error) {
	return getInvites(ctx, s.db, userUuid)
}

func (s *SQLDraftStore) GetInvite(ctx context.Context, inviteId int) (DraftInvite, error) {
	return getInvite(ctx, s.db, inviteId)
}

func (s *SQLDraftStore) GetNumPlayersInInvitedDraft(ctx context.Context, inviteId int) (int, error) {
	return getNumPlayersInInvitedDraft(ctx, s.db, inviteId)
}

func (s *SQLDraftStore) CancelOutstandingInvites(ctx context.Context, draftId int) error {
	return cancelOutstandingInvites(ctx, s.db, draftId)
}

func (s *SQLDraftStore) AcceptInvite(ctx context.Context, inviteId int) (int, uuid.UUID, error) {
	return acceptInvite(ctx, s.db, inviteId)
}

func (s *SQLDraftStore) AddPlayerToDraft(ctx context.Context, draftId int, player uuid.UUID) error {
	return addPlayerToDraft(ctx, s.db, draftId, player)
}

func (s *SQLDraftStore) InvitePlayer(ctx context.Context, draftId int, invitingUserUuid uuid.UUID, invitedUserUuid uuid.UUID) (int, error) {
	return invitePlayer(ctx, s.db, draftId, invitingUserUuid, invitedUserUuid)
}

func (s *SQLDraftStore) GetDraftPlayerId(ctx context.Context, draftId int, userUuid uuid.UUID) (int, error) {
	return getDraftPlayerId(ctx, s.db, draftId, userUuid)
}

func (s *SQLDraftStore) ShouldSkipPick(ctx context.Context, draftPlayerId int) (bool, error) {
	ctx, cancel := WithQueryTimeout(ctx, readTimeout)
	defer cancel()
	return shouldSkipPick(ctx, s.db, draftPlayerId)
}

func (s *SQLDraftStore) MarkShouldSkipPick(ctx context.Context, draftPlayerId int, shouldSkip bool) error {
	return markShouldSkipPick(ctx, s.db, draftPlayerId, shouldSkip)
}

func (s *SQLDraftStore) UpdateDraftStatus(ctx context.Context, draftId int, status DraftState) error {
	ctx, cancel := WithQueryTimeout(ctx, writeTimeout)
	defer cancel()
	return updateDraftStatus(ctx, s.db, draftId, status)
}

func (s *SQLDraftStore) UpdateDraft(ctx context.Context, draft *DraftModel) error {
	ctx, cancel := WithQueryTimeout(ctx, writeTimeout)
	defer cancel()
	return updateDraft(ctx, s.db, draft)
}

func (s *SQLDraftStore) GetPicks(ctx context.Context, draft int) ([]Pick, error) {
	ctx, cancel := WithQueryTimeout(ctx, readTimeout)
	defer cancel()
	return getPicks(ctx, s.db, draft)
}

func (s *SQLDraftStore) GetDraftPlayerUser(ctx context.Context, draftPlayerId int) (User, error) {
	return getDraftPlayerUser(ctx, s.db, draftPlayerId)
}

func (s *SQLDraftStore) MakePickAvailable(ctx context.Context, draftPlayerId int, availableTime time.Time, expirationTime time.Time) (int, error) {
	ctx, cancel := WithQueryTimeout(ctx, writeTimeout)
	defer cancel()
	return makePickAvailable(ctx, s.db, draftPlayerId, availableTime, expirationTime)
}

func (s *SQLDraftStore) MakePick(ctx context.Context, pick Pick) error {
	ctx, cancel := WithQueryTimeout(ctx, writeTimeout)
	defer cancel()
	return makePick(ctx, s.db, pick)
}

func (s *SQLDraftStore) NextPick(ctx context.Context, draftId int) (DraftPlayer, error) {
	ctx, cancel := WithQueryTimeout(ctx, readTimeout)
	defer cancel()
	return nextPick(ctx, s.db, draftId)
}

func (s *SQLDraftStore) GetCurrentPick(ctx context.Context, draftId int) (Pick, error) {
	ctx, cancel := WithQueryTimeout(ctx, readTimeout)
	defer cancel()
	return getCurrentPick(ctx, s.db, draftId)
}

func (s *SQLDraftStore) SkipPick(ctx context.Context, pickId int) error {
	ctx, cancel := WithQueryTimeout(ctx, writeTimeout)
	defer cancel()
	return skipPick(ctx, s.db, pickId)
}

func (s *SQLDraftStore) UpdatePickExpirationTime(ctx context.Context, pickId int, expirationTime time.Time) error {
	ctx, cancel := WithQueryTimeout(ctx, writeTimeout)
	defer cancel()
	return updatePickExpirationTime(ctx, s.db, pickId, expirationTime)
}

func (s *SQLDraftStore) GetPreviousPick(ctx context.Context, draftId int, currentPickId int) (Pick, error) {
	ctx, cancel := WithQueryTimeout(ctx, readTimeout)
	defer cancel()
	return getPreviousPick(ctx, s.db, draftId, currentPickId)
}

func (s *SQLDraftStore) DeletePick(ctx context.Context, pickId int) error {
	ctx, cancel := WithQueryTimeout(ctx, writeTimeout)
	defer cancel()
	return deletePick(ctx, s.db, pickId)
}

func (s *SQLDraftStore) ResetPick(ctx context.Context, pickId int, expirationTime time.Time) error {
	ctx, cancel := WithQueryTimeout(ctx, writeTimeout)
	defer cancel()
	return resetPick(ctx, s.db, pickId, expirationTime)
}

func (s *SQLDraftStore) GetDraftsInStatus(ctx context.Context, status DraftState) ([]int, error) {
	ctx, cancel := WithQueryTimeout(ctx, readTimeout)
	defer cancel()
	return getDraftsInStatus(ctx, s.db, status)
}

func (s *SQLDraftStore) GetDraftsToStart(ctx context.Context, cutoffDate time.Time) ([]int, error) {
	ctx, cancel := WithQueryTimeout(ctx, readTimeout)
	defer cancel()
	return getDraftsToStart(ctx, s.db, cutoffDate)
}

func (s *SQLDraftStore) RandomizePickOrder(ctx context.Context, draftId int) error {
	ctx, cancel := WithQueryTimeout(ctx, writeTimeout)
	defer cancel()
	return randomizePickOrder(ctx, s.db, draftId)
}

func (s *SQLDraftStore) HasBeenPicked(ctx context.Context, draftId int, team string) (bool, error) {
	return hasBeenPicked(ctx, s.db, draftId, team)
}
