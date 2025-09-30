package validation

import (
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Valid          bool
	Error          string
	SanitizedValue interface{}
}

// ValidateDraftID validates and sanitizes a draft ID string
func ValidateDraftID(input string) ValidationResult {
	// Trim whitespace
	trimmed := strings.TrimSpace(input)

	// Check if empty
	if trimmed == "" {
		return ValidationResult{Valid: false, Error: "draft ID cannot be empty"}
	}

	// Parse as integer
	id, err := strconv.Atoi(trimmed)
	if err != nil {
		return ValidationResult{Valid: false, Error: "draft ID must be numeric"}
	}

	// Check bounds (reasonable limits for draft IDs)
	if id <= 0 {
		return ValidationResult{Valid: false, Error: "draft ID must be positive"}
	}
	if id > 999999 {
		return ValidationResult{Valid: false, Error: "draft ID too large"}
	}

	return ValidationResult{Valid: true, SanitizedValue: id}
}

// ValidateUsername validates and sanitizes a username
func ValidateUsername(input string) ValidationResult {
	// Trim whitespace
	trimmed := strings.TrimSpace(input)

	// Check length
	if len(trimmed) < 3 {
		return ValidationResult{Valid: false, Error: "username must be at least 3 characters"}
	}
	if len(trimmed) > 30 {
		return ValidationResult{Valid: false, Error: "username must be no more than 30 characters"}
	}

	// Check for valid characters (alphanumeric, underscore, no spaces)
	validPattern := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	if !validPattern.MatchString(trimmed) {
		return ValidationResult{Valid: false, Error: "username can only contain letters, numbers, and underscores"}
	}

	// HTML escape for safety
	sanitized := html.EscapeString(trimmed)

	return ValidationResult{Valid: true, SanitizedValue: sanitized}
}

// ValidatePassword validates password strength
func ValidatePassword(input string) ValidationResult {
	// Check minimum length
	if len(input) < 8 {
		return ValidationResult{Valid: false, Error: "password must be at least 8 characters"}
	}

	// Check maximum length (prevent DoS)
	if len(input) > 128 {
		return ValidationResult{Valid: false, Error: "password must be no more than 128 characters"}
	}

	// Check for common weak passwords first (before complexity checks)
	commonPasswords := []string{"password", "123456", "qwerty", "admin", "letmein"}
	for _, common := range commonPasswords {
		if strings.Contains(strings.ToLower(input), common) {
			return ValidationResult{Valid: false, Error: "password is too common"}
		}
	}

	// Check for required character types
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(input)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(input)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(input)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(input)

	if !hasUpper {
		return ValidationResult{Valid: false, Error: "password must contain at least one uppercase letter"}
	}
	if !hasLower {
		return ValidationResult{Valid: false, Error: "password must contain at least one lowercase letter"}
	}
	if !hasDigit {
		return ValidationResult{Valid: false, Error: "password must contain at least one number"}
	}
	if !hasSpecial {
		return ValidationResult{Valid: false, Error: "password must contain at least one special character"}
	}

	return ValidationResult{Valid: true, SanitizedValue: input}
}

// ValidateDraftName validates and sanitizes a draft name
func ValidateDraftName(input string) ValidationResult {
	// Trim whitespace
	trimmed := strings.TrimSpace(input)

	// Check if empty
	if trimmed == "" {
		return ValidationResult{Valid: false, Error: "draft name cannot be empty"}
	}

	// Check length
	if len(trimmed) > 100 {
		return ValidationResult{Valid: false, Error: "draft name must be no more than 100 characters"}
	}

	// Check for potentially dangerous characters
	if strings.Contains(trimmed, "<") || strings.Contains(trimmed, ">") {
		return ValidationResult{Valid: false, Error: "draft name contains invalid characters"}
	}

	// HTML escape for safety
	sanitized := html.EscapeString(trimmed)

	return ValidationResult{Valid: true, SanitizedValue: sanitized}
}

// ValidateInterval validates and sanitizes a draft interval
func ValidateInterval(input string) ValidationResult {
	// Trim whitespace
	trimmed := strings.TrimSpace(input)

	// Check if empty
	if trimmed == "" {
		return ValidationResult{Valid: false, Error: "interval cannot be empty"}
	}

	// Parse as integer
	interval, err := strconv.Atoi(trimmed)
	if err != nil {
		return ValidationResult{Valid: false, Error: "interval must be numeric"}
	}

	// Check bounds (reasonable limits for draft intervals in seconds)
	if interval < 1 {
		return ValidationResult{Valid: false, Error: "interval must be at least 1 second"}
	}
	if interval > 3600 { // 1 hour max
		return ValidationResult{Valid: false, Error: "interval must be no more than 3600 seconds (1 hour)"}
	}

	return ValidationResult{Valid: true, SanitizedValue: interval}
}

// ValidateUUID validates and sanitizes a UUID string
func ValidateUUID(input string) ValidationResult {
	// Trim whitespace
	trimmed := strings.TrimSpace(input)

	// Check if empty
	if trimmed == "" {
		return ValidationResult{Valid: false, Error: "UUID cannot be empty"}
	}

	// Parse UUID
	parsed, err := uuid.Parse(trimmed)
	if err != nil {
		return ValidationResult{Valid: false, Error: "invalid UUID format"}
	}

	return ValidationResult{Valid: true, SanitizedValue: parsed}
}

// SanitizeInput provides general input sanitization
func SanitizeInput(input string) string {
	// Trim whitespace
	trimmed := strings.TrimSpace(input)

	// HTML escape
	return html.EscapeString(trimmed)
}

// ValidateTeamKey validates FRC team keys (format: frcXXXX)
func ValidateTeamKey(input string) ValidationResult {
	// Trim whitespace
	trimmed := strings.TrimSpace(input)

	// Check if empty
	if trimmed == "" {
		return ValidationResult{Valid: false, Error: "team key cannot be empty"}
	}

	// Check format (frc followed by 1-4 digits)
	validPattern := regexp.MustCompile(`^frc\d{1,4}$`)
	if !validPattern.MatchString(trimmed) {
		return ValidationResult{Valid: false, Error: "team key must be in format frcXXXX (e.g., frc254)"}
	}

	return ValidationResult{Valid: true, SanitizedValue: trimmed}
}
