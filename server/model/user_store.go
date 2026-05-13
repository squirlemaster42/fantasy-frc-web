package model

import (
	"context"

	"github.com/google/uuid"
)

type UserStore interface {
	GetUserBySessionToken(ctx context.Context, sessionToken string) (uuid.UUID, error)
	GetUsername(ctx context.Context, userUuid uuid.UUID) (string, error)
	SearchUsers(ctx context.Context, searchString string, draftId int) ([]User, error)
	ValidateSessionToken(ctx context.Context, sessionToken string) (bool, error)
	UsernameTaken(ctx context.Context, username string) (bool, error)
	ValidateLogin(ctx context.Context, username string, password string) (bool, error)
	GetUserUuidByUsername(ctx context.Context, username string) (uuid.UUID, error)
	RegisterSession(ctx context.Context, userUuid uuid.UUID, sessionToken string) error
	UnRegisterSession(ctx context.Context, sessionToken string) error
	RegisterUser(ctx context.Context, username string, password string) (uuid.UUID, error)
	GetDiscordId(ctx context.Context, userUuid uuid.UUID) (string, error)
	UpdateDiscordId(ctx context.Context, userUuid uuid.UUID, discordId string) error
	UpdatePassword(ctx context.Context, username string, newPassword string) error
	InvalidateAllUserSessionsExcept(ctx context.Context, userUuid uuid.UUID, keepSessionToken string) error
	UserIsAdmin(ctx context.Context, userUuid uuid.UUID) (bool, error)
}
