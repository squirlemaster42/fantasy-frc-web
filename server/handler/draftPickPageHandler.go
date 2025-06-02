package handler

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"server/assert"
	"server/model"
	"server/picking"
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
    if err != nil {
        slog.Warn("Failed to parse draft id string", "Draft Id String", c.Param("id"), "Error", err)
        return err
    }

    return h.renderPickPage(c, draftId, userId, nil)
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

    pickError := h.DraftManager.MakePick(draftId, pickStruct)
    if pick == "frc" || !isCurrentPick || pickError != nil {
        slog.Warn("Could Not Make Pick", "Current Pick", isCurrentPick, "Pick", pick, "User Id", userId, "Error", err)
    }

    return h.renderPickPage(c, draftId, userId, pickError)
}

func (h *Handler) renderPickPage(c echo.Context, draftId int, userId int, pickError error) error {
    draftModel, err := model.GetDraft(h.Database, draftId)
    if err != nil {
        slog.Warn("User is attempting to render pick page for invalid draft", "Draft", draftId, "User", userId)
    }
    url := fmt.Sprintf("/u/draft/%d/makePick", draftId)
    notifierUrl := fmt.Sprintf("/u/draft/%d/pickNotifier", draftId)
    isCurrentPick := draftModel.NextPick.User.Id == userId
    pickPageModel := draft.PickPage {
        Draft: draftModel,
        PickUrl: url,
        NotifierUrl: notifierUrl,
        IsCurrentPick: isCurrentPick,
        PickError: pickError,
        IsSkipping: model.ShoudSkipPick(h.Database, draftModel.NextPick.Id),
    }
    pickPageIndex := draft.DraftPickIndex(pickPageModel)
    username := model.GetUsername(h.Database, userId)
    pickPageView := draft.DraftPick(" | Draft Picks", true, username, pickPageIndex, draftId)
    err = Render(c, pickPageView)
    return err
}

type WebSocketListener struct {
    messageQueue chan picking.PickEvent
}

func (w WebSocketListener) ReceivePickEvent(pickEvent picking.PickEvent) {
    w.messageQueue <- pickEvent
}

func (h *Handler) PickNotifier(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Pick Notifier")
    websocket.Handler(func(ws *websocket.Conn) {
        draftIdStr := c.Param("id")
        draftId, err := strconv.Atoi(draftIdStr)

        if err != nil {
            slog.Error("Failed to parse draft id string", "Draft Id String", draftIdStr, "Error", err)
        }

        wsl := WebSocketListener {
            messageQueue: make(chan picking.PickEvent),
        }
        h.DraftManager.AddPickListener(draftId, wsl)
        userTok, err := c.Cookie("sessionToken")
        assert.NoError(err, "Failed to get user token")
        userId := model.GetUserBySessionToken(h.Database, userTok.Value)
        defer func() {
            err = ws.Close()
            if err != nil {
                slog.Warn("Failed to close pick notifier web socket", "Draft Id", draftId, "User", userId, "Error", err)
            }
        }()
        //TODO Figure out how to unregister the listener
        //defer h.Notifier.UnregiserWatcher(watcher)
        assert.NoError(err, "Could not parse draft id")
        for {
            msg := <- wsl.messageQueue
            slog.Info("Writing pick event to client", "Event", msg)
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
                slog.Warn("Failed to sent message to websocket")
                break
            }
        }
    }).ServeHTTP(c.Response(), c.Request())
    return nil
}
