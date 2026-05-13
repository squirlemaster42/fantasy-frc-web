package model

import (
	"context"

	"github.com/google/uuid"
)

type DraftStore interface {
	GetDraft(ctx context.Context, draftId int) (DraftModel, error)
	GetDraftsForUser(ctx context.Context, userUuid uuid.UUID) ([]DraftModel, error)
	GetDraftsByName(ctx context.Context, searchString string) ([]DraftModel, error)
	CreateDraft(ctx context.Context, draft *DraftModel) (int, error)
	GetDraftScore(ctx context.Context, draftId int) ([]DraftPlayer, error)
	GetDraftPickRows(ctx context.Context, teamKeys []string) ([]DraftPickRow, error)
	GetInvites(ctx context.Context, userUuid uuid.UUID) ([]DraftInvite, error)
	GetInvite(ctx context.Context, inviteId int) (DraftInvite, error)
	GetNumPlayersInInvitedDraft(ctx context.Context, inviteId int) (int, error)
	CancelOutstandingInvites(ctx context.Context, draftId int) error
	AcceptInvite(ctx context.Context, inviteId int) (int, uuid.UUID, error)
	AddPlayerToDraft(ctx context.Context, draftId int, player uuid.UUID) error
	InvitePlayer(ctx context.Context, draftId int, invitingUserUuid uuid.UUID, invitedUserUuid uuid.UUID) (int, error)
	GetDraftPlayerId(ctx context.Context, draftId int, userUuid uuid.UUID) (int, error)
	ShouldSkipPick(ctx context.Context, draftPlayerId int) (bool, error)
	MarkShouldSkipPick(ctx context.Context, draftPlayerId int, shouldSkip bool) error
}
