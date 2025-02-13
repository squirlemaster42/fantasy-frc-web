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
