package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"server/log"
)

const defaultPickWindowsConfigFile = "../config/pick-windows.json"

// TimeRange defines an inclusive hour window used for pick scheduling.
type TimeRange struct {
	StartHour int `json:"start_hour"`
	EndHour   int `json:"end_hour"`
}

// PickWindowConfig holds the global pick scheduling configuration.
type PickWindowConfig struct {
	PickTime time.Duration
	Windows  map[time.Weekday]TimeRange
}

// DefaultPickWindowConfig returns the built-in pick scheduling defaults.
func DefaultPickWindowConfig() PickWindowConfig {
	return PickWindowConfig{
		PickTime: 1 * time.Hour,
		Windows: map[time.Weekday]TimeRange{
			time.Sunday:    {StartHour: 8, EndHour: 22},
			time.Monday:    {StartHour: 17, EndHour: 22},
			time.Tuesday:   {StartHour: 17, EndHour: 22},
			time.Wednesday: {StartHour: 17, EndHour: 22},
			time.Thursday:  {StartHour: 17, EndHour: 22},
			time.Friday:    {StartHour: 17, EndHour: 22},
			time.Saturday:  {StartHour: 8, EndHour: 22},
		},
	}
}

type pickWindowConfigJSON struct {
	PickTime string                    `json:"pick_time"`
	Windows  map[string]TimeRange      `json:"windows"`
}

// LoadPickWindowConfigFromEnv loads pick-window configuration from the file
// specified by PICK_WINDOWS_CONFIG_FILE, or from the default path. If the file
// is not present, the default configuration is returned.
func LoadPickWindowConfigFromEnv() (PickWindowConfig, error) {
	configFile := os.Getenv("PICK_WINDOWS_CONFIG_FILE")
	if configFile == "" {
		configFile = defaultPickWindowsConfigFile
	}

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Info(context.Background(), "Pick windows config file not found, using defaults", "path", configFile)
		return DefaultPickWindowConfig(), nil
	}

	return LoadPickWindowConfigFromFile(configFile)
}

// LoadPickWindowConfigFromFile reads and validates a PickWindowConfig from a JSON file.
func LoadPickWindowConfigFromFile(path string) (PickWindowConfig, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return PickWindowConfig{}, fmt.Errorf("failed to read pick windows config file %s: %w", path, err)
	}

	var raw pickWindowConfigJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return PickWindowConfig{}, fmt.Errorf("failed to parse pick windows config file %s: %w", path, err)
	}

	cfg := DefaultPickWindowConfig()

	if raw.PickTime != "" {
		d, err := time.ParseDuration(raw.PickTime)
		if err != nil {
			return PickWindowConfig{}, fmt.Errorf("invalid pick_time %q in config file %s: %w", raw.PickTime, path, err)
		}
		if d <= 0 {
			return PickWindowConfig{}, fmt.Errorf("pick_time must be positive in config file %s", path)
		}
		cfg.PickTime = d
	}

	if len(raw.Windows) > 0 {
		parsedWindows := make(map[time.Weekday]TimeRange, 7)
		for name, window := range raw.Windows {
			weekday, err := parseWeekday(name)
			if err != nil {
				return PickWindowConfig{}, fmt.Errorf("invalid window key %q in config file %s: %w", name, path, err)
			}
			if err := validateTimeRange(window); err != nil {
				return PickWindowConfig{}, fmt.Errorf("invalid window for %s in config file %s: %w", name, path, err)
			}
			parsedWindows[weekday] = window
		}
		if len(parsedWindows) != 7 {
			return PickWindowConfig{}, fmt.Errorf("pick windows config file %s must define all 7 weekdays", path)
		}
		cfg.Windows = parsedWindows
	}

	return cfg, nil
}

func parseWeekday(name string) (time.Weekday, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "sunday":
		return time.Sunday, nil
	case "monday":
		return time.Monday, nil
	case "tuesday":
		return time.Tuesday, nil
	case "wednesday":
		return time.Wednesday, nil
	case "thursday":
		return time.Thursday, nil
	case "friday":
		return time.Friday, nil
	case "saturday":
		return time.Saturday, nil
	default:
		return 0, fmt.Errorf("unknown weekday %q", name)
	}
}

func validateTimeRange(tr TimeRange) error {
	if tr.StartHour < 0 || tr.StartHour > 23 {
		return fmt.Errorf("start_hour must be between 0 and 23, got %d", tr.StartHour)
	}
	if tr.EndHour < 0 || tr.EndHour > 23 {
		return fmt.Errorf("end_hour must be between 0 and 23, got %d", tr.EndHour)
	}
	if tr.StartHour >= tr.EndHour {
		return fmt.Errorf("start_hour (%d) must be less than end_hour (%d)", tr.StartHour, tr.EndHour)
	}
	return nil
}

// GetPickExpirationTime calculates when a pick should expire given the current
// time and the configured pick windows. All scheduling is done in the
// hardcoded US/Eastern timezone.
func (cfg PickWindowConfig) GetPickExpirationTime(ctx context.Context, t time.Time, expirationDuration time.Duration) time.Time {
	t = t.In(EasternLocation)
	log.Info(ctx, "Getting Expiration Time", "Current Time", t)
	expirationTime := t.Add(expirationDuration)
	validTime := cfg.Windows[expirationTime.Weekday()]
	nextDay := t.Add(24 * time.Hour)

	// If the expiration time is in the pick window and we are currently in the pick window
	if expirationTime.Hour() >= validTime.StartHour && expirationTime.Hour() <= validTime.EndHour &&
		t.Hour() >= validTime.StartHour && t.Hour() <= validTime.EndHour {
		log.Debug(ctx, "Expiration Time and Current Time in Window")
		return expirationTime
	}

	// If the expiration time is not in the pick window but the current time is
	if (expirationTime.Hour() < validTime.StartHour || expirationTime.Hour() > validTime.EndHour) &&
		t.Hour() >= validTime.StartHour && t.Hour() <= validTime.EndHour {
		log.Debug(ctx, "Expiration Time not in window and Current Time in Window")
		nextWindow := cfg.Windows[nextDay.Weekday()]
		diff := int(expirationDuration.Hours()) - (validTime.EndHour - t.Hour())
		expirationTime = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), nextWindow.StartHour, nextDay.Minute(), nextDay.Second(), nextDay.Nanosecond(), EasternLocation)
		return expirationTime.Add(time.Duration(diff) * time.Hour)
	}

	// If the current time is not in the pick window
	// We need to find the next pick windows and set the expiration time to
	// expirationDuration after the start of that window
	// To find the next window we get the window for the current day
	// If we are before that window we take that one, if not we take the next one
	log.Debug(ctx, "Current Time not in Window")
	if t.Hour() > validTime.EndHour {
		// If we are after the window move the valid time to the next day
		validTime = cfg.Windows[nextDay.Weekday()]
		return time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), validTime.StartHour, 0, 0, 0, EasternLocation).Add(expirationDuration)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), validTime.StartHour, 0, 0, 0, EasternLocation).Add(expirationDuration)
}
