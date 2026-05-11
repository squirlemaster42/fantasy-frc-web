package model

import (
	"context"

	"github.com/google/uuid"
)

type DraftStore interface {
	GetDraft(ctx context.Context, draftId int) (DraftModel, error)
	GetDraftsForUser(ctx context.Context, userUuid uuid.UUID) ([]DraftModel, error)
	GetDraftsByName(ctx context.Context, searchString string) *[]DraftModel
	CreateDraft(ctx context.Context, draft *DraftModel) (int, error)
	GetDraftScore(ctx context.Context, draftId int) []DraftPlayer
	GetDraftPickRows(ctx context.Context, teamKeys []string) ([]DraftPickRow, error)
	GetInvites(ctx context.Context, userUuid uuid.UUID) []DraftInvite
	GetInvite(ctx context.Context, inviteId int) (DraftInvite, error)
	GetNumPlayersInInvitedDraft(ctx context.Context, inviteId int) int
	CancelOutstandingInvites(ctx context.Context, draftId int) error
	AcceptInvite(ctx context.Context, inviteId int) (int, uuid.UUID)
	AddPlayerToDraft(ctx context.Context, draftId int, player uuid.UUID)
	InvitePlayer(ctx context.Context, draftId int, invitingUserUuid uuid.UUID, invitedUserUuid uuid.UUID) (int, error)
	GetDraftPlayerId(ctx context.Context, draftId int, userUuid uuid.UUID) (int, error)
	ShouldSkipPick(ctx context.Context, draftPlayerId int) bool
	MarkShouldSkipPick(ctx context.Context, draftPlayerId int, shouldSkip bool) error
}
