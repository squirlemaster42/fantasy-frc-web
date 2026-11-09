package utils

import (
	"fmt"
	"os"
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
