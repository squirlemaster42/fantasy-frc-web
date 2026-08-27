package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPickWindowConfig(t *testing.T) {
	cfg := DefaultPickWindowConfig()
	assert.Equal(t, 1*time.Hour, cfg.PickTime)
	assert.Equal(t, TimeRange{StartHour: 8, EndHour: 22}, cfg.Windows[time.Sunday])
	assert.Equal(t, TimeRange{StartHour: 17, EndHour: 22}, cfg.Windows[time.Monday])
}

func TestLoadPickWindowConfigFromFile_Defaults(t *testing.T) {
	path := filepath.Join("..", "..", "config", "pick-windows.json")
	cfg, err := LoadPickWindowConfigFromFile(path)
	require.NoError(t, err)

	// Read the raw config so the test stays valid when pick_time changes.
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var rawCfg struct {
		PickTime string                `json:"pick_time"`
		Windows  map[string]TimeRange  `json:"windows"`
	}
	require.NoError(t, json.Unmarshal(raw, &rawCfg))

	expectedPickTime, err := time.ParseDuration(rawCfg.PickTime)
	require.NoError(t, err)

	assert.Equal(t, expectedPickTime, cfg.PickTime)
	assert.Equal(t, TimeRange{StartHour: 8, EndHour: 22}, cfg.Windows[time.Sunday])
	assert.Equal(t, TimeRange{StartHour: 17, EndHour: 22}, cfg.Windows[time.Monday])
}

func TestLoadPickWindowConfigFromFile_CustomValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pick-windows.json")
	content := `{
		"pick_time": "30m",
		"windows": {
			"Sunday":    {"start_hour": 9, "end_hour": 21},
			"Monday":    {"start_hour": 18, "end_hour": 23},
			"Tuesday":   {"start_hour": 18, "end_hour": 23},
			"Wednesday": {"start_hour": 18, "end_hour": 23},
			"Thursday":  {"start_hour": 18, "end_hour": 23},
			"Friday":    {"start_hour": 18, "end_hour": 23},
			"Saturday":  {"start_hour": 9, "end_hour": 21}
		}
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	cfg, err := LoadPickWindowConfigFromFile(path)
	require.NoError(t, err)

	assert.Equal(t, 30*time.Minute, cfg.PickTime)
	assert.Equal(t, TimeRange{StartHour: 9, EndHour: 21}, cfg.Windows[time.Sunday])
	assert.Equal(t, TimeRange{StartHour: 18, EndHour: 23}, cfg.Windows[time.Monday])
}

func TestLoadPickWindowConfigFromEnv_MissingFileUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PICK_WINDOWS_CONFIG_FILE", filepath.Join(dir, "does-not-exist.json"))

	cfg, err := LoadPickWindowConfigFromEnv()
	require.NoError(t, err)

	defaultCfg := DefaultPickWindowConfig()
	assert.Equal(t, defaultCfg.PickTime, cfg.PickTime)
	assert.Equal(t, defaultCfg.Windows, cfg.Windows)
}

func TestLoadPickWindowConfigFromEnv_UsesEnvPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pick-windows.json")
	content := `{
		"pick_time": "45m",
		"windows": {
			"Sunday":    {"start_hour": 10, "end_hour": 20},
			"Monday":    {"start_hour": 12, "end_hour": 20},
			"Tuesday":   {"start_hour": 12, "end_hour": 20},
			"Wednesday": {"start_hour": 12, "end_hour": 20},
			"Thursday":  {"start_hour": 12, "end_hour": 20},
			"Friday":    {"start_hour": 12, "end_hour": 20},
			"Saturday":  {"start_hour": 10, "end_hour": 20}
		}
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	t.Setenv("PICK_WINDOWS_CONFIG_FILE", path)

	cfg, err := LoadPickWindowConfigFromEnv()
	require.NoError(t, err)

	assert.Equal(t, 45*time.Minute, cfg.PickTime)
	assert.Equal(t, TimeRange{StartHour: 10, EndHour: 20}, cfg.Windows[time.Saturday])
}

func TestLoadPickWindowConfigFromFile_InvalidPickTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pick-windows.json")
	content := `{"pick_time": "not-a-duration", "windows": {}}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := LoadPickWindowConfigFromFile(path)
	assert.Error(t, err)
}

func TestLoadPickWindowConfigFromFile_InvalidWeekday(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pick-windows.json")
	content := `{
		"pick_time": "1h",
		"windows": {
			"Someday": {"start_hour": 8, "end_hour": 22}
		}
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := LoadPickWindowConfigFromFile(path)
	assert.Error(t, err)
}

func TestLoadPickWindowConfigFromFile_InvalidHourRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pick-windows.json")
	content := `{
		"pick_time": "1h",
		"windows": {
			"Sunday": {"start_hour": 22, "end_hour": 8}
		}
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := LoadPickWindowConfigFromFile(path)
	assert.Error(t, err)
}

func TestLoadPickWindowConfigFromFile_MissingWeekday(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pick-windows.json")
	content := `{
		"pick_time": "1h",
		"windows": {
			"Sunday": {"start_hour": 8, "end_hour": 22}
		}
	}`
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	_, err := LoadPickWindowConfigFromFile(path)
	assert.Error(t, err)
}

func TestPickWindowConfig_GetPickExpirationTime(t *testing.T) {
	cfg := PickWindowConfig{
		PickTime: 1 * time.Hour,
		Windows: map[time.Weekday]TimeRange{
			time.Monday: {StartHour: 17, EndHour: 22},
		},
	}

	assert.Equal(t,
		time.Date(2025, time.April, 7, 18, 0, 0, 0, EasternLocation),
		cfg.GetPickExpirationTime(t.Context(), time.Date(2025, time.April, 7, 17, 0, 0, 0, EasternLocation), 1*time.Hour),
	)
}
