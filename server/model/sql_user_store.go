package model

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type SQLUserStore struct {
	db *sql.DB
}

func NewSQLUserStore(db *sql.DB) *SQLUserStore {
	return &SQLUserStore{db: db}
}

func (s *SQLUserStore) GetUserBySessionToken(ctx context.Context, sessionToken string) (uuid.UUID, error) {
	return getUserBySessionToken(ctx, s.db, sessionToken)
}

func (s *SQLUserStore) GetUsername(ctx context.Context, userUuid uuid.UUID) (string, error) {
	return getUsername(ctx, s.db, userUuid)
}

func (s *SQLUserStore) SearchUsers(ctx context.Context, searchString string, draftId int) ([]User, error) {
	return searchUsers(ctx, s.db, searchString, draftId)
}

func (s *SQLUserStore) ValidateSessionToken(ctx context.Context, sessionToken string) (bool, error) {
	return validateSessionToken(ctx, s.db, sessionToken)
}

func (s *SQLUserStore) UsernameTaken(ctx context.Context, username string) (bool, error) {
	return usernameTaken(ctx, s.db, username)
}

func (s *SQLUserStore) ValidateLogin(ctx context.Context, username string, password string) (bool, error) {
	return validateLogin(ctx, s.db, username, password)
}

func (s *SQLUserStore) GetUserUuidByUsername(ctx context.Context, username string) (uuid.UUID, error) {
	return getUserUuidByUsername(ctx, s.db, username)
}

func (s *SQLUserStore) RegisterSession(ctx context.Context, userUuid uuid.UUID, sessionToken string) error {
	return registerSession(ctx, s.db, userUuid, sessionToken)
}

func (s *SQLUserStore) UnRegisterSession(ctx context.Context, sessionToken string) error {
	return unregisterSession(ctx, s.db, sessionToken)
}

func (s *SQLUserStore) RegisterUser(ctx context.Context, username string, password string) (uuid.UUID, error) {
	return registerUser(ctx, s.db, username, password)
}

func (s *SQLUserStore) GetDiscordId(ctx context.Context, userUuid uuid.UUID) (string, error) {
	return getDiscordId(ctx, s.db, userUuid)
}

func (s *SQLUserStore) UpdateDiscordId(ctx context.Context, userUuid uuid.UUID, discordId string) error {
	return updateDiscordId(ctx, s.db, userUuid, discordId)
}

func (s *SQLUserStore) UpdatePassword(ctx context.Context, username string, newPassword string) error {
	return updatePassword(ctx, s.db, username, newPassword)
}

func (s *SQLUserStore) InvalidateAllUserSessionsExcept(ctx context.Context, userUuid uuid.UUID, keepSessionToken string) error {
	return invalidateAllUserSessionsExcept(ctx, s.db, userUuid, keepSessionToken)
}

func (s *SQLUserStore) UserIsAdmin(ctx context.Context, userUuid uuid.UUID) (bool, error) {
	return userIsAdmin(ctx, s.db, userUuid)
}
