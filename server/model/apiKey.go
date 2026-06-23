package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/assert"
	"server/authentication/jwt"
	"server/log"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ApiKey represents a machine-to-machine integration owned by a user.
type ApiKey struct {
	Id          uuid.UUID
	UserUuid    uuid.UUID
	ClientId    string
	DisplayName string
	Scopes      StringArray
	Revoked     bool
	CreatedAt   time.Time
	LastUsedAt  sql.NullTime
}

// ApiKeyStore persists and validates API keys.
type ApiKeyStore interface {
	CreateApiKey(ctx context.Context, userUuid uuid.UUID, displayName string) (*ApiKey, string, error)
	ValidateApiKey(ctx context.Context, clientId string, clientSecret string) (uuid.UUID, error)
	GetApiKey(ctx context.Context, id uuid.UUID) (*ApiKey, error)
	GetApiKeysForUser(ctx context.Context, userUuid uuid.UUID) ([]ApiKey, error)
	RevokeApiKey(ctx context.Context, id uuid.UUID, userUuid uuid.UUID) error
}

// SQLApiKeyStore is a Postgres-backed ApiKeyStore.
type SQLApiKeyStore struct {
	db *sql.DB
}

func NewSQLApiKeyStore(db *sql.DB) *SQLApiKeyStore {
	return &SQLApiKeyStore{db: db}
}

func (s *SQLApiKeyStore) CreateApiKey(ctx context.Context, userUuid uuid.UUID, displayName string) (*ApiKey, string, error) {
	assert := assert.CreateAssertWithContext("CreateApiKey")
	assert.AddContext("UserUuid", userUuid)

	clientId, err := jwt.GenerateClientID()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate client id: %w", err)
	}

	clientSecret, err := jwt.GenerateSecret()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate client secret: %w", err)
	}

	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(clientSecret), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash client secret: %w", err)
	}

	query := `INSERT INTO UserApiKeys (UserUuid, ClientId, ClientSecretHash, DisplayName)
			  VALUES ($1, $2, $3, $4)
			  RETURNING Id, UserUuid, ClientId, DisplayName, Scopes, Revoked, CreatedAt, LastUsedAt;`
	stmt, err := s.db.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "CreateApiKey: Failed to close statement", "error", err)
		}
	}()

	var key ApiKey
	err = stmt.QueryRowContext(ctx, userUuid, clientId, string(hashedSecret), displayName).Scan(
		&key.Id,
		&key.UserUuid,
		&key.ClientId,
		&key.DisplayName,
		&key.Scopes,
		&key.Revoked,
		&key.CreatedAt,
		&key.LastUsedAt,
	)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create api key: %w", err)
	}

	return &key, clientSecret, nil
}

func (s *SQLApiKeyStore) ValidateApiKey(ctx context.Context, clientId string, clientSecret string) (uuid.UUID, error) {
	assert := assert.CreateAssertWithContext("ValidateApiKey")
	assert.AddContext("ClientId", clientId)

	query := `SELECT UserUuid, ClientSecretHash
			  FROM UserApiKeys
			  WHERE ClientId = $1 AND Revoked = FALSE;`
	stmt, err := s.db.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "ValidateApiKey: Failed to close statement", "error", err)
		}
	}()

	var userUuid uuid.UUID
	var secretHash string
	err = stmt.QueryRowContext(ctx, clientId).Scan(&userUuid, &secretHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.UUID{}, ErrInvalidApiKey
		}
		return uuid.UUID{}, fmt.Errorf("failed to validate api key: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(secretHash), []byte(clientSecret)); err != nil {
		return uuid.UUID{}, ErrInvalidApiKey
	}

	if err := s.updateLastUsedAt(ctx, clientId); err != nil {
		log.Warn(ctx, "ValidateApiKey: Failed to update last used time", "error", err)
	}

	return userUuid, nil
}

func (s *SQLApiKeyStore) GetApiKey(ctx context.Context, id uuid.UUID) (*ApiKey, error) {
	assert := assert.CreateAssertWithContext("GetApiKey")
	assert.AddContext("Id", id)

	query := `SELECT Id, UserUuid, ClientId, DisplayName, Scopes, Revoked, CreatedAt, LastUsedAt
			  FROM UserApiKeys
			  WHERE Id = $1;`
	stmt, err := s.db.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "GetApiKey: Failed to close statement", "error", err)
		}
	}()

	var key ApiKey
	err = stmt.QueryRowContext(ctx, id).Scan(
		&key.Id,
		&key.UserUuid,
		&key.ClientId,
		&key.DisplayName,
		&key.Scopes,
		&key.Revoked,
		&key.CreatedAt,
		&key.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get api key: %w", err)
	}

	return &key, nil
}

func (s *SQLApiKeyStore) GetApiKeysForUser(ctx context.Context, userUuid uuid.UUID) ([]ApiKey, error) {
	assert := assert.CreateAssertWithContext("GetApiKeysForUser")
	assert.AddContext("UserUuid", userUuid)

	query := `SELECT Id, UserUuid, ClientId, DisplayName, Scopes, Revoked, CreatedAt, LastUsedAt
			  FROM UserApiKeys
			  WHERE UserUuid = $1
			  ORDER BY CreatedAt DESC;`
	stmt, err := s.db.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "GetApiKeysForUser: Failed to close statement", "error", err)
		}
	}()

	rows, err := stmt.QueryContext(ctx, userUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to query api keys: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Warn(ctx, "GetApiKeysForUser: Failed to close rows", "error", err)
		}
	}()

	keys := make([]ApiKey, 0)
	for rows.Next() {
		var key ApiKey
		err := rows.Scan(
			&key.Id,
			&key.UserUuid,
			&key.ClientId,
			&key.DisplayName,
			&key.Scopes,
			&key.Revoked,
			&key.CreatedAt,
			&key.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan api key: %w", err)
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func (s *SQLApiKeyStore) RevokeApiKey(ctx context.Context, id uuid.UUID, userUuid uuid.UUID) error {
	assert := assert.CreateAssertWithContext("RevokeApiKey")
	assert.AddContext("Id", id)
	assert.AddContext("UserUuid", userUuid)

	query := `UPDATE UserApiKeys
			  SET Revoked = TRUE
			  WHERE Id = $1 AND UserUuid = $2;`
	stmt, err := s.db.PrepareContext(ctx, query)
	assert.NoError(ctx, err, "failed to prepare statement")
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "RevokeApiKey: Failed to close statement", "error", err)
		}
	}()

	result, err := stmt.ExecContext(ctx, id, userUuid)
	if err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check revoke result: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *SQLApiKeyStore) updateLastUsedAt(ctx context.Context, clientId string) error {
	query := `UPDATE UserApiKeys SET LastUsedAt = now()::timestamptz WHERE ClientId = $1;`
	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			log.Warn(ctx, "updateLastUsedAt: Failed to close statement", "error", err)
		}
	}()
	_, err = stmt.ExecContext(ctx, clientId)
	return err
}

var (
	ErrInvalidApiKey = errors.New("invalid api key")
	ErrNotFound      = errors.New("not found")
)
