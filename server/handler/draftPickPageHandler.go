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
	"golang.org/x/net/websocket"
)

func (h *Handler) ServePickPage(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Server Pick Page")
    h.Logger.Log(fmt.Sprintf("Serving pick page to %s", c.RealIP()))
    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")
    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    draftId, err := strconv.Atoi(c.Param("id"))
    h.renderPickPage(c, draftId, userId, false)

    return err
}

func reverseArray(s []model.Pick) []model.Pick {
    for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
        s[i], s[j] = s[j], s[i]
    }
    return s
}

func (h *Handler) HandlerPickRequest(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Handle Pick Request")
    //We need to validate that the curent player is allowed to make a pick for the draft
    //they are on. We then need to make that pick at the draft that they are on
    //Get the player, draft id and the pick

    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")
    draftIdStr := c.Param("id")
    pick := "frc" + c.FormValue("pickInput")
    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    draftId, err := strconv.Atoi(draftIdStr)
    h.Logger.Log(fmt.Sprintf("Got request for player %d to make pick %s in draft %d", userId, pick, draftId))
    assert.NoError(err, "Invalid draft id") //Make sure that the pick is valid
    isInvalid := false
    if !model.ValidPick(h.Database, &h.TbaHandler, pick, draftId) {
        isInvalid = true
        h.Logger.Log("Invalid Pick")
    } else {
        pickId, err := strconv.Atoi(c.FormValue("pickId"))
        assert.NoError(err, "Failed to convert pickId to int")

        //Make the pick
        draftPlayer := model.GetDraftPlayerId(h.Database, draftId, userId)
        pickStruct := model.Pick{
            Id: pickId,
            Player: draftPlayer,
            Pick: sql.NullString{
                Valid: true,
                String: pick,
            },
            PickTime: sql.NullTime{
                Valid: true,
                Time: time.Now(),
            },
        }
        model.MakePick(h.Database, pickStruct)

        nextPickPlayer := model.NextPick(h.Database, draftId)
        model.MakePickAvailable(h.Database, nextPickPlayer.Id, time.Now())

        /*
        draftModel := model.GetDraft(h.Database, draftId)
        //We need to rethink this because we need to notify the watcher who has the next pick with different html
        draftText := "<tbody id=\"pickTableBody\">" + getPickHtml(h.Database, draftId, len(draftModel.Players), false) + "</tbody>"
        h.Notifier.NotifyWatchers(draftId, draftText)
        */
    }

    h.renderPickPage(c, draftId, userId, isInvalid)
    return nil
}

func (h *Handler) renderPickPage(c echo.Context, draftId int, userId int, invalidPick bool) error {
    draftModel := model.GetDraft(h.Database, draftId)
    url := fmt.Sprintf("/u/draft/%d/makePick", draftId)
    notifierUrl := fmt.Sprintf("/u/draft/%d/pickNotifier", draftId)
    nextPick := model.GetAvailablePickId(h.Database, draftId)
    vals := fmt.Sprintf("{\"pickId\": %d}", nextPick.Id)
    pickPageIndex := draft.DraftPickIndex(draftModel, "", url, invalidPick, notifierUrl, vals)
    username := model.GetUsername(h.Database, userId)
    pickPageView := draft.DraftPick(" | Draft Picks", true, username, pickPageIndex, draftId)
    err := Render(c, pickPageView)
    return err
}

func (h *Handler) PickNotifier(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Pick Notifier")
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
