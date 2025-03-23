package utils

import (
	"fmt"
	"time"
)

func Events() []string {
    return []string{
        "2024new",
        "2024mil",
        "2024joh",
        "2024hop",
        "2024gal",
        "2024dal",
        "2024cur",
        "2024arc",
        "2024cmptx",
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
        if argStr[curChar] == '-' && len(argStr) > curChar + 1 {
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
    endHour int
}


var ALLOWED_TIMES = map[time.Weekday]TimeRange {
    time.Sunday: {
        startHour: 8,
        endHour: 22,
    },
    time.Monday: {
        startHour: 17,
        endHour: 22,
    },
    time.Tuesday: {
        startHour: 17,
        endHour: 22,
    },
    time.Wednesday: {
        startHour: 17,
        endHour: 22,
    },
    time.Thursday: {
        startHour: 17,
        endHour: 22,
    },
    time.Friday: {
        startHour: 17,
        endHour: 22,
    },
    time.Saturday: {
        startHour: 8,
        endHour: 22,
    },
}

func GetPickExpirationTime(t time.Time) time.Time {
    expirationTime := t.Add(PICK_TIME)
    validTime := ALLOWED_TIMES[expirationTime.Weekday()]

    if expirationTime.Hour() >= validTime.startHour && expirationTime.Hour() <= validTime.endHour {
        return expirationTime
    }

    nextDay := t.Add(24 * time.Hour)
    nextWindow := ALLOWED_TIMES[nextDay.Weekday()]
    diff := int(PICK_TIME.Hours()) - (validTime.endHour - t.Hour())
    expirationTime = time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), nextWindow.startHour, nextDay.Minute(), nextDay.Second(), nextDay.Nanosecond(), nextDay.Location())
    return expirationTime.Add(time.Duration(diff) * time.Hour)
}
