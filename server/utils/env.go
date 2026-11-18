package utils

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RequireEnv returns the value of the environment variable with the given key.
// If the variable is unset or empty, it returns an error.
func RequireEnv(key string) (string, error) {
	val := os.Getenv(key)
	if val == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return val, nil
}

// MustGetEnvString returns the environment variable value, or defaultVal if it
// is unset or empty.
func MustGetEnvString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

// LoadEnvFile loads environment variables from a .env-style file. It supports
// KEY=VALUE assignments, blank lines, and #-prefixed comments. Values may be
// quoted with single or double quotes; surrounding whitespace is trimmed. If
// the file does not exist, LoadEnvFile returns nil so callers can fall back to
// process environment variables.
func LoadEnvFile(path string) (retErr error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to open env file %s: %w", path, err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && retErr == nil {
			retErr = fmt.Errorf("failed to close env file %s: %w", path, closeErr)
		}
	}()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("invalid env file line %d: %q", lineNum, line)
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = unquoteEnvValue(value)

		if key == "" {
			return fmt.Errorf("empty key on env file line %d", lineNum)
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read env file %s: %w", path, err)
	}
	return nil
}

func unquoteEnvValue(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}
