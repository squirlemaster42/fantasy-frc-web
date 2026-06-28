// Package jwt provides stateless JWT signing and validation using only the Go
// standard library. Tokens are signed with HS256.
package jwt

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"server/log"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	Issuer = "fantasy-frc"
	Audience = "api"
	Algorithm = "HS256"
)

const minSigningKeyLength = 32

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
	ErrWeakKey      = errors.New("signing key must be at least 32 bytes")
)

// Claims represents the JWT payload.
type Claims struct {
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
	Audience  string `json:"aud"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
	TokenID   string `json:"jti"`
}

// header is the fixed JWT header for HS256 tokens.
type header struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

// Sign creates a new JWT for the given user that expires after duration.
func Sign(userUuid uuid.UUID, signingKey []byte, duration time.Duration) (string, error) {
	if len(signingKey) < minSigningKeyLength {
		return "", ErrWeakKey
	}

	now := time.Now().UTC()
	jti, err := generateTokenID()
	if err != nil {
		return "", fmt.Errorf("failed to generate token id: %w", err)
	}

	h := header{Algorithm: Algorithm, Type: "JWT"}
	headerBytes, err := json.Marshal(h)
	if err != nil {
		return "", fmt.Errorf("failed to marshal jwt header: %w", err)
	}

	claims := Claims{
		Subject:   userUuid.String(),
		Issuer:    Issuer,
		Audience:  Audience,
		ExpiresAt: now.Add(duration).Unix(),
		IssuedAt:  now.Unix(),
		TokenID:   jti,
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal jwt claims: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signingInput := encodedHeader + "." + encodedClaims

	signature := hmacSHA256(signingInput, signingKey)
	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + encodedSignature, nil
}

// Validate parses and validates a JWT, returning the user UUID from the subject claim.
func Validate(tokenString string, signingKey []byte) (uuid.UUID, error) {
	if len(signingKey) < minSigningKeyLength {
		return uuid.UUID{}, ErrWeakKey
	}

	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return uuid.UUID{}, ErrInvalidToken
	}

	encodedHeader, encodedClaims, encodedSignature := parts[0], parts[1], parts[2]

	signingInput := encodedHeader + "." + encodedClaims
	expectedSignature := hmacSHA256(signingInput, signingKey)
	decodedSignature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}
	if len(decodedSignature) != len(expectedSignature) || subtle.ConstantTimeCompare(decodedSignature, expectedSignature) != 1 {
		return uuid.UUID{}, ErrInvalidToken
	}

	headerBytes, err := base64.RawURLEncoding.DecodeString(encodedHeader)
	if err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}
	var h header
	if err := json.Unmarshal(headerBytes, &h); err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}
	if h.Algorithm != Algorithm {
		return uuid.UUID{}, ErrInvalidToken
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(encodedClaims)
	if err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}
	var claims Claims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}

	if claims.Issuer != Issuer {
		return uuid.UUID{}, ErrInvalidToken
	}
	if claims.Audience != Audience {
		return uuid.UUID{}, ErrInvalidToken
	}
	if claims.ExpiresAt < time.Now().UTC().Unix() {
		return uuid.UUID{}, ErrExpiredToken
	}

	userUuid, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.UUID{}, ErrInvalidToken
	}

	return userUuid, nil
}

// hmacSHA256 returns the HMAC-SHA256 of input using key.
func hmacSHA256(input string, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(input))
	return mac.Sum(nil)
}

// generateTokenID returns a cryptographically secure random token id.
func generateTokenID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateSecret creates a high-entropy secret suitable for an API client secret.
func GenerateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// GenerateClientID creates a unique public client identifier.
func GenerateClientID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "ffrc_" + base64.RawURLEncoding.EncodeToString(b), nil
}

// LogValidationFailure logs a structured validation failure without exposing
// the token value.
func LogValidationFailure(ctx context.Context, err error) {
	var reason string
	switch {
	case errors.Is(err, ErrExpiredToken):
		reason = "expired"
	case errors.Is(err, ErrInvalidToken):
		reason = "invalid"
	default:
		reason = "unknown"
	}
	log.Warn(ctx, "JWT validation failed", "reason", reason)
}

