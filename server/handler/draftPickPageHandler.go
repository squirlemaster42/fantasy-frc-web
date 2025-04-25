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
    h.renderPickPage(c, draftId, userId, nil)

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

    draftModel, err := model.GetDraft(h.Database, draftId)
    if err != nil {
        slog.Warn("User attempted to make pick in invalid draft", "Draft Id", draftId, "User Id", userId)
        return err
    }

    isCurrentPick := draftModel.NextPick.User.Id == userId

    //Make the pick
    draftPlayer := model.GetDraftPlayerId(h.Database, draftId, userId)
    pickId := model.GetCurrentPick(h.Database, draftId).Id
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

    pickError := h.DraftPickManager.GetPickManagerForDraft(draftId).MakePick(pickStruct)
    if pick == "frc" || !isCurrentPick || pickError != nil {
        slog.Warn("Count Not Make Pick", "Current Pick", isCurrentPick, "Pick", pick, "User Id", userId, "Error", err)
    } else {
        //TODO we need to move this logic into the pick manager
        model.MakePick(h.Database, pickStruct)
        nextPickPlayer := model.NextPick(h.Database, draftId)

        //Make the next pick available if we havn't aleady made all picks
        picks := model.GetPicks(h.Database, draftId)

        slog.Info("Checking if we should make another pick available", "Num picks", len(picks))
        if len(picks) < 64 {
            slog.Info("Making next pick available", "Draft Id", draftId)
            model.MakePickAvailable(h.Database, nextPickPlayer.Id, time.Now(), utils.GetPickExpirationTime(time.Now()))
        } else {
            //Set draft to the teams playing state
            //This isnt entirely correct becuase it doesnt account for skips
            //But I dont care about that for this year
            slog.Info("Update status to TEAMS_PLAYING", "Draft Id", draftId)
            model.UpdateDraftStatus(h.Database, draftId, model.TEAMS_PLAYING)
            h.DraftDaemon.RemoveDraft(draftId)
        }
        //End logic to move

        //We should be able to remove this and it should come from the pick manager instead
        h.Notifier.NotifyWatchers(draftId)
    }

    h.renderPickPage(c, draftId, userId, pickError)
    return nil
}

func (h *Handler) renderPickPage(c echo.Context, draftId int, userId int, pickError error) error {
    draftModel, err := model.GetDraft(h.Database, draftId)
    if err != nil {
        slog.Warn("User is attempting to render pick page for invalid draft", "Draft", draftId, "User", userId)
    }
    url := fmt.Sprintf("/u/draft/%d/makePick", draftId)
    notifierUrl := fmt.Sprintf("/u/draft/%d/pickNotifier", draftId)
    isCurrentPick := draftModel.NextPick.User.Id == userId
    pickPageIndex := draft.DraftPickIndex(draftModel, url, notifierUrl, isCurrentPick, pickError)
    username := model.GetUsername(h.Database, userId)
    pickPageView := draft.DraftPick(" | Draft Picks", true, username, pickPageIndex, draftId)
    err = Render(c, pickPageView)
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
            msg := <-watcher.NotifierQueue
            if msg {
                draftModel, err := model.GetDraft(h.Database, draftId)
                if err != nil {
                    slog.Warn("Attempting to notify draft that does not exist", "Draft Id", draftId)
                    continue
                }

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
