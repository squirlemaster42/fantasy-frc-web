package handler

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"server/assert"
	"server/model"
	"server/utils"
	"server/view/draft"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

func (h *Handler) ServePickPage(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Server Pick Page")
    slog.Info("Serving pick page", "Ip", c.RealIP())
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
    slog.Info("Attempting to pick team", "Team", pick)
    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    draftId, err := strconv.Atoi(draftIdStr)
    slog.Info("Got request for player to make pick in draft", "User Id", userId, "Pick", pick, "Draft Id", draftId)
    assert.NoError(err, "Invalid draft id") //Make sure that the pick is valid

    draftModel := model.GetDraft(h.Database, draftId)
    isCurrentPick := draftModel.NextPick.User.Id == userId

    isInvalid := false
    validPick := model.ValidPick(h.Database, &h.TbaHandler, pick, draftId)
    if !validPick || !isCurrentPick{
        isInvalid = true
        slog.Warn("Count Not Make Pick", "Valid Pick", validPick, "Current Pick", isCurrentPick, "Pick", pick, "User Id", userId)
    } else {
        pickId := model.GetCurrentPick(h.Database, draftId).Id

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

        //Make the next pick available if we havn't aleady made all picks
        picks := model.GetPicks(h.Database, draftId)

        if len(picks) < 64 {
            model.MakePickAvailable(h.Database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now()))
        }

        h.Notifier.NotifyWatchers(draftId)
    }

    h.renderPickPage(c, draftId, userId, isInvalid)
    return nil
}

func (h *Handler) renderPickPage(c echo.Context, draftId int, userId int, invalidPick bool) error {
    draftModel := model.GetDraft(h.Database, draftId)
    url := fmt.Sprintf("/u/draft/%d/makePick", draftId)
    notifierUrl := fmt.Sprintf("/u/draft/%d/pickNotifier", draftId)
    isCurrentPick := draftModel.NextPick.User.Id == userId
    pickPageIndex := draft.DraftPickIndex(draftModel, url, invalidPick, notifierUrl, isCurrentPick)
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
        userTok, err := c.Cookie("sessionToken")
        assert.NoError(err, "Failed to get user token")
        userId := model.GetUserBySessionToken(h.Database, userTok.Value)
        defer ws.Close()
        defer h.Notifier.UnregiserWatcher(watcher)
        assert.NoError(err, "Could not parse draft id")
        for {
            msg := <-watcher.notifierQueue
            if msg {
                draftModel := model.GetDraft(h.Database, draftId)

                var html strings.Builder
                pickPage := draft.RenderPicks(draftModel, draftModel.NextPick.User.Id == userId)
                err = pickPage.Render(context.Background(), &html)
                assert.NoError(err, "Failed to render picks for notifier")

                err = websocket.Message.Send(ws, html.String())
                if err != nil {
                    break
                }
            }
        }
    }).ServeHTTP(c.Response(), c.Request())
    return nil
}
