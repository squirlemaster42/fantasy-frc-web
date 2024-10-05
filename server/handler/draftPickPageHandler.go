package handler

import (
	"database/sql"
	"server/model"
	"server/view/draft"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

func (h *Handler) ServePickPage(c echo.Context) error {
    draftId, err := strconv.Atoi(c.Param("id"))
    draftModel := model.GetDraft(h.Database, draftId)
    pickPageIndex := draft.DraftPickIndex(draftModel, getPickHtml(h.Database, draftId, len(draftModel.Players)))
    pickPageView := draft.DraftPick(" | " + draftModel.DisplayName, false, pickPageIndex)
    err = Render(c, pickPageView)

    return err
}

func getPickHtml(db *sql.DB, draftId int, numPlayers int) string {
    var stringBuilder strings.Builder
    picks := model.GetPicks(db, draftId)

    row := 0
    totalRows := len(picks) / numPlayers
    for loc, pick := range picks {
        if loc % numPlayers == 0 {
            if row != 0 {
                stringBuilder.WriteString("</tr>")
            }
            stringBuilder.WriteString("<tr class=\"bg-white border-b dark:bg-gray-800 dark:border:gray-700\">")
            if row == totalRows {
                blanks := numPlayers - (len(picks) % numPlayers)
                if blanks != 0 {
                    for i := 0; i < blanks; i++ {
                        stringBuilder.WriteString("<td class=\"border px-6 py-3\"></td>")
                    }
                }
            }
            row++
        }
        stringBuilder.WriteString("<td class=\"border px-6 py-3\">")
        stringBuilder.WriteString(pick.Pick)
        stringBuilder.WriteString("</td>")
    }
    stringBuilder.WriteString("</tr>")

    for row < 8 {
        stringBuilder.WriteString("<tr class=\"bg-white border-b dark:bg-gray-800 dark:border:gray-700\">")
        for i := 0; i < numPlayers; i++ {
            stringBuilder.WriteString("<td class=\"border px-6 py-5\"></td>")
        }
        stringBuilder.WriteString("</tr>")
        row++
    }

    return stringBuilder.String()
}
