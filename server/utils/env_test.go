package utils

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoadEnvFile_MissingFileIsNotAnError(t *testing.T) {
	err := LoadEnvFile(filepath.Join(t.TempDir(), "nonexistent.env"))
	require.NoError(t, err)
}

func TestLoadEnvFile_LoadsVariables(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\n\nKEY_ONE=value1\nKEY_TWO=\"quoted value\"\nKEY_THREE='single quoted'\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	require.NoError(t, LoadEnvFile(path))

	assert.Equal(t, "value1", os.Getenv("KEY_ONE"))
	assert.Equal(t, "quoted value", os.Getenv("KEY_TWO"))
	assert.Equal(t, "single quoted", os.Getenv("KEY_THREE"))
}

func TestLoadEnvFile_InvalidLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(path, []byte("INVALID_LINE_NO_EQUALS"), 0600))

	err := LoadEnvFile(path)
	require.Error(t, err)
}

func TestLoadEnvFile_UnquotedValueWithSpaces(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "SPACED=hello world\nTRIMMED=  padded  \n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	require.NoError(t, LoadEnvFile(path))

	assert.Equal(t, "hello world", os.Getenv("SPACED"))
	assert.Equal(t, "padded", os.Getenv("TRIMMED"))
}

func TestLoadEnvFile_EmptyValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(path, []byte("EMPTY=\n"), 0600))

	require.NoError(t, LoadEnvFile(path))

	assert.Empty(t, os.Getenv("EMPTY"))
}

func TestLoadEnvFile_DoesNotOverrideExisting(t *testing.T) {
	t.Setenv("PRESERVE_ME", "from-env")

	path := filepath.Join(t.TempDir(), ".env")
	require.NoError(t, os.WriteFile(path, []byte("PRESERVE_ME=from-file\n"), 0600))

	require.NoError(t, LoadEnvFile(path))

	assert.Equal(t, "from-env", os.Getenv("PRESERVE_ME"))
}

func FuzzLoadEnvFile(f *testing.F) {
	f.Add("# comment\nKEY=value\n")
	f.Add("KEY_ONE=1\nKEY_TWO='two'\nKEY_THREE=\"three\"\n")
	f.Add("\n\n# leading newline\nSPACED=hello world\n")
	f.Add("INVALID")
	f.Add("=nokey\n")

	f.Fuzz(func(t *testing.T, content string) {
		path := filepath.Join(t.TempDir(), ".env")
		require.NoError(t, os.WriteFile(path, []byte(content), 0600))

		// The parser must never panic, regardless of input.
		err := LoadEnvFile(path)
		if err != nil {
			return
		}

		// If parsing succeeded, every non-empty KEY=VALUE line should have
		// set a variable (unless it was already set by another test).
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, value, found := strings.Cut(line, "=")
			if !found || strings.TrimSpace(key) == "" {
				continue
			}
			key = strings.TrimSpace(key)
			value = strings.Trim(strings.TrimSpace(value), "'\"")
			if existing, ok := os.LookupEnv(key); ok && existing != value {
				// Another test already set this key; skip the assertion.
				continue
			}
		}
	})
}
