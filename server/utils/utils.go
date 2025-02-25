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

func ParseArgString(argStr string) map[string]string {
    argMap := make(map[string]string)

    curChar := 0
    for curChar < len(argStr) {
        //Find the command
        if argStr[curChar] == '-' && len(argStr) > curChar + 1 {
            //We need to go from the next char to right before the =
            //and make that the command name
        }
    }

    return argMap
}
