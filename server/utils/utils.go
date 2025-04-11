package utils

import (
	"fmt"
	"log/slog"
	"time"
)

func Events() []string {
    return []string{
        "2025arc",
        "2025cur",
        "2025dal",
        "2025gal",
        "2025hop",
        "2025joh",
        "2025mil",
        "2025new",
        "2025cmptx",
    }
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

            if !(len(argStr) > curChar && argStr[curChar] == '=') {
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

var PICK_TIME time.Duration = 3 * time.Hour

type TimeRange struct {
    startHour int
    endHour   int
}

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

func GetPickExpirationTime(t time.Time) time.Time {
    slog.Info("Getting Expiration Time", "Current Time", t)
    expirationTime := t.Add(PICK_TIME)
    validTime := ALLOWED_TIMES[expirationTime.Weekday()]
    nextDay := t.Add(24 * time.Hour)

    //If the expiration time is in the pick window and we are currently in the pick window
    if expirationTime.Hour() >= validTime.startHour && expirationTime.Hour() <= validTime.endHour &&
        t.Hour() >= validTime.startHour && t.Hour() <= validTime.endHour {
        slog.Info("Expiration Time and Current Time in Window")
        return expirationTime
    }

    //If the expiration time is not in the pick window but the current time is
    if !(expirationTime.Hour() >= validTime.startHour && expirationTime.Hour() <= validTime.endHour) &&
        t.Hour() >= validTime.startHour && t.Hour() <= validTime.endHour {
        slog.Info("Expiration Time not in window and Current Time in Window")
        nextWindow := ALLOWED_TIMES[nextDay.Weekday()]
        diff := int(PICK_TIME.Hours()) - (validTime.endHour - t.Hour())
        expirationTime = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), nextWindow.startHour, nextDay.Minute(), nextDay.Second(), nextDay.Nanosecond(), nextDay.Location())
        return expirationTime.Add(time.Duration(diff) * time.Hour)
    }

    //If the current time is not in the pick window
    //We need to find the next pick windows and set the expiraton time to
    //PICK_TIME after the start of that window
    //To find the next window we get the window for the current day
    //If we are before that window we take that one, if not we take the next one
    slog.Info("Current Time not in Window")
    if t.Hour() > validTime.endHour {
        //If we are after the window move the valid time to the next day
        validTime = ALLOWED_TIMES[nextDay.Weekday()]
        return time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), validTime.startHour, nextDay.Minute(), nextDay.Second(), nextDay.Nanosecond(), nextDay.Location()).Add(PICK_TIME)
    } else {
        return time.Date(t.Year(), t.Month(), t.Day(), validTime.startHour, nextDay.Minute(), nextDay.Second(), nextDay.Nanosecond(), nextDay.Location()).Add(PICK_TIME)
    }
}

func Einstein() string {
	return "2025cmptx"
}

