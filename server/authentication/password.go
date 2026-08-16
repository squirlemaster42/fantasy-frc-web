package authentication

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordHasher abstracts password hashing and verification.
type PasswordHasher interface {
	// Hash returns a salted hash of the given plaintext password.
	Hash(password string) (string, error)
	// Compare returns nil if the password matches the hash.
	Compare(password, hash string) error
	// DummyHash returns a precomputed hash used for constant-time comparison
	// when a username does not exist, preventing timing-based enumeration.
	DummyHash() []byte
}

// dummyPassword is used only for generating timing-sink hashes for unknown usernames.
// It is never compared against a real credential.
var dummyPassword = []byte("dummy-password-that-is-never-correct")

// bcryptPasswordHasher implements PasswordHasher using bcrypt.
type bcryptPasswordHasher struct {
	cost      int
	dummyHash []byte
}

// NewBcryptPasswordHasher creates a bcrypt password hasher using the provided cost.
// The cost must be between bcrypt.MinCost and bcrypt.MaxCost.
func NewBcryptPasswordHasher(cost int) (PasswordHasher, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, fmt.Errorf("bcrypt cost %d out of range [%d, %d]", cost, bcrypt.MinCost, bcrypt.MaxCost)
	}

	dummyHash, err := bcrypt.GenerateFromPassword(dummyPassword, cost)
	if err != nil {
		return nil, fmt.Errorf("failed to generate dummy password hash: %w", err)
	}

	return &bcryptPasswordHasher{
		cost:      cost,
		dummyHash: dummyHash,
	}, nil
}

func (h *bcryptPasswordHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

func (h *bcryptPasswordHasher) Compare(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func (h *bcryptPasswordHasher) DummyHash() []byte {
	return h.dummyHash
}
