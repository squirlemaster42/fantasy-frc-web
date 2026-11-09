package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireEnv_Set(t *testing.T) {
	t.Setenv("TEST_REQUIRE_ENV", "value")
	val, err := RequireEnv("TEST_REQUIRE_ENV")
	require.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestRequireEnv_Unset(t *testing.T) {
	val, err := RequireEnv("TEST_REQUIRE_ENV_UNSET")
	require.Error(t, err)
	assert.Empty(t, val)
}

func TestMustGetEnvString_Set(t *testing.T) {
	t.Setenv("TEST_MUST_STRING", "override")
	val := MustGetEnvString("TEST_MUST_STRING", "default")
	assert.Equal(t, "override", val)
}

func TestMustGetEnvString_Unset(t *testing.T) {
	val := MustGetEnvString("TEST_MUST_STRING_UNSET", "default")
	assert.Equal(t, "default", val)
}

func TestMustGetEnvString_Empty(t *testing.T) {
	t.Setenv("TEST_MUST_STRING_EMPTY", "")
	val := MustGetEnvString("TEST_MUST_STRING_EMPTY", "default")
	assert.Equal(t, "default", val)
}
