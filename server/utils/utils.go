package utils

import (
	"context"
	"fmt"
	"regexp"
	"server/log"
	"strconv"
	"strings"
	"time"
)

func Events() []string {
	return []string{
		"2026arc",
		"2026cur",
		"2026dal",
		"2026gal",
		"2026hop",
		"2026joh",
		"2026mil",
		"2026new",
		"2026cmptx",
	}
}

func Einstein() string {
	return "2026cmptx"
}

func GetUpdateUrl(draftId int) string {
	if draftId == -1 {
		return "/u/createDraft"
	} else {
		return fmt.Sprintf("/u/draft/%d/updateDraft", draftId)
	}
}

func ParseArgString(argStr string) (map[string]string, error) {
	argMap := make(map[string]string)

	curChar := 0
	for curChar < len(argStr) {
		//Find the command
		if argStr[curChar] == '-' && len(argStr) > curChar+1 {
			//We need to go from the next char to right before the =
			//and make that the command name
			argName := ""
			curChar++
			for len(argStr) > curChar && argStr[curChar] != '=' {
				argName += string(argStr[curChar])
				curChar++
			}

			if len(argStr) <= curChar || argStr[curChar] != '=' {
				//There is no value for this flag so we just signify its present by putting the key in the map
				argMap[argName] = ""
				continue
			}

			//We need to get the arg value
			//The arg val can either just exist of can be have double quotes
			//We dont need to worry about having nested quotes
			curChar++

			if len(argStr) <= curChar {
				break
			}

			var searchChar byte = ' '
			if argStr[curChar] == '"' {
				searchChar = '"'
				curChar++
			}

			argVal := ""
			for len(argStr) > curChar && argStr[curChar] != searchChar {
				argVal += string(argStr[curChar])
				curChar++
			}

			argMap[argName] = argVal
		}
		curChar++
	}

	return argMap, nil
}

var PICK_TIME time.Duration = 1 * time.Hour

// EasternLocation is the canonical America/New_York timezone used for all
// draft scheduling, pick windows, and user-facing time display.
var EasternLocation *time.Location

func init() {
	var err error
	EasternLocation, err = time.LoadLocation("America/New_York")
	if err != nil {
		log.Fatal(context.Background(), "Failed to load Eastern timezone", "Error", err)
	}
}

type TimeRange struct {
	startHour int
	endHour   int
}

// todo we should make it so this in configurable per draft
var ALLOWED_TIMES = map[time.Weekday]TimeRange{
	time.Sunday: {
		startHour: 8,
		endHour:   22,
	},
	time.Monday: {
		startHour: 17,
		endHour:   22,
	},
	time.Tuesday: {
		startHour: 17,
		endHour:   22,
	},
	time.Wednesday: {
		startHour: 17,
		endHour:   22,
	},
	time.Thursday: {
		startHour: 17,
		endHour:   22,
	},
	time.Friday: {
		startHour: 17,
		endHour:   22,
	},
	time.Saturday: {
		startHour: 8,
		endHour:   22,
	},
}

func GetPickExpirationTime(ctx context.Context, t time.Time, expirationDuration time.Duration) time.Time {
	// All pick scheduling is done in Eastern time.
	t = t.In(EasternLocation)
	log.Info(ctx, "Getting Expiration Time", "Current Time", t)
	expirationTime := t.Add(expirationDuration)
	validTime := ALLOWED_TIMES[expirationTime.Weekday()]
	nextDay := t.Add(24 * time.Hour)

	//If the expiration time is in the pick window and we are currently in the pick window
	if expirationTime.Hour() >= validTime.startHour && expirationTime.Hour() <= validTime.endHour &&
		t.Hour() >= validTime.startHour && t.Hour() <= validTime.endHour {
		log.Debug(ctx, "Expiration Time and Current Time in Window")
		return expirationTime
	}

	//If the expiration time is not in the pick window but the current time is
	if (expirationTime.Hour() < validTime.startHour || expirationTime.Hour() > validTime.endHour) &&
		t.Hour() >= validTime.startHour && t.Hour() <= validTime.endHour {
		log.Debug(ctx, "Expiration Time not in window and Current Time in Window")
		nextWindow := ALLOWED_TIMES[nextDay.Weekday()]
		diff := int(expirationDuration.Hours()) - (validTime.endHour - t.Hour())
		expirationTime = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), nextWindow.startHour, nextDay.Minute(), nextDay.Second(), nextDay.Nanosecond(), EasternLocation)
		return expirationTime.Add(time.Duration(diff) * time.Hour)
	}

	//If the current time is not in the pick window
	//We need to find the next pick windows and set the expiraton time to
	//expirationDuration after the start of that window
	//To find the next window we get the window for the current day
	//If we are before that window we take that one, if not we take the next one
	log.Debug(ctx, "Current Time not in Window")
	if t.Hour() > validTime.endHour {
		//If we are after the window move the valid time to the next day
		validTime = ALLOWED_TIMES[nextDay.Weekday()]
		return time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), validTime.startHour, 0, 0, 0, EasternLocation).Add(expirationDuration)
	} else {
		return time.Date(t.Year(), t.Month(), t.Day(), validTime.startHour, 0, 0, 0, EasternLocation).Add(expirationDuration)
	}
}

// comparePlayoffMatch parses and compares two playoff match strings (e.g. "f1m1", "sf2m1").
// The prefixLen is the number of characters to skip after the underscore before the match number.
func comparePlayoffMatch(matchA, matchB string, prefixLen int) (bool, error) {
	splitMatchA := strings.Split(matchA, "_")
	splitMatchB := strings.Split(matchB, "_")
	if len(splitMatchA) != 2 {
		return false, fmt.Errorf("match A string %q was invalid", matchA)
	}
	if len(splitMatchB) != 2 {
		return false, fmt.Errorf("match B string %q was invalid", matchB)
	}
	splitMatchA = strings.Split(splitMatchA[1][prefixLen:], "m")
	splitMatchB = strings.Split(splitMatchB[1][prefixLen:], "m")
	if len(splitMatchA) != 2 {
		return false, fmt.Errorf("match A string %q was invalid", matchA)
	}
	if len(splitMatchB) != 2 {
		return false, fmt.Errorf("match B string %q was invalid", matchB)
	}
	matchANum, err := strconv.Atoi(splitMatchA[0])
	if err != nil {
		return false, fmt.Errorf("failed to parse match A num %q: %w", splitMatchA[0], err)
	}
	matchBNum, err := strconv.Atoi(splitMatchB[0])
	if err != nil {
		return false, fmt.Errorf("failed to parse match B num %q: %w", splitMatchB[0], err)
	}

	if matchANum != matchBNum {
		return matchANum < matchBNum, nil
	}

	matchANum, err = strconv.Atoi(splitMatchA[1])
	if err != nil {
		return false, fmt.Errorf("failed to parse match A num %q: %w", splitMatchA[1], err)
	}
	matchBNum, err = strconv.Atoi(splitMatchB[1])
	if err != nil {
		return false, fmt.Errorf("failed to parse match B num %q: %w", splitMatchB[1], err)
	}

	return matchANum < matchBNum, nil
}

// Return true if matchA comes before matchB
func CompareMatchOrder(ctx context.Context, matchA string, matchB string) (bool, error) {
	matchALevel, err := getMatchLevel(matchA)
	if err != nil {
		return false, fmt.Errorf("invalid match A %q: %w", matchA, err)
	}

	matchBLevel, err := getMatchLevel(matchB)
	if err != nil {
		return false, fmt.Errorf("invalid match B %q: %w", matchB, err)
	}

	aPrecidence, ok := matchPrecidence()[matchALevel]
	if !ok {
		return false, fmt.Errorf("match precedence not found for level %q", matchALevel)
	}
	bPrecidence, ok := matchPrecidence()[matchBLevel]
	if !ok {
		return false, fmt.Errorf("match precedence not found for level %q", matchBLevel)
	}

	if aPrecidence != bPrecidence {
		return aPrecidence < bPrecidence, nil
	}

	if matchALevel != matchBLevel {
		return false, fmt.Errorf("match levels are not the same: %q vs %q", matchALevel, matchBLevel)
	}

	if matchALevel == "qm" {
		splitMatchA := strings.Split(matchA, "_")
		splitMatchB := strings.Split(matchB, "_")
		if len(splitMatchA) != 2 {
			return false, fmt.Errorf("match A string %q was invalid", matchA)
		}
		if len(splitMatchB) != 2 {
			return false, fmt.Errorf("match B string %q was invalid", matchB)
		}
		matchANumStr := strings.TrimSpace(splitMatchA[1][2:])
		matchBNumStr := strings.TrimSpace(splitMatchB[1][2:])
		matchANum, err := strconv.Atoi(matchANumStr)
		if err != nil {
			return false, fmt.Errorf("failed to parse match A num %q: %w", matchANumStr, err)
		}
		matchBNum, err := strconv.Atoi(matchBNumStr)
		if err != nil {
			return false, fmt.Errorf("failed to parse match B num %q: %w", matchBNumStr, err)
		}
		return matchANum < matchBNum, nil
	}

	if matchALevel == "f" {
		return comparePlayoffMatch(matchA, matchB, 1)
	}

	if matchALevel == "sf" {
		return comparePlayoffMatch(matchA, matchB, 2)
	}

	return false, fmt.Errorf("unknown match type %q", matchALevel)
}

func matchPrecidence() map[string]int {
	return map[string]int{
		"qm": 0,
		"qf": 1,
		"sf": 2,
		"f":  3,
	}
}

func getMatchLevel(matchKey string) (string, error) {
	pattern := regexp.MustCompile("_[a-z]+")
	match := pattern.FindString(matchKey)[1:]
	if len(match) != 2 && len(match) != 1 {
		return "", fmt.Errorf("match string %s was not im expected format", match)
	}
	return match, nil
}

func GetWebhookFilePath() string {
	return "./webhookSecret.txt"
}
