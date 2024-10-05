package handler

import (
	"database/sql"
	"fmt"
	"server/assert"
	"server/model"
	"server/view/draft"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func (h *Handler) ServePickPage(c echo.Context) error {
    draftId, err := strconv.Atoi(c.Param("id"))
    draftModel := model.GetDraft(h.Database, draftId)
    url := fmt.Sprintf("/draft/%d/makePick", draftId)
    pickPageIndex := draft.DraftPickIndex(draftModel, getPickHtml(h.Database, draftId, len(draftModel.Players)), url)
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
                    for i := 0; i < blanks - 1; i++ {
                        stringBuilder.WriteString("<td class=\"border px-6 py-3\"></td>")
                    }
                    //TODO Need to figure out how to set the width of this
                    stringBuilder.WriteString("<td class=\"border\">")
                    //TODO Need to disable if its not the current persons pick
                    stringBuilder.WriteString("<input name=\"pickInput\" class=\"w-full h-full bg-transparent pl-4 border-none\"/>")
                    stringBuilder.WriteString("</td>")
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

func (h *Handler) HandlerPickRequest(c echo.Context) error {
    //We need to validate that the curent player is allowed to make a pick for the draft
    //they are on. We then need to make that pick at the draft that they are on
    //How do we get the draft id here?


    //Get the player, draft id and the pick

    //TODO maybe we should change validate login to just take in the context type
    //so we dont hape to parse out the token every time
    userTok, err := c.Cookie("sessionToken")
    assert.NoErrorCF(err, "Failed to get user token")
    draftIdStr := c.Param("id")
    pick := c.FormValue("pickInput")
    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    draftId, err := strconv.Atoi(draftIdStr)
    assert.NoErrorCF(err, "Invalid draft id")

    //Make sure that the pick is valid
    //TODO Check that we actually got the session token
    //if we didnt the login was invalid
    if !model.ValidPick(h.Database, pick, draftId) {
        //TODO Figure out how to surface this
        h.Logger.Log("Invalid Pick")
    }

    //Make the pick
    //TODO Get pick order (or maybe we should just drop that concept)
    draftPlayer := model.GetDraftPlayerId(h.Database, draftId, userId)
    pickStruct := model.Pick{
        Player: draftPlayer,
        PickOrder: 0,
        Pick: pick,
        PickTime: time.Now(),
    }
    model.MakePick(h.Database, pickStruct)

    //TODO Figure out what to return to the user
    //I think we just want to reconstruct what we show for the normal view
    //and since the pick was already made it will just be in there

    //TODO Notify other users via the web socket

    return nil
}

func renderPickPage() error {
    //TODO Move reder stuff to here
    return nil
}
