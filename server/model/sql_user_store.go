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

func (s *SQLUserStore) GetUserBySessionToken(ctx context.Context, sessionToken string) uuid.UUID {
	return GetUserBySessionToken(ctx, s.db, sessionToken)
}

func (s *SQLUserStore) GetUsername(ctx context.Context, userUuid uuid.UUID) string {
	return GetUsername(ctx, s.db, userUuid)
}

func (s *SQLUserStore) SearchUsers(ctx context.Context, searchString string, draftId int) ([]User, error) {
	return SearchUsers(ctx, s.db, searchString, draftId)
}

func (s *SQLUserStore) ValidateSessionToken(ctx context.Context, sessionToken string) bool {
	return ValidateSessionToken(ctx, s.db, sessionToken)
}

func (s *SQLUserStore) UsernameTaken(ctx context.Context, username string) (bool, error) {
	return UsernameTaken(ctx, s.db, username)
}

func (s *SQLUserStore) ValidateLogin(ctx context.Context, username string, password string) bool {
	return ValidateLogin(ctx, s.db, username, password)
}

func (s *SQLUserStore) GetUserUuidByUsername(ctx context.Context, username string) uuid.UUID {
	return GetUserUuidByUsername(ctx, s.db, username)
}

func (s *SQLUserStore) RegisterSession(ctx context.Context, userUuid uuid.UUID, sessionToken string) {
	RegisterSession(ctx, s.db, userUuid, sessionToken)
}

func (s *SQLUserStore) UnRegisterSession(ctx context.Context, sessionToken string) {
	UnRegisterSession(ctx, s.db, sessionToken)
}

func (s *SQLUserStore) RegisterUser(ctx context.Context, username string, password string) uuid.UUID {
	return RegisterUser(ctx, s.db, username, password)
}

func (s *SQLUserStore) GetDiscordId(ctx context.Context, userUuid uuid.UUID) string {
	return GetDiscordId(ctx, s.db, userUuid)
}

func (s *SQLUserStore) UpdateDiscordId(ctx context.Context, userUuid uuid.UUID, discordId string) {
	UpdateDiscordId(ctx, s.db, userUuid, discordId)
}

func (s *SQLUserStore) UpdatePassword(ctx context.Context, username string, newPassword string) {
	UpdatePassword(ctx, s.db, username, newPassword)
}
