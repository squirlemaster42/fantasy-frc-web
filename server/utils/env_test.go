package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRequireEnv_Set(t *testing.T) {
	t.Setenv("TEST_REQUIRE_ENV", "value")
	val, err := RequireEnv("TEST_REQUIRE_ENV")
	assert.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestRequireEnv_Unset(t *testing.T) {
	t.Setenv("TEST_REQUIRE_ENV_UNSET", "")
	val, err := RequireEnv("TEST_REQUIRE_ENV_UNSET")
	assert.Error(t, err)
	assert.Equal(t, "", val)
	assert.Contains(t, err.Error(), "required environment variable")
}

func TestGetEnvBoolStrict_Unset(t *testing.T) {
	t.Setenv("TEST_BOOL_UNSET", "")
	val, err := GetEnvBoolStrict("TEST_BOOL_UNSET", true)
	assert.NoError(t, err)
	assert.True(t, val)
}

func TestGetEnvBoolStrict_Valid(t *testing.T) {
	t.Setenv("TEST_BOOL_VALID", "false")
	val, err := GetEnvBoolStrict("TEST_BOOL_VALID", true)
	assert.NoError(t, err)
	assert.False(t, val)
}

func TestGetEnvBoolStrict_Invalid(t *testing.T) {
	t.Setenv("TEST_BOOL_INVALID", "not-a-bool")
	val, err := GetEnvBoolStrict("TEST_BOOL_INVALID", true)
	assert.Error(t, err)
	assert.False(t, val)
	assert.Contains(t, err.Error(), "invalid bool value")
}

func TestGetEnvIntStrict_Unset(t *testing.T) {
	t.Setenv("TEST_INT_UNSET", "")
	val, err := GetEnvIntStrict("TEST_INT_UNSET", 42)
	assert.NoError(t, err)
	assert.Equal(t, 42, val)
}

func TestGetEnvIntStrict_Valid(t *testing.T) {
	t.Setenv("TEST_INT_VALID", "7")
	val, err := GetEnvIntStrict("TEST_INT_VALID", 42)
	assert.NoError(t, err)
	assert.Equal(t, 7, val)
}

func TestGetEnvIntStrict_Invalid(t *testing.T) {
	t.Setenv("TEST_INT_INVALID", "abc")
	val, err := GetEnvIntStrict("TEST_INT_INVALID", 42)
	assert.Error(t, err)
	assert.Equal(t, 0, val)
	assert.Contains(t, err.Error(), "invalid int value")
}

func TestGetEnvInt64Strict_Valid(t *testing.T) {
	t.Setenv("TEST_INT64_VALID", "99")
	val, err := GetEnvInt64Strict("TEST_INT64_VALID", 42)
	assert.NoError(t, err)
	assert.Equal(t, int64(99), val)
}

func TestGetEnvInt64Strict_Invalid(t *testing.T) {
	t.Setenv("TEST_INT64_INVALID", "xyz")
	val, err := GetEnvInt64Strict("TEST_INT64_INVALID", 42)
	assert.Error(t, err)
	assert.Equal(t, int64(0), val)
}

func TestGetEnvDurationStrict_Valid(t *testing.T) {
	t.Setenv("TEST_DURATION_VALID", "5m")
	val, err := GetEnvDurationStrict("TEST_DURATION_VALID", time.Hour)
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Minute, val)
}

func TestGetEnvDurationStrict_Invalid(t *testing.T) {
	t.Setenv("TEST_DURATION_INVALID", "forever")
	val, err := GetEnvDurationStrict("TEST_DURATION_INVALID", time.Hour)
	assert.Error(t, err)
	assert.Equal(t, time.Duration(0), val)
}
