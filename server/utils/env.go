package utils

import (
	"fmt"
	"os"
	"strconv"
	"time"
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

// GetEnvBoolStrict returns the default value if the environment variable is unset.
// If the variable is set but cannot be parsed as a bool, it returns an error.
func GetEnvBoolStrict(key string, defaultVal bool) (bool, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}
	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		return false, fmt.Errorf("environment variable %s has invalid bool value %q: %w", key, val, err)
	}
	return boolVal, nil
}

// GetEnvIntStrict returns the default value if the environment variable is unset.
// If the variable is set but cannot be parsed as an int, it returns an error.
func GetEnvIntStrict(key string, defaultVal int) (int, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s has invalid int value %q: %w", key, val, err)
	}
	return intVal, nil
}

// GetEnvInt64Strict returns the default value if the environment variable is unset.
// If the variable is set but cannot be parsed as an int64, it returns an error.
func GetEnvInt64Strict(key string, defaultVal int64) (int64, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}
	intVal, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s has invalid int64 value %q: %w", key, val, err)
	}
	return intVal, nil
}

// MustGetEnvBool returns the default value if the environment variable is unset.
// It panics if the variable is set but cannot be parsed as a bool.
func MustGetEnvBool(key string, defaultVal bool) bool {
	val, err := GetEnvBoolStrict(key, defaultVal)
	if err != nil {
		panic(err)
	}
	return val
}

// MustGetEnvInt returns the default value if the environment variable is unset.
// It panics if the variable is set but cannot be parsed as an int.
func MustGetEnvInt(key string, defaultVal int) int {
	val, err := GetEnvIntStrict(key, defaultVal)
	if err != nil {
		panic(err)
	}
	return val
}

// MustGetEnvInt64 returns the default value if the environment variable is unset.
// It panics if the variable is set but cannot be parsed as an int64.
func MustGetEnvInt64(key string, defaultVal int64) int64 {
	val, err := GetEnvInt64Strict(key, defaultVal)
	if err != nil {
		panic(err)
	}
	return val
}

// MustGetEnvDuration returns the default value if the environment variable is unset.
// It panics if the variable is set but cannot be parsed as a duration.
func MustGetEnvDuration(key string, defaultVal time.Duration) time.Duration {
	val, err := GetEnvDurationStrict(key, defaultVal)
	if err != nil {
		panic(err)
	}
	return val
}

// MustGetEnvString returns the default value if the environment variable is unset.
// An empty value is treated as unset.
func MustGetEnvString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return defaultVal
}

// GetEnvDurationStrict returns the default value if the environment variable is unset.
// If the variable is set but cannot be parsed as a duration, it returns an error.
func GetEnvDurationStrict(key string, defaultVal time.Duration) (time.Duration, error) {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal, nil
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s has invalid duration value %q: %w", key, val, err)
	}
	return d, nil
}

func GetEnvInt(key string, defaultVal int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultVal
	}
	return intVal
}

func GetEnvInt64(key string, defaultVal int64) int64 {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	intVal, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return defaultVal
	}
	return intVal
}

func GetEnvBool(key string, defaultVal bool) bool {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		return defaultVal
	}
	return boolVal
}

func GetEnvDuration(key string, defaultVal time.Duration) time.Duration {
	val := os.Getenv(key)
	if val == "" {
		return defaultVal
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return defaultVal
	}
	return d
}
