package authentication

import (
	"fmt"
	"strings"
	"unicode"
)

// validateUsername trims whitespace and checks length and allowed characters.
// It returns the normalized username and an empty string if valid, otherwise an error message.
func validateUsername(username string, minLength, maxLength int, allowedSpecialChars string) (string, string) {
	normalized := strings.TrimSpace(username)
	if len(normalized) < len(username) {
		return "", "Username cannot contain leading or trailing spaces"
	}
	if strings.ContainsAny(normalized, " \t\n\r") {
		return "", "Username cannot contain spaces"
	}
	if len(normalized) < minLength {
		return "", fmt.Sprintf("Username must be at least %d characters", minLength)
	}
	if len(normalized) > maxLength {
		return "", fmt.Sprintf("Username must be at most %d characters", maxLength)
	}
	for _, ch := range normalized {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && !strings.ContainsRune(allowedSpecialChars, ch) {
			return "", fmt.Sprintf("Username can only contain letters, numbers, and %s", allowedSpecialChars)
		}
	}
	return normalized, ""
}

// validatePassword checks that password and confirmPassword match and that the password
// meets the configured complexity rules.
func validatePassword(password, confirmPassword string, minLength int) string {
	if password != confirmPassword {
		return "Passwords Do Not Match"
	}
	if len(password) < minLength {
		return fmt.Sprintf("Password must be at least %d characters", minLength)
	}
	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return "Password must contain at least one uppercase letter, one lowercase letter, and one digit"
	}
	return ""
}
