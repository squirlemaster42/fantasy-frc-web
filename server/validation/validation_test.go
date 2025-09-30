package validation

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateDraftID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ValidationResult
	}{
		{
			name:  "valid draft id",
			input: "123",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: 123,
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: ValidationResult{
				Valid: false,
				Error: "draft ID cannot be empty",
			},
		},
		{
			name:  "whitespace only",
			input: "   ",
			expected: ValidationResult{
				Valid: false,
				Error: "draft ID cannot be empty",
			},
		},
		{
			name:  "non-numeric",
			input: "abc",
			expected: ValidationResult{
				Valid: false,
				Error: "draft ID must be numeric",
			},
		},
		{
			name:  "negative number",
			input: "-1",
			expected: ValidationResult{
				Valid: false,
				Error: "draft ID must be positive",
			},
		},
		{
			name:  "zero",
			input: "0",
			expected: ValidationResult{
				Valid: false,
				Error: "draft ID must be positive",
			},
		},
		{
			name:  "too large",
			input: "999999999",
			expected: ValidationResult{
				Valid: false,
				Error: "draft ID too large",
			},
		},
		{
			name:  "with whitespace",
			input: " 123 ",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: 123,
			},
		},
		{
			name:  "decimal number",
			input: "123.45",
			expected: ValidationResult{
				Valid: false,
				Error: "draft ID must be numeric",
			},
		},
		{
			name:  "maximum valid",
			input: "999999",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: 999999,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateDraftID(tt.input)
			assert.Equal(t, tt.expected.Valid, result.Valid)
			if tt.expected.Valid {
				assert.Equal(t, tt.expected.SanitizedValue, result.SanitizedValue)
				assert.Empty(t, result.Error)
			} else {
				assert.Contains(t, result.Error, tt.expected.Error)
				assert.Nil(t, result.SanitizedValue)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ValidationResult
	}{
		{
			name:  "valid username",
			input: "testuser123",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "testuser123",
			},
		},
		{
			name:  "too short",
			input: "ab",
			expected: ValidationResult{
				Valid: false,
				Error: "username must be at least 3 characters",
			},
		},
		{
			name:  "too long",
			input: strings.Repeat("a", 31),
			expected: ValidationResult{
				Valid: false,
				Error: "username must be no more than 30 characters",
			},
		},
		{
			name:  "invalid chars with spaces",
			input: "user name",
			expected: ValidationResult{
				Valid: false,
				Error: "username can only contain letters, numbers, and underscores",
			},
		},
		{
			name:  "invalid chars with special",
			input: "user@domain",
			expected: ValidationResult{
				Valid: false,
				Error: "username can only contain letters, numbers, and underscores",
			},
		},
		{
			name:  "empty",
			input: "",
			expected: ValidationResult{
				Valid: false,
				Error: "username must be at least 3 characters",
			},
		},
		{
			name:  "valid with underscore",
			input: "user_name",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "user_name",
			},
		},
		{
			name:  "valid mixed case",
			input: "UserName123",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "UserName123",
			},
		},
		{
			name:  "with leading/trailing spaces",
			input: "  username  ",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "username",
			},
		},
		{
			name:  "html characters",
			input: "user<script>",
			expected: ValidationResult{
				Valid: false,
				Error: "username can only contain letters, numbers, and underscores",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateUsername(tt.input)
			assert.Equal(t, tt.expected.Valid, result.Valid)
			if tt.expected.Valid {
				assert.Equal(t, tt.expected.SanitizedValue, result.SanitizedValue)
				assert.Empty(t, result.Error)
			} else {
				assert.Contains(t, result.Error, tt.expected.Error)
				assert.Nil(t, result.SanitizedValue)
			}
		})
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ValidationResult
	}{
		{
			name:  "strong password",
			input: "MySecurePass123!",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "MySecurePass123!",
			},
		},
		{
			name:  "too short",
			input: "12345",
			expected: ValidationResult{
				Valid: false,
				Error: "password must be at least 8 characters",
			},
		},
		{
			name:  "too long",
			input: strings.Repeat("a", 129),
			expected: ValidationResult{
				Valid: false,
				Error: "password must be no more than 128 characters",
			},
		},
		{
			name:  "no uppercase",
			input: "mysecret123!",
			expected: ValidationResult{
				Valid: false,
				Error: "password must contain at least one uppercase letter",
			},
		},
		{
			name:  "no lowercase",
			input: "MYSECRET123!",
			expected: ValidationResult{
				Valid: false,
				Error: "password must contain at least one lowercase letter",
			},
		},
		{
			name:  "no numbers",
			input: "MySecret!",
			expected: ValidationResult{
				Valid: false,
				Error: "password must contain at least one number",
			},
		},
		{
			name:  "no special chars",
			input: "MySecret123",
			expected: ValidationResult{
				Valid: false,
				Error: "password must contain at least one special character",
			},
		},
		{
			name:  "common password",
			input: "password123!",
			expected: ValidationResult{
				Valid: false,
				Error: "password is too common",
			},
		},
		{
			name:  "another common password",
			input: "qwerty123!",
			expected: ValidationResult{
				Valid: false,
				Error: "password is too common",
			},
		},
		{
			name:  "valid complex password",
			input: "Tr!ckyP@ssw0rd2024",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "Tr!ckyP@ssw0rd2024",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidatePassword(tt.input)
			assert.Equal(t, tt.expected.Valid, result.Valid)
			if tt.expected.Valid {
				assert.Equal(t, tt.expected.SanitizedValue, result.SanitizedValue)
				assert.Empty(t, result.Error)
			} else {
				assert.Contains(t, result.Error, tt.expected.Error)
				assert.Nil(t, result.SanitizedValue)
			}
		})
	}
}

func TestValidateDraftName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ValidationResult
	}{
		{
			name:  "valid name",
			input: "My Awesome Draft",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "My Awesome Draft",
			},
		},
		{
			name:  "empty",
			input: "",
			expected: ValidationResult{
				Valid: false,
				Error: "draft name cannot be empty",
			},
		},
		{
			name:  "too long",
			input: strings.Repeat("a", 101),
			expected: ValidationResult{
				Valid: false,
				Error: "draft name must be no more than 100 characters",
			},
		},
		{
			name:  "valid special chars",
			input: "Draft #1 - Regionals",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "Draft #1 - Regionals",
			},
		},
		{
			name:  "xss attempt script",
			input: "<script>alert('xss')</script>",
			expected: ValidationResult{
				Valid: false,
				Error: "draft name contains invalid characters",
			},
		},
		{
			name:  "xss attempt img",
			input: "<img src=x onerror=alert('xss')>",
			expected: ValidationResult{
				Valid: false,
				Error: "draft name contains invalid characters",
			},
		},
		{
			name:  "with html entities",
			input: "Tom & Jerry",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "Tom &amp; Jerry",
			},
		},
		{
			name:  "with quotes",
			input: `"Special" Draft`,
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "&#34;Special&#34; Draft",
			},
		},
		{
			name:  "with whitespace",
			input: "  Draft Name  ",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "Draft Name",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateDraftName(tt.input)
			assert.Equal(t, tt.expected.Valid, result.Valid)
			if tt.expected.Valid {
				assert.Equal(t, tt.expected.SanitizedValue, result.SanitizedValue)
				assert.Empty(t, result.Error)
			} else {
				assert.Contains(t, result.Error, tt.expected.Error)
				assert.Nil(t, result.SanitizedValue)
			}
		})
	}
}

func TestValidateInterval(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ValidationResult
	}{
		{
			name:  "valid interval",
			input: "30",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: 30,
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: ValidationResult{
				Valid: false,
				Error: "interval cannot be empty",
			},
		},
		{
			name:  "non-numeric",
			input: "abc",
			expected: ValidationResult{
				Valid: false,
				Error: "interval must be numeric",
			},
		},
		{
			name:  "zero",
			input: "0",
			expected: ValidationResult{
				Valid: false,
				Error: "interval must be at least 1 second",
			},
		},
		{
			name:  "negative",
			input: "-1",
			expected: ValidationResult{
				Valid: false,
				Error: "interval must be at least 1 second",
			},
		},
		{
			name:  "too large",
			input: "3601",
			expected: ValidationResult{
				Valid: false,
				Error: "interval must be no more than 3600 seconds",
			},
		},
		{
			name:  "maximum valid",
			input: "3600",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: 3600,
			},
		},
		{
			name:  "minimum valid",
			input: "1",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: 1,
			},
		},
		{
			name:  "with whitespace",
			input: " 60 ",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: 60,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateInterval(tt.input)
			assert.Equal(t, tt.expected.Valid, result.Valid)
			if tt.expected.Valid {
				assert.Equal(t, tt.expected.SanitizedValue, result.SanitizedValue)
				assert.Empty(t, result.Error)
			} else {
				assert.Contains(t, result.Error, tt.expected.Error)
				assert.Nil(t, result.SanitizedValue)
			}
		})
	}
}

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ValidationResult
	}{
		{
			name:  "valid UUID",
			input: "550e8400-e29b-41d4-a716-446655440000",
			expected: ValidationResult{
				Valid: true,
				// SanitizedValue will be the parsed UUID
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: ValidationResult{
				Valid: false,
				Error: "UUID cannot be empty",
			},
		},
		{
			name:  "invalid format",
			input: "not-a-uuid",
			expected: ValidationResult{
				Valid: false,
				Error: "invalid UUID format",
			},
		},
		{
			name:  "invalid length",
			input: "550e8400-e29b-41d4-a716",
			expected: ValidationResult{
				Valid: false,
				Error: "invalid UUID format",
			},
		},
		{
			name:  "with whitespace",
			input: " 550e8400-e29b-41d4-a716-446655440000 ",
			expected: ValidationResult{
				Valid: true,
				// SanitizedValue will be the parsed UUID
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateUUID(tt.input)
			assert.Equal(t, tt.expected.Valid, result.Valid)
			if tt.expected.Valid {
				assert.NotNil(t, result.SanitizedValue)
				assert.Empty(t, result.Error)
			} else {
				assert.Contains(t, result.Error, tt.expected.Error)
				assert.Nil(t, result.SanitizedValue)
			}
		})
	}
}

func TestValidateTeamKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ValidationResult
	}{
		{
			name:  "valid team key",
			input: "frc254",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "frc254",
			},
		},
		{
			name:  "valid team key single digit",
			input: "frc1",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "frc1",
			},
		},
		{
			name:  "empty string",
			input: "",
			expected: ValidationResult{
				Valid: false,
				Error: "team key cannot be empty",
			},
		},
		{
			name:  "missing frc prefix",
			input: "254",
			expected: ValidationResult{
				Valid: false,
				Error: "team key must be in format frcXXXX",
			},
		},
		{
			name:  "wrong prefix",
			input: "ftc254",
			expected: ValidationResult{
				Valid: false,
				Error: "team key must be in format frcXXXX",
			},
		},
		{
			name:  "too many digits",
			input: "frc12345",
			expected: ValidationResult{
				Valid: false,
				Error: "team key must be in format frcXXXX",
			},
		},
		{
			name:  "with whitespace",
			input: " frc254 ",
			expected: ValidationResult{
				Valid:          true,
				SanitizedValue: "frc254",
			},
		},
		{
			name:  "uppercase",
			input: "FRC254",
			expected: ValidationResult{
				Valid: false,
				Error: "team key must be in format frcXXXX",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateTeamKey(tt.input)
			assert.Equal(t, tt.expected.Valid, result.Valid)
			if tt.expected.Valid {
				assert.Equal(t, tt.expected.SanitizedValue, result.SanitizedValue)
				assert.Empty(t, result.Error)
			} else {
				assert.Contains(t, result.Error, tt.expected.Error)
				assert.Nil(t, result.SanitizedValue)
			}
		})
	}
}

func TestSanitizeInput(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "Hello World",
			expected: "Hello World",
		},
		{
			name:     "html entities",
			input:    "Tom & Jerry",
			expected: "Tom &amp; Jerry",
		},
		{
			name:     "quotes",
			input:    `"Hello"`,
			expected: "&#34;Hello&#34;",
		},
		{
			name:     "script tag",
			input:    "<script>",
			expected: "&lt;script&gt;",
		},
		{
			name:     "with whitespace",
			input:    "  text  ",
			expected: "text",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeInput(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
