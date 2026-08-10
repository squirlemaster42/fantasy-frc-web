package authentication

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestNewBcryptPasswordHasher_InvalidCost(t *testing.T) {
	_, err := NewBcryptPasswordHasher(3)
	assert.Error(t, err)

	_, err = NewBcryptPasswordHasher(32)
	assert.Error(t, err)
}

func TestBcryptPasswordHasher_HashAndCompare(t *testing.T) {
	hasher, err := NewBcryptPasswordHasher(bcrypt.MinCost)
	require.NoError(t, err)

	hash, err := hasher.Hash("Secret123!")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, "Secret123!", hash)

	assert.NoError(t, hasher.Compare("Secret123!", hash))
	assert.Error(t, hasher.Compare("wrongpassword", hash))
}

func TestBcryptPasswordHasher_DummyHashUsesConfiguredCost(t *testing.T) {
	hasher, err := NewBcryptPasswordHasher(bcrypt.MinCost)
	require.NoError(t, err)

	dummyHash := hasher.DummyHash()
	require.NotEmpty(t, dummyHash)

	cost, err := bcrypt.Cost(dummyHash)
	require.NoError(t, err)
	assert.Equal(t, bcrypt.MinCost, cost)
}
