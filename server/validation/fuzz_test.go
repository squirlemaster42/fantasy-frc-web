package validation

import (
	"strings"
	"testing"
)

// FuzzValidateDraftID tests the ValidateDraftID function with random inputs
func FuzzValidateDraftID(f *testing.F) {
	// Add seed corpus with known edge cases
	seeds := []string{
		"123",
		"",
		"abc",
		"-1",
		"0",
		"999999999",
		" 123 ",
		"123.45",
		"999999",
		"000123",
		"123abc",
		"999999999999999999999", // Very large number
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		result := ValidateDraftID(input)

		// Function should never panic
		// Result should always be a valid ValidationResult struct
		if result.Valid {
			// If valid, SanitizedValue should be an int
			if _, ok := result.SanitizedValue.(int); !ok {
				t.Errorf("Valid result should contain int, got %T", result.SanitizedValue)
			}
			// Error should be empty
			if result.Error != "" {
				t.Errorf("Valid result should have empty error, got: %s", result.Error)
			}
		} else {
			// If invalid, SanitizedValue should be nil
			if result.SanitizedValue != nil {
				t.Errorf("Invalid result should have nil SanitizedValue, got: %v", result.SanitizedValue)
			}
			// Error should not be empty
			if result.Error == "" {
				t.Errorf("Invalid result should have non-empty error")
			}
		}
	})
}

// FuzzValidateUsername tests the ValidateUsername function with random inputs
func FuzzValidateUsername(f *testing.F) {
	seeds := []string{
		"testuser123",
		"ab",
		strings.Repeat("a", 31),
		"user name",
		"user@domain",
		"",
		"user_name",
		"UserName123",
		"  username  ",
		"user<script>",
		strings.Repeat("a", 50), // Very long
		"用户",                    // Unicode
		"user\nname",            // Newline
		"user\tname",            // Tab
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		result := ValidateUsername(input)

		// Function should never panic
		if result.Valid {
			// If valid, SanitizedValue should be a string
			if _, ok := result.SanitizedValue.(string); !ok {
				t.Errorf("Valid result should contain string, got %T", result.SanitizedValue)
			}
			// Error should be empty
			if result.Error != "" {
				t.Errorf("Valid result should have empty error, got: %s", result.Error)
			}
		} else {
			// If invalid, SanitizedValue should be nil
			if result.SanitizedValue != nil {
				t.Errorf("Invalid result should have nil SanitizedValue, got: %v", result.SanitizedValue)
			}
			// Error should not be empty
			if result.Error == "" {
				t.Errorf("Invalid result should have non-empty error")
			}
		}
	})
}

// FuzzValidatePassword tests the ValidatePassword function with random inputs
func FuzzValidatePassword(f *testing.F) {
	seeds := []string{
		"MySecurePass123!",
		"12345",
		strings.Repeat("a", 129),
		"mypassword123!",
		"MYPASSWORD123!",
		"MyPassword!",
		"MyPassword123",
		"password123!",
		"qwerty123!",
		"Tr!ckyP@ssw0rd2024",
		strings.Repeat("A", 200), // Very long
		"",                       // Empty
		"short",
		"NoDigits!",
		"nodigits",
		"NODIGITS123",
		"NoSpecialChars123",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		result := ValidatePassword(input)

		// Function should never panic
		if result.Valid {
			// If valid, SanitizedValue should be a string
			if _, ok := result.SanitizedValue.(string); !ok {
				t.Errorf("Valid result should contain string, got %T", result.SanitizedValue)
			}
			// Error should be empty
			if result.Error != "" {
				t.Errorf("Valid result should have empty error, got: %s", result.Error)
			}
		} else {
			// If invalid, SanitizedValue should be nil
			if result.SanitizedValue != nil {
				t.Errorf("Invalid result should have nil SanitizedValue, got: %v", result.SanitizedValue)
			}
			// Error should not be empty
			if result.Error == "" {
				t.Errorf("Invalid result should have non-empty error")
			}
		}
	})
}

// FuzzValidateDraftName tests the ValidateDraftName function with random inputs
func FuzzValidateDraftName(f *testing.F) {
	seeds := []string{
		"My Awesome Draft",
		"",
		strings.Repeat("a", 101),
		"Draft #1 - Regionals",
		"<script>alert('xss')</script>",
		"<img src=x onerror=alert('xss')>",
		"Tom & Jerry",
		`"Special" Draft`,
		"  Draft Name  ",
		strings.Repeat("a", 200),  // Very long
		"Draft\nName",             // Newline
		"Draft\tName",             // Tab
		"Draft" + string(rune(0)), // Null byte
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		result := ValidateDraftName(input)

		// Function should never panic
		if result.Valid {
			// If valid, SanitizedValue should be a string
			if _, ok := result.SanitizedValue.(string); !ok {
				t.Errorf("Valid result should contain string, got %T", result.SanitizedValue)
			}
			// Error should be empty
			if result.Error != "" {
				t.Errorf("Valid result should have empty error, got: %s", result.Error)
			}
		} else {
			// If invalid, SanitizedValue should be nil
			if result.SanitizedValue != nil {
				t.Errorf("Invalid result should have nil SanitizedValue, got: %v", result.SanitizedValue)
			}
			// Error should not be empty
			if result.Error == "" {
				t.Errorf("Invalid result should have non-empty error")
			}
		}
	})
}

// FuzzValidateInterval tests the ValidateInterval function with random inputs
func FuzzValidateInterval(f *testing.F) {
	seeds := []string{
		"30",
		"",
		"abc",
		"0",
		"-1",
		"3601",
		"3600",
		"1",
		" 60 ",
		"999999",
		"123.45",
		"00060",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		result := ValidateInterval(input)

		// Function should never panic
		if result.Valid {
			// If valid, SanitizedValue should be an int
			if _, ok := result.SanitizedValue.(int); !ok {
				t.Errorf("Valid result should contain int, got %T", result.SanitizedValue)
			}
			// Error should be empty
			if result.Error != "" {
				t.Errorf("Valid result should have empty error, got: %s", result.Error)
			}
		} else {
			// If invalid, SanitizedValue should be nil
			if result.SanitizedValue != nil {
				t.Errorf("Invalid result should have nil SanitizedValue, got: %v", result.SanitizedValue)
			}
			// Error should not be empty
			if result.Error == "" {
				t.Errorf("Invalid result should have non-empty error")
			}
		}
	})
}

// FuzzValidateUUID tests the ValidateUUID function with random inputs
func FuzzValidateUUID(f *testing.F) {
	seeds := []string{
		"550e8400-e29b-41d4-a716-446655440000",
		"",
		"not-a-uuid",
		"550e8400-e29b-41d4-a716",
		" 550e8400-e29b-41d4-a716-446655440000 ",
		strings.Repeat("a", 50),
		"550e8400-e29b-41d4-a716-446655440000-extra",
		"550e8400e29b41d4a716446655440000", // No dashes
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		result := ValidateUUID(input)

		// Function should never panic
		if result.Valid {
			// If valid, SanitizedValue should be a UUID
			// We can't easily check the type without importing uuid, so just ensure it's not nil
			if result.SanitizedValue == nil {
				t.Errorf("Valid result should have non-nil SanitizedValue")
			}
			// Error should be empty
			if result.Error != "" {
				t.Errorf("Valid result should have empty error, got: %s", result.Error)
			}
		} else {
			// If invalid, SanitizedValue should be nil
			if result.SanitizedValue != nil {
				t.Errorf("Invalid result should have nil SanitizedValue, got: %v", result.SanitizedValue)
			}
			// Error should not be empty
			if result.Error == "" {
				t.Errorf("Invalid result should have non-empty error")
			}
		}
	})
}

// FuzzValidateTeamKey tests the ValidateTeamKey function with random inputs
func FuzzValidateTeamKey(f *testing.F) {
	seeds := []string{
		"frc254",
		"frc1",
		"",
		"254",
		"ftc254",
		"frc12345",
		" frc254 ",
		"FRC254",
		strings.Repeat("f", 10) + "254",
		"frc" + strings.Repeat("1", 10),
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		result := ValidateTeamKey(input)

		// Function should never panic
		if result.Valid {
			// If valid, SanitizedValue should be a string
			if _, ok := result.SanitizedValue.(string); !ok {
				t.Errorf("Valid result should contain string, got %T", result.SanitizedValue)
			}
			// Error should be empty
			if result.Error != "" {
				t.Errorf("Valid result should have empty error, got: %s", result.Error)
			}
		} else {
			// If invalid, SanitizedValue should be nil
			if result.SanitizedValue != nil {
				t.Errorf("Invalid result should have nil SanitizedValue, got: %v", result.SanitizedValue)
			}
			// Error should not be empty
			if result.Error == "" {
				t.Errorf("Invalid result should have non-empty error")
			}
		}
	})
}

// FuzzSanitizeInput tests the SanitizeInput function with random inputs
func FuzzSanitizeInput(f *testing.F) {
	seeds := []string{
		"Hello World",
		"Tom & Jerry",
		`"Hello"`,
		"<script>",
		"  text  ",
		"",
		"   ",
		strings.Repeat("<", 100) + "script" + strings.Repeat(">", 100),
		"Normal text with <b>bold</b> tags",
		"Text with & < > \" ' symbols",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		result := SanitizeInput(input)

		// Function should never panic
		// Result should always be a string
		if result == "" && input != "" && len(strings.TrimSpace(input)) > 0 {
			// If input has non-whitespace content, result should not be empty
			// (unless input was only whitespace)
			t.Errorf("SanitizeInput should preserve non-whitespace content, input: %q, result: %q", input, result)
		}

		// Result should not contain unescaped < or > if input did
		if strings.Contains(input, "<") && !strings.Contains(result, "&lt;") && strings.Contains(result, "<") {
			t.Errorf("SanitizeInput should escape < characters, input: %q, result: %q", input, result)
		}
		if strings.Contains(input, ">") && !strings.Contains(result, "&gt;") && strings.Contains(result, ">") {
			t.Errorf("SanitizeInput should escape > characters, input: %q, result: %q", input, result)
		}
	})
}
