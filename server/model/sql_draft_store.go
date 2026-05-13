package model

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type SQLDraftStore struct {
	db *sql.DB
}

func NewSQLDraftStore(db *sql.DB) *SQLDraftStore {
	return &SQLDraftStore{db: db}
}

func (s *SQLDraftStore) GetDraft(ctx context.Context, draftId int) (DraftModel, error) {
	return GetDraft(ctx, s.db, draftId)
}

func (s *SQLDraftStore) GetDraftsByName(ctx context.Context, searchString string) ([]DraftModel, error) {
	return GetDraftsByName(ctx, s.db, searchString)
}

func (s *SQLDraftStore) GetDraftScore(ctx context.Context, draftId int) ([]DraftPlayer, error) {
	return GetDraftScore(ctx, s.db, draftId)
}

func (s *SQLDraftStore) GetDraftPickRows(ctx context.Context, teamKeys []string) ([]DraftPickRow, error) {
	return GetDraftPickRows(ctx, s.db, teamKeys)
}

func (s *SQLDraftStore) GetDraftsForUser(ctx context.Context, userUuid uuid.UUID) ([]DraftModel, error) {
	return GetDraftsForUser(ctx, s.db, userUuid)
}

func (s *SQLDraftStore) CreateDraft(ctx context.Context, draft *DraftModel) (int, error) {
	return CreateDraft(ctx, s.db, draft)
}

func (s *SQLDraftStore) GetInvites(ctx context.Context, userUuid uuid.UUID) ([]DraftInvite, error) {
	return GetInvites(ctx, s.db, userUuid)
}

func (s *SQLDraftStore) GetInvite(ctx context.Context, inviteId int) (DraftInvite, error) {
	return GetInvite(ctx, s.db, inviteId)
}

func (s *SQLDraftStore) GetNumPlayersInInvitedDraft(ctx context.Context, inviteId int) (int, error) {
	return GetNumPlayersInInvitedDraft(ctx, s.db, inviteId)
}

func (s *SQLDraftStore) CancelOutstandingInvites(ctx context.Context, draftId int) error {
	return CancelOutstandingInvites(ctx, s.db, draftId)
}

func (s *SQLDraftStore) AcceptInvite(ctx context.Context, inviteId int) (int, uuid.UUID, error) {
	return AcceptInvite(ctx, s.db, inviteId)
}

func (s *SQLDraftStore) AddPlayerToDraft(ctx context.Context, draftId int, player uuid.UUID) error {
	return AddPlayerToDraft(ctx, s.db, draftId, player)
}

func (s *SQLDraftStore) InvitePlayer(ctx context.Context, draftId int, invitingUserUuid uuid.UUID, invitedUserUuid uuid.UUID) (int, error) {
	return InvitePlayer(ctx, s.db, draftId, invitingUserUuid, invitedUserUuid)
}

func (s *SQLDraftStore) GetDraftPlayerId(ctx context.Context, draftId int, userUuid uuid.UUID) (int, error) {
	return GetDraftPlayerId(ctx, s.db, draftId, userUuid)
}

func (s *SQLDraftStore) ShouldSkipPick(ctx context.Context, draftPlayerId int) (bool, error) {
	return ShouldSkipPick(ctx, s.db, draftPlayerId)
}

func (s *SQLDraftStore) MarkShouldSkipPick(ctx context.Context, draftPlayerId int, shouldSkip bool) error {
	return MarkShouldSkipPick(ctx, s.db, draftPlayerId, shouldSkip)
}
