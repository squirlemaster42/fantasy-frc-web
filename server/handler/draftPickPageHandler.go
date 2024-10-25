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

    "golang.org/x/net/websocket"
	"github.com/labstack/echo/v4"
)

func (h *Handler) ServePickPage(c echo.Context) error {
	draftId, err := strconv.Atoi(c.Param("id"))
	renderPickPage(c, h.Database, draftId, false)

	return err
}

func getPickHtml(db *sql.DB, draftId int, numPlayers int, currentPick bool) string {
	var stringBuilder strings.Builder
	picks := model.GetPicks(db, draftId)

    start := 0
    end := numPlayers
    curRow := 0
	totalRows := len(picks) / numPlayers
	for {
        row := picks[start:end]
        if curRow % 2 == 1 {
            row = reverseArray(row)
        }

        stringBuilder.WriteString("<tr class=\"bg-white border-b dark:bg-gray-800 dark:border:gray-700\">")
        if curRow == (len(picks) % numPlayers) - 1 {
            blanks := numPlayers - (len(picks) % numPlayers)
            if blanks != 0 {
                for i := 0; i < blanks-1; i++ {
                    stringBuilder.WriteString("<td class=\"border px-6 py-3\"></td>")
                }
                stringBuilder.WriteString("<td class=\"border\">")
                //TODO Need to disable if its not the current persons pick
                if currentPick {
                    stringBuilder.WriteString("<input name=\"pickInput\" class=\"w-full h-full bg-transparent pl-4 border-none\"/>")
                } else {
                    stringBuilder.WriteString("<input name=\"pickInput\" disabled class=\"w-full h-full bg-transparent pl-4 border-none\"/>")
                }
                stringBuilder.WriteString("</td>")
            }
        }

        for _, pick := range row {
            stringBuilder.WriteString("<td class=\"border px-6 py-3\">")
            stringBuilder.WriteString(pick.Pick)
            stringBuilder.WriteString("</td>")
        }
        stringBuilder.WriteString("</tr>")

        start = end
        end = min(end + numPlayers, len(picks))
        curRow++
        if curRow > totalRows {
            break
        }
    }
    stringBuilder.WriteString("</tr>")

    //TODO Change 8 to number of picks
    for curRow < 8 {
        stringBuilder.WriteString("<tr class=\"bg-white border-b dark:bg-gray-800 dark:border:gray-700\">")
        for i := 0; i < numPlayers; i++ {
            stringBuilder.WriteString("<td class=\"border px-6 py-5\"></td>")
        }
        stringBuilder.WriteString("</tr>")
        curRow++
    }

    return stringBuilder.String()
}

func reverseArray(s []model.Pick) []model.Pick {
    for i, j := 0, len(s) - 1; i < j; i, j = i + 1, j - 1 {
        s[i], s[j] = s[j], s[i]
    }
    return s
}

func (h *Handler) HandlerPickRequest(c echo.Context) error {
    //We need to validate that the curent player is allowed to make a pick for the draft
    //they are on. We then need to make that pick at the draft that they are on
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
    isInvalid := false
    //if we didnt the login was invalid
    if !model.ValidPick(h.Database, &h.TbaHandler, pick, draftId) {
        isInvalid = true
        h.Logger.Log("Invalid Pick")
    } else {
        //Make the pick
        //TODO Get pick order (or maybe we should just drop that concept)
        draftPlayer := model.GetDraftPlayerId(h.Database, draftId, userId)
        pickStruct := model.Pick{
            Player:    draftPlayer,
            PickOrder: 0,
            Pick:      pick,
            PickTime:  time.Now(),
        }
        model.MakePick(h.Database, pickStruct)

        draftModel := model.GetDraft(h.Database, draftId)
        draftText := "<tbody id=\"pickTableBody\">" + getPickHtml(h.Database, draftId, len(draftModel.Players), false) + "</tbody>"
        h.Notifier.NotifyWatchers(draftId, draftText)
    }

    renderPickPage(c, h.Database, draftId, isInvalid)

    return nil
}

func renderPickPage(c echo.Context, database *sql.DB, draftId int, invalidPick bool) error {
    draftModel := model.GetDraft(database, draftId)
    url := fmt.Sprintf("/draft/%d/makePick", draftId)
    notifierUrl := fmt.Sprintf("/draft/%d/pickNotifier", draftId)
    html := getPickHtml(database, draftId, len(draftModel.Players), false)
	pickPageIndex := draft.DraftPickIndex(draftModel, html, url, invalidPick, notifierUrl)
	pickPageView := draft.DraftPick(" | "+draftModel.DisplayName, false, pickPageIndex)
	err := Render(c, pickPageView)
	return err
}

func (h *Handler) PickNotifier(c echo.Context) error {
    //TODO Need to do authentication
    websocket.Handler(func (ws *websocket.Conn) {
        draftIdStr := c.Param("id")
        draftId, err := strconv.Atoi(draftIdStr)
        watcher := h.Notifier.RegisterWatcher(draftId)
        defer ws.Close()
        defer h.Notifier.UnregiserWatcher(watcher)
        assert.NoErrorCF(err, "Could not parse draft id")
        for {
            msg := <- watcher.notifierQueue
            err = websocket.Message.Send(ws, msg)
            assert.NoErrorCF(err, "Websocket receive failed")
        }
    }).ServeHTTP(c.Response(), c.Request())
    return nil
}
