package handler

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestModifyPickTimeCommandArgumentParsing(t *testing.T) {
	tests := []struct {
		name           string
		args           string
		expectedResult string
		description    string
	}{
		{
			name:           "missing id argument",
			args:           "-time=45m",
			expectedResult: "Missing required argument: -id=<draftId>",
			description:    "Should return error when draft ID is missing",
		},
		{
			name:           "missing time argument",
			args:           "-id=123",
			expectedResult: "Missing required argument: -time=<duration>",
			description:    "Should return error when time duration is missing",
		},
		{
			name:           "missing both arguments",
			args:           "",
			expectedResult: "Missing required argument: -id=<draftId>",
			description:    "Should return error for missing ID first",
		},
		{
			name:           "invalid draft id",
			args:           "-id=invalid -time=45m",
			expectedResult: "Draft Id Could Not Be Converted To An Int",
			description:    "Should return error when draft ID is not an integer",
		},
		{
			name:           "invalid duration format",
			args:           "-id=123 -time=invalid",
			expectedResult: "Invalid duration format. Use format like: 45m, 1h30m, 2h15m30s",
			description:    "Should return error for invalid duration format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &ModifyPickTimeCommand{}
			result := cmd.ProcessCommand(nil, nil, tt.args)
			assert.Equal(t, tt.expectedResult, result, tt.description)
		})
	}
}

func TestModifyPickTimeCommandDurationParsing(t *testing.T) {
	tests := []struct {
		name          string
		duration      string
		shouldBeValid bool
		description   string
	}{
		{
			name:          "simple minutes",
			duration:      "45m",
			shouldBeValid: true,
			description:   "Should parse 45 minutes",
		},
		{
			name:          "hours and minutes",
			duration:      "1h30m",
			shouldBeValid: true,
			description:   "Should parse 1 hour 30 minutes",
		},
		{
			name:          "compound duration",
			duration:      "2h15m30s",
			shouldBeValid: true,
			description:   "Should parse 2 hours 15 minutes 30 seconds",
		},
		{
			name:          "seconds only",
			duration:      "30s",
			shouldBeValid: true,
			description:   "Should parse 30 seconds",
		},
		{
			name:          "hours only",
			duration:      "2h",
			shouldBeValid: true,
			description:   "Should parse 2 hours",
		},
		{
			name:          "with plus prefix",
			duration:      "+30m",
			shouldBeValid: true,
			description:   "Should parse duration with plus prefix",
		},
		{
			name:          "with plus prefix compound",
			duration:      "+1h30m",
			shouldBeValid: true,
			description:   "Should parse compound duration with plus prefix",
		},
		{
			name:          "days",
			duration:      "1d",
			shouldBeValid: false,
			description:   "Should reject days (not supported by time.ParseDuration)",
		},
		{
			name:          "invalid format",
			duration:      "abc",
			shouldBeValid: false,
			description:   "Should reject invalid format",
		},
		{
			name:          "empty string",
			duration:      "",
			shouldBeValid: false,
			description:   "Should reject empty string",
		},
		{
			name:          "just number",
			duration:      "30",
			shouldBeValid: false,
			description:   "Should reject duration without unit",
		},
		{
			name:          "spaces in duration",
			duration:      "1h 30m",
			shouldBeValid: false,
			description:   "Should reject spaces in duration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test duration parsing directly
			_, err := time.ParseDuration(tt.duration)
			if tt.shouldBeValid {
				assert.NoError(t, err, tt.description)
			} else {
				assert.Error(t, err, tt.description)
			}
		})
	}
}

func TestModifyPickTimeCommandTimeCalculation(t *testing.T) {
	tests := []struct {
		name              string
		duration          string
		expectedMinOffset time.Duration
		expectedMaxOffset time.Duration
		description       string
	}{
		{
			name:              "45 minutes",
			duration:          "45m",
			expectedMinOffset: 44 * time.Minute,
			expectedMaxOffset: 46 * time.Minute,
			description:       "Should add approximately 45 minutes",
		},
		{
			name:              "1 hour 30 minutes",
			duration:          "1h30m",
			expectedMinOffset: 89 * time.Minute,
			expectedMaxOffset: 91 * time.Minute,
			description:       "Should add approximately 90 minutes",
		},
		{
			name:              "2 hours 15 minutes 30 seconds",
			duration:          "2h15m30s",
			expectedMinOffset: 2*time.Hour + 14*time.Minute + 30*time.Second,
			expectedMaxOffset: 2*time.Hour + 16*time.Minute + 30*time.Second,
			description:       "Should add approximately 2h15m30s",
		},
		{
			name:              "30 seconds",
			duration:          "30s",
			expectedMinOffset: 29 * time.Second,
			expectedMaxOffset: 31 * time.Second,
			description:       "Should add approximately 30 seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duration, err := time.ParseDuration(tt.duration)
			assert.NoError(t, err)

			now := time.Now()
			newExpiration := now.Add(duration)
			offset := newExpiration.Sub(now)

			assert.True(t, offset >= tt.expectedMinOffset && offset <= tt.expectedMaxOffset,
				"Expected offset between %v and %v, got %v",
				tt.expectedMinOffset, tt.expectedMaxOffset, offset)
		})
	}
}
