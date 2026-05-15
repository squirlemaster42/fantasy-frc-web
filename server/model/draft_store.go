package model

import (
	"context"
	"time"

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
	UpdateDraftStatus(ctx context.Context, draftId int, status DraftState) error
	UpdateDraft(ctx context.Context, draft *DraftModel) error
	GetPicks(ctx context.Context, draft int) ([]Pick, error)
	GetDraftPlayerUser(ctx context.Context, draftPlayerId int) (User, error)
	MakePickAvailable(ctx context.Context, draftPlayerId int, availableTime time.Time, expirationTime time.Time) (int, error)
	MakePick(ctx context.Context, pick Pick) error
	NextPick(ctx context.Context, draftId int) (DraftPlayer, error)
	GetCurrentPick(ctx context.Context, draftId int) (Pick, error)
	SkipPick(ctx context.Context, pickId int) error
	UpdatePickExpirationTime(ctx context.Context, pickId int, expirationTime time.Time) error
	GetPreviousPick(ctx context.Context, draftId int, currentPickId int) (Pick, error)
	DeletePick(ctx context.Context, pickId int) error
	ResetPick(ctx context.Context, pickId int, expirationTime time.Time) error
	GetDraftsInStatus(ctx context.Context, status DraftState) ([]int, error)
	GetDraftsToStart(ctx context.Context, cutoffDate time.Time) ([]int, error)
	RandomizePickOrder(ctx context.Context, draftId int) error
	HasBeenPicked(ctx context.Context, draftId int, team string) (bool, error)
}
