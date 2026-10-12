package authentication

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
)

// generateSessionToken creates a cryptographically random 128-bit session token.
func generateSessionToken() (string, error) {
	randomBytes := make([]byte, SessionTokenBytes())
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base32.StdEncoding.EncodeToString(randomBytes), nil
}
