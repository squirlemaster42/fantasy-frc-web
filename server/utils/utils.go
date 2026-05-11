package utils

import (
	"context"
	"fmt"
	"regexp"
	"server/assert"
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
	log.Info(ctx, "Getting Expiration Time", "Current Time", t)
	expirationTime := t.Add(expirationDuration)
	validTime := ALLOWED_TIMES[expirationTime.Weekday()]
	nextDay := t.Add(24 * time.Hour)

	//If the expiration time is in the pick window and we are currently in the pick window
	if expirationTime.Hour() >= validTime.startHour && expirationTime.Hour() <= validTime.endHour &&
		t.Hour() >= validTime.startHour && t.Hour() <= validTime.endHour {
		log.Info(ctx, "Expiration Time and Current Time in Window")
		return expirationTime
	}

	//If the expiration time is not in the pick window but the current time is
	if (expirationTime.Hour() < validTime.startHour || expirationTime.Hour() > validTime.endHour) &&
		t.Hour() >= validTime.startHour && t.Hour() <= validTime.endHour {
		log.Info(ctx, "Expiration Time not in window and Current Time in Window")
		nextWindow := ALLOWED_TIMES[nextDay.Weekday()]
		diff := int(expirationDuration.Hours()) - (validTime.endHour - t.Hour())
		expirationTime = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), nextWindow.startHour, nextDay.Minute(), nextDay.Second(), nextDay.Nanosecond(), nextDay.Location())
		return expirationTime.Add(time.Duration(diff) * time.Hour)
	}

	//If the current time is not in the pick window
	//We need to find the next pick windows and set the expiraton time to
	//expirationDuration after the start of that window
	//To find the next window we get the window for the current day
	//If we are before that window we take that one, if not we take the next one
	log.Info(ctx, "Current Time not in Window")
	if t.Hour() > validTime.endHour {
		//If we are after the window move the valid time to the next day
		validTime = ALLOWED_TIMES[nextDay.Weekday()]
		return time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), validTime.startHour, 0, 0, 0, nextDay.Location()).Add(expirationDuration)
	} else {
		return time.Date(t.Year(), t.Month(), t.Day(), validTime.startHour, 0, 0, 0, nextDay.Location()).Add(expirationDuration)
	}
}

// Return true if matchA comes before matchB
func CompareMatchOrder(ctx context.Context, matchA string, matchB string) (bool, error) {
	assert := assert.CreateAssertWithContext("Compare Match Order")
	assert.AddContext("Match A", matchA)
	assert.AddContext("Match B", matchB)

	matchALevel, err := getMatchLevel(matchA)
	if err != nil {
		return false, err
	}

	matchBLevel, err := getMatchLevel(matchB)
	if err != nil {
		return false, err
	}

	assert.AddContext("Match A Level", matchALevel)
	assert.AddContext("Match B Level", matchBLevel)
	aPrecidence, ok := matchPrecidence()[matchALevel]
	assert.RunAssert(ctx, ok, "Match Precidence Was Not Found")
	bPrecidence, ok := matchPrecidence()[matchBLevel]
	assert.RunAssert(ctx, ok, "Match Precidence Was Not Found")

	if aPrecidence != bPrecidence {
		return aPrecidence < bPrecidence, nil
	}

	assert.RunAssert(ctx, matchALevel == matchBLevel, "Match levels are not the same")

	if matchALevel == "qm" {
		splitMatchA := strings.Split(matchA, "_")
		splitMatchB := strings.Split(matchB, "_")
		// TODO Return errors instead of crash
		assert.RunAssert(ctx, len(splitMatchA) == 2, "Match A string was invalid")
		assert.RunAssert(ctx, len(splitMatchB) == 2, "Match B string was invalid")
		matchANumStr := strings.TrimSpace(splitMatchA[1][2:])
		matchBNumStr := strings.TrimSpace(splitMatchB[1][2:])
		assert.AddContext("Match A Num", matchANumStr)
		assert.AddContext("Match B Num", matchBNumStr)
		matchANum, err := strconv.Atoi(matchANumStr)
		assert.NoError(ctx, err, "Match A num Atoi failed")
		matchBNum, err := strconv.Atoi(matchBNumStr)
		assert.NoError(ctx, err, "Match B num Atoi failed")
		return matchANum < matchBNum, nil
	}

	if matchALevel == "f" {
		splitMatchA := strings.Split(matchA, "_")
		splitMatchB := strings.Split(matchB, "_")
		assert.RunAssert(ctx, len(splitMatchA) == 2, "Match A string was invalid")
		assert.RunAssert(ctx, len(splitMatchB) == 2, "Match B string was invalid")
		splitMatchA = strings.Split(splitMatchA[1][1:], "m")
		splitMatchB = strings.Split(splitMatchB[1][1:], "m")
		assert.RunAssert(ctx, len(splitMatchA) == 2, "Match A string was invalid")
		assert.RunAssert(ctx, len(splitMatchB) == 2, "Match B string was invalid")
		matchANum, err := strconv.Atoi(splitMatchA[0])
		assert.NoError(ctx, err, "Match A num Atoi failed")
		matchBNum, err := strconv.Atoi(splitMatchB[0])
		assert.NoError(ctx, err, "Match B num Atoi failed")

		if matchANum != matchBNum {
			return matchANum < matchBNum, nil
		}

		assert.RunAssert(ctx, matchANum == matchBNum, "Match nums are the same but shouldn't be")

		matchANum, err = strconv.Atoi(splitMatchA[1])
		assert.NoError(ctx, err, "Match A num Atoi failed")
		matchBNum, err = strconv.Atoi(splitMatchB[1])
		assert.NoError(ctx, err, "Match B num Atoi failed")

		return matchANum < matchBNum, nil
	}

	if matchALevel == "sf" {
		splitMatchA := strings.Split(matchA, "_")
		splitMatchB := strings.Split(matchB, "_")
		assert.RunAssert(ctx, len(splitMatchA) == 2, "Match A string was invalid")
		assert.RunAssert(ctx, len(splitMatchB) == 2, "Match B string was invalid")
		splitMatchA = strings.Split(splitMatchA[1][2:], "m")
		splitMatchB = strings.Split(splitMatchB[1][2:], "m")
		assert.RunAssert(ctx, len(splitMatchA) == 2, "Match A string was invalid")
		assert.RunAssert(ctx, len(splitMatchB) == 2, "Match B string was invalid")
		matchANum, err := strconv.Atoi(splitMatchA[0])
		assert.NoError(ctx, err, "Match A num Atoi failed")
		matchBNum, err := strconv.Atoi(splitMatchB[0])
		assert.NoError(ctx, err, "Match B num Atoi failed")

		if matchANum != matchBNum {
			return matchANum < matchBNum, nil
		}

		assert.RunAssert(ctx, matchANum == matchBNum, "Match nums are the same but shouldn't be")

		matchANum, err = strconv.Atoi(splitMatchA[1])
		assert.NoError(ctx, err, "Match A num Atoi failed")
		matchBNum, err = strconv.Atoi(splitMatchB[1])
		assert.NoError(ctx, err, "Match B num Atoi failed")

		return matchANum < matchBNum, nil
	}

	assert.RunAssert(ctx, 1 == 0, "Unknown match type found")
	return false, nil // This is unreachable
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
	assert := assert.CreateAssertWithContext("Get Match Level")
	assert.AddContext("Match Key", matchKey)
	pattern := regexp.MustCompile("_[a-z]+")
	match := pattern.FindString(matchKey)[1:]
	assert.AddContext("Match", match)
	if !(len(match) == 2 || len(match) == 1) {
		return "", fmt.Errorf("match string %s was not im expected format", match)
	}
	return match, nil
}

func GetWebhookFilePath() string {
	return "./webhookSecret.txt"
}
