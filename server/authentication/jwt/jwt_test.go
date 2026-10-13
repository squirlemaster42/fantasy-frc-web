package jwt

import (
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/stretchr/testify/assert"
)

func TestSignAndValidate(t *testing.T) {
	key := []byte("super-secret-key-that-is-32-bytes!")
	userUuid := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")

	token, err := Sign(userUuid, key, 15*time.Minute, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, 3, len(strings.Split(token, ".")))

	validatedUuid, scopes, err := Validate(token, key)
	assert.NoError(t, err)
	assert.Equal(t, userUuid, validatedUuid)
	assert.Empty(t, scopes)
}

func TestValidateExpiredToken(t *testing.T) {
	key := []byte("super-secret-key-that-is-32-bytes!")
	userUuid := uuid.New()

	token, err := Sign(userUuid, key, -1*time.Second, nil)
	assert.NoError(t, err)

	_, _, err = Validate(token, key)
	assert.ErrorIs(t, err, ErrExpiredToken)
}

func TestValidateInvalidSignature(t *testing.T) {
	key := []byte("super-secret-key-that-is-32-bytes!")
	wrongKey := []byte("different-secret-key-that-is-32-by")
	userUuid := uuid.New()

	token, err := Sign(userUuid, key, 15*time.Minute, nil)
	assert.NoError(t, err)

	_, _, err = Validate(token, wrongKey)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateMalformedToken(t *testing.T) {
	key := []byte("super-secret-key-that-is-32-bytes!")

	_, _, err := Validate("not.a.jwt", key)
	assert.ErrorIs(t, err, ErrInvalidToken)

	_, _, err = Validate("only.two.parts", key)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestValidateWrongAlgorithm(t *testing.T) {
	key := []byte("super-secret-key-that-is-32-bytes!")
	// A token with header {"alg":"none","typ":"JWT"} and empty signature
	token := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJzdWIiOiJhMGVlYmM5OS05YzBiLTRlZjgtYmI2ZC02YmI5YmQzODBhMTEiLCJpc3MiOiJmYW50YXN5LWZyYyIsImF1ZCI6ImFwaSIsImV4cCI6OTk5OTk5OTk5OSwiaWF0IjoxfQ."

	_, _, err := Validate(token, key)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestSignWeakKey(t *testing.T) {
	key := []byte("short")
	userUuid := uuid.New()

	_, err := Sign(userUuid, key, 15*time.Minute, nil)
	assert.ErrorIs(t, err, ErrWeakKey)
}

func TestValidateWeakKey(t *testing.T) {
	key := []byte("short")

	_, _, err := Validate("any.token.here", key)
	assert.ErrorIs(t, err, ErrWeakKey)
}

func TestSignAndValidateWithScopes(t *testing.T) {
	key := []byte("super-secret-key-that-is-32-bytes!")
	userUuid := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	scopes := []string{"drafts:read", "drafts:write"}

	token, err := Sign(userUuid, key, 15*time.Minute, scopes)
	assert.NoError(t, err)

	validatedUuid, validatedScopes, err := Validate(token, key)
	assert.NoError(t, err)
	assert.Equal(t, userUuid, validatedUuid)
	assert.Equal(t, scopes, validatedScopes)
}

func TestGenerateClientIDAndSecret(t *testing.T) {
	clientId, err := GenerateClientID()
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(clientId, "ffrc_"))

	secret, err := GenerateSecret()
	assert.NoError(t, err)
	assert.NotEmpty(t, secret)
	assert.Greater(t, len(secret), 20)
}
