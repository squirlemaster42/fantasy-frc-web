package handler

import (
    "database/sql"
    "encoding/json"
    "fmt"
    "server/assert"
    "server/model"
    "server/view/draft"
    "strconv"
    "strings"
    "time"

    "github.com/labstack/echo/v4"
    "golang.org/x/net/websocket"
)

func (h *Handler) ServePickPage(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Server Pick Page")
    h.Logger.Log(fmt.Sprintf("Serving pick page to %s", c.RealIP()))
    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")
    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    draftId, err := strconv.Atoi(c.Param("id"))
    renderPickPage(c, h.Database, draftId, userId, false)

    return err
}

func getPickHtml(db *sql.DB, draftId int, numPlayers int, currentPick bool, pickId int) string {
    var stringBuilder strings.Builder
    picks := model.GetPicks(db, draftId)

    start := 0
    end := numPlayers - 1
    curRow := 0
    totalRows := len(picks) / numPlayers
    for {
        var row []model.Pick
        if len(picks) != 0 {
            row = picks[start:end]
            if curRow%2 == 1 {
                row = reverseArray(row)
            }
        }

        stringBuilder.WriteString("<tr class=\"bg-white border-b dark:bg-gray-800 dark:border:gray-700\">")
        if curRow == (len(picks)%numPlayers)-1 {
            blanks := numPlayers - (len(picks) % numPlayers)
            if blanks != 0 {
                for i := 0; i < blanks-1; i++ {
                    stringBuilder.WriteString("<td class=\"border px-6 py-3\"></td>")
                }
                stringBuilder.WriteString("<td class=\"border\">")
                if currentPick {
                    stringBuilder.WriteString(fmt.Sprintf("<input name=\"pickInput\" hx-vals=\"{pickId: %d}\" class=\"w-full h-full bg-transparent pl-4 border-none\"/>", pickId))
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
        end = min(end+numPlayers, len(picks))
        curRow++
        if curRow > totalRows {
            break
        }
    }
    stringBuilder.WriteString("</tr>")

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
    for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
        s[i], s[j] = s[j], s[i]
    }
    return s
}

type PickRequest struct {
    PickId int `json:"pickId"`
}

func (h *Handler) HandlerPickRequest(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Handle Pick Request")
    //We need to validate that the curent player is allowed to make a pick for the draft
    //they are on. We then need to make that pick at the draft that they are on
    //Get the player, draft id and the pick

    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")
    draftIdStr := c.Param("id")
    pick := c.FormValue("pickInput")
    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    draftId, err := strconv.Atoi(draftIdStr)
    assert.NoError(err, "Invalid draft id") //Make sure that the pick is valid
    isInvalid := false
    if !model.ValidPick(h.Database, &h.TbaHandler, pick, draftId) {
        isInvalid = true
        h.Logger.Log("Invalid Pick")
    } else {
        var pickInfo PickRequest
        err := json.NewDecoder(c.Request().Body).Decode(&pickInfo)
        assert.NoError(err, "Failed to decode request json")

        //Make the pick
        draftPlayer := model.GetDraftPlayerId(h.Database, draftId, userId)
        pickStruct := model.Pick{
            Id:       pickInfo.PickId,
            Player:   draftPlayer,
            Pick:     pick,
            PickTime: time.Now(),
        }
        model.MakePick(h.Database, pickStruct)

        nextPickPlayer := model.NextPick(h.Database, draftId)
        model.MakePickAvailable(h.Database, nextPickPlayer.Id, time.Now())

        draftModel := model.GetDraft(h.Database, draftId)
        //We need to rethink this because we need to notify the watcher who has the next pick with differnt html
        draftText := "<tbody id=\"pickTableBody\">" + getPickHtml(h.Database, draftId, len(draftModel.Players), false, 0) + "</tbody>"
        h.Notifier.NotifyWatchers(draftId, draftText)
    }

    renderPickPage(c, h.Database, draftId, userId, isInvalid)

    return nil
}

func renderPickPage(c echo.Context, database *sql.DB, draftId int, userId int, invalidPick bool) error {
    draftModel := model.GetDraft(database, draftId)
    url := fmt.Sprintf("/draft/%d/makePick", draftId)
    notifierUrl := fmt.Sprintf("/draft/%d/pickNotifier", draftId)
    nextPick := model.NextPick(database, draftId)
    isPicking := nextPick.User.Id == userId
    html := getPickHtml(database, draftId, len(draftModel.Players), isPicking, nextPick.Id)
    pickPageIndex := draft.DraftPickIndex(draftModel, html, url, invalidPick, notifierUrl)
    username := model.GetUsername(database, userId)
    pickPageView := draft.DraftPick(" | Draft Picks", true, username, pickPageIndex, draftId)
    err := Render(c, pickPageView)
    return err
}

func (h *Handler) PickNotifier(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Pick Notifier")
    //TODO Do we need more auth here?
    websocket.Handler(func(ws *websocket.Conn) {
        draftIdStr := c.Param("id")
        draftId, err := strconv.Atoi(draftIdStr)
        watcher := h.Notifier.RegisterWatcher(draftId)
        defer ws.Close()
        defer h.Notifier.UnregiserWatcher(watcher)
        assert.NoError(err, "Could not parse draft id")
        for {
            msg := <-watcher.notifierQueue
            //Enable the input for the current pick
            userTok, err := c.Cookie("sessionToken")
            assert.NoError(err, "Failed to get user token")
            userId := model.GetUserBySessionToken(h.Database, userTok.Value)
            //Disabled should appear no where else in this string so we can just find and replace
            if model.NextPick(h.Database, draftId).User.Id == userId {
                msg = strings.Replace(msg, "disabled", "", -1)
            }
            err = websocket.Message.Send(ws, msg)
            assert.NoError(err, "Websocket receive failed")
        }
    }).ServeHTTP(c.Response(), c.Request())
    return nil
}
