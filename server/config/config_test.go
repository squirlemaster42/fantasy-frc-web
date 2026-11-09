package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Setenv("TEST_CONFIG_NAME", "app")
	t.Setenv("TEST_CONFIG_PORT", "8080")
	t.Setenv("TEST_CONFIG_DEBUG", "true")
	t.Setenv("TEST_CONFIG_TIMEOUT", "30s")
	t.Setenv("TEST_CONFIG_EVENTS", "arc,dal,gal")

	type cfg struct {
		Name    string        `env:"TEST_CONFIG_NAME" default:"fallback"`
		Port    int           `env:"TEST_CONFIG_PORT" default:"80"`
		Debug   bool          `env:"TEST_CONFIG_DEBUG" default:"false"`
		Timeout time.Duration `env:"TEST_CONFIG_TIMEOUT" default:"5s"`
		Events  []string      `env:"TEST_CONFIG_EVENTS"`
		Missing string        `env:"TEST_CONFIG_MISSING" default:"default-value"`
	}

	var c cfg
	require.NoError(t, Load(&c))

	assert.Equal(t, "app", c.Name)
	assert.Equal(t, 8080, c.Port)
	assert.True(t, c.Debug)
	assert.Equal(t, 30*time.Second, c.Timeout)
	assert.Equal(t, []string{"arc", "dal", "gal"}, c.Events)
	assert.Equal(t, "default-value", c.Missing)
}

func TestLoadRequiredMissing(t *testing.T) {
	type cfg struct {
		Required string `env:"TEST_CONFIG_REQUIRED_MISSING" required:"true"`
	}

	var c cfg
	err := Load(&c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required environment variable")
}

func TestLoadInvalidInt(t *testing.T) {
	t.Setenv("TEST_CONFIG_BAD_INT", "not-a-number")

	type cfg struct {
		Value int `env:"TEST_CONFIG_BAD_INT"`
	}

	var c cfg
	err := Load(&c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid value")
}

func TestLoadInvalidBool(t *testing.T) {
	t.Setenv("TEST_CONFIG_BAD_BOOL", "maybe")

	type cfg struct {
		Value bool `env:"TEST_CONFIG_BAD_BOOL"`
	}

	var c cfg
	err := Load(&c)
	require.Error(t, err)
}

func TestLoadInvalidDuration(t *testing.T) {
	t.Setenv("TEST_CONFIG_BAD_DURATION", "forever")

	type cfg struct {
		Value time.Duration `env:"TEST_CONFIG_BAD_DURATION"`
	}

	var c cfg
	err := Load(&c)
	require.Error(t, err)
}

func TestLoadNestedStruct(t *testing.T) {
	t.Setenv("TEST_CONFIG_NESTED_VALUE", "42")

	type nested struct {
		Value int `env:"TEST_CONFIG_NESTED_VALUE"`
	}
	type cfg struct {
		Inner nested
	}

	var c cfg
	require.NoError(t, Load(&c))
	assert.Equal(t, 42, c.Inner.Value)
}

func TestLoadNonPointer(t *testing.T) {
	type cfg struct{}
	var c cfg
	err := Load(c)
	require.Error(t, err)
}

func TestLoadSlice(t *testing.T) {
	t.Setenv("TEST_CONFIG_SLICE", "a, b,c")

	type cfg struct {
		Values []string `env:"TEST_CONFIG_SLICE"`
	}

	var c cfg
	require.NoError(t, Load(&c))
	assert.Equal(t, []string{"a", "b", "c"}, c.Values)
}

func TestLoadEmptyDefaultLeavesZeroValue(t *testing.T) {
	type cfg struct {
		Value int `env:"TEST_CONFIG_EMPTY_DEFAULT" default:""`
	}

	var c cfg
	require.NoError(t, Load(&c))
	assert.Equal(t, 0, c.Value)
}

func TestLoadRequiredEmptyString(t *testing.T) {
	t.Setenv("TEST_CONFIG_REQUIRED_EMPTY", "")

	type cfg struct {
		Required string `env:"TEST_CONFIG_REQUIRED_EMPTY" required:"true"`
	}

	var c cfg
	err := Load(&c)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required environment variable")
}
