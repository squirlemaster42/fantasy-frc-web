package utils

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
        return "/u/draft/updateDraft"
    }
}

//TODO Add errors for malformed argument strings
func ParseArgString(argStr string) map[string]string {
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

    return argMap
}
