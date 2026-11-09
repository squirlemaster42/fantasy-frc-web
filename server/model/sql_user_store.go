package model

import (
	"context"
	"database/sql"

	"server/database"

	"uuid"
)

type SQLUserStore struct {
	database.SQLStore
}

func NewSQLUserStore(db *sql.DB) *SQLUserStore {
	return &SQLUserStore{SQLStore: database.NewSQLStore(db)}
}

func (s *SQLUserStore) GetUserBySessionToken(ctx context.Context, sessionToken string) (uuid.UUID, error) {
	return getUserBySessionToken(ctx, s.DB, sessionToken)
}

func (s *SQLUserStore) GetUsername(ctx context.Context, userUuid uuid.UUID) (string, error) {
	return getUsername(ctx, s.DB, userUuid)
}

func (s *SQLUserStore) SearchUsers(ctx context.Context, searchString string, draftId int) ([]User, error) {
	return searchUsers(ctx, s.DB, searchString, draftId)
}

func (s *SQLUserStore) ValidateSessionToken(ctx context.Context, sessionToken string) (bool, error) {
	return validateSessionToken(ctx, s.DB, sessionToken)
}

func (s *SQLUserStore) UsernameTaken(ctx context.Context, username string) (bool, error) {
	return usernameTaken(ctx, s.DB, username)
}

func (s *SQLUserStore) GetUserUuidByUsername(ctx context.Context, username string) (uuid.UUID, error) {
	return getUserUuidByUsername(ctx, s.DB, username)
}

func (s *SQLUserStore) GetPasswordHashByUsername(ctx context.Context, username string) (string, error) {
	return getPasswordHashByUsername(ctx, s.DB, username)
}

func (s *SQLUserStore) RegisterSession(ctx context.Context, userUuid uuid.UUID, sessionToken string) error {
	return registerSession(ctx, s.DB, userUuid, sessionToken)
}

func (s *SQLUserStore) UnRegisterSession(ctx context.Context, sessionToken string) error {
	return unregisterSession(ctx, s.DB, sessionToken)
}

func (s *SQLUserStore) RegisterUser(ctx context.Context, username string, passwordHash string) (uuid.UUID, error) {
	return registerUser(ctx, s.DB, username, passwordHash)
}

func (s *SQLUserStore) GetDiscordId(ctx context.Context, userUuid uuid.UUID) (string, error) {
	return getDiscordId(ctx, s.DB, userUuid)
}

func (s *SQLUserStore) UpdateDiscordId(ctx context.Context, userUuid uuid.UUID, discordId string) error {
	return updateDiscordId(ctx, s.DB, userUuid, discordId)
}

func (s *SQLUserStore) UpdatePassword(ctx context.Context, username string, passwordHash string) error {
	return updatePassword(ctx, s.DB, username, passwordHash)
}

func (s *SQLUserStore) InvalidateAllUserSessionsExcept(ctx context.Context, userUuid uuid.UUID, keepSessionToken string) error {
	return invalidateAllUserSessionsExcept(ctx, s.DB, userUuid, keepSessionToken)
}

func (s *SQLUserStore) UserIsAdmin(ctx context.Context, userUuid uuid.UUID) (bool, error) {
	return userIsAdmin(ctx, s.DB, userUuid)
}
