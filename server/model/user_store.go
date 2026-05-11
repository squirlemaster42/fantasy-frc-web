package model

import (
	"context"

	"github.com/google/uuid"
)

type UserStore interface {
	GetUserBySessionToken(ctx context.Context, sessionToken string) uuid.UUID
	GetUsername(ctx context.Context, userUuid uuid.UUID) string
	SearchUsers(ctx context.Context, searchString string, draftId int) ([]User, error)
	ValidateSessionToken(ctx context.Context, sessionToken string) bool
	UsernameTaken(ctx context.Context, username string) (bool, error)
	ValidateLogin(ctx context.Context, username string, password string) bool
	GetUserUuidByUsername(ctx context.Context, username string) uuid.UUID
	RegisterSession(ctx context.Context, userUuid uuid.UUID, sessionToken string)
	UnRegisterSession(ctx context.Context, sessionToken string)
	RegisterUser(ctx context.Context, username string, password string) uuid.UUID
	GetDiscordId(ctx context.Context, userUuid uuid.UUID) string
	UpdateDiscordId(ctx context.Context, userUuid uuid.UUID, discordId string)
	UpdatePassword(ctx context.Context, username string, newPassword string)
}
