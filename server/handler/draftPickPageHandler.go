package handler

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"server/assert"
	"server/model"
	"server/picking"
	"server/view/draft"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/net/websocket"
)

func (h *Handler) ServePickPage(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Server Pick Page")
    slog.Info("Serving pick page", "Ip", c.RealIP())
    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")
    userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
    draftId, err := strconv.Atoi(c.Param("id"))
    if err != nil {
        slog.Warn("Failed to parse draft id string", "Draft Id String", c.Param("id"), "Error", err)
        return err
    }

    return h.renderPickPage(c, draftId, userUuid, nil)
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
    userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
    draftId, err := strconv.Atoi(draftIdStr)
    slog.Info("Got request for player to make pick in draft", "User Uuid", userUuid, "Pick", pick, "Draft Id", draftId)
    assert.NoError(err, "Invalid draft id") //Make sure that the pick is valid

    draftModel, err := model.GetDraft(h.Database, draftId)
    if err != nil {
        slog.Warn("User attempted to make pick in invalid draft", "Draft Id", draftId, "User Uuid", userUuid)
        return err
    }

    isCurrentPick := draftModel.NextPick.User.UserUuid == userUuid

    //Make the pick
    draftPlayer := model.GetDraftPlayerId(h.Database, draftId, userUuid)
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
        slog.Warn("Could Not Make Pick", "Current Pick", isCurrentPick, "Pick", pick, "User Uuid", userUuid, "Error", err)
    }

    return h.renderPickPage(c, draftId, userUuid, pickError)
}

func (h *Handler) renderPickPage(c echo.Context, draftId int, userUuid uuid.UUID, pickError error) error {
    draftModel, err := model.GetDraft(h.Database, draftId)
    if err != nil {
        slog.Warn("User is attempting to render pick page for invalid draft", "Draft", draftId, "User Uuid", userUuid)
    }
    pickUrl := fmt.Sprintf("/u/draft/%d/makePick", draftId)
    notifierUrl := fmt.Sprintf("/u/draft/%d/pickNotifier", draftId)
    skipUrl := fmt.Sprintf("/u/draft/%d/skipPickToggle", draftId)
    isCurrentPick := draftModel.NextPick.User.UserUuid == userUuid
    draftPlayerId := model.GetDraftPlayerId(h.Database, draftId, userUuid)
    isSkipping := model.ShoudSkipPick(h.Database, draftPlayerId)
    slog.Info("Loaded if picks should be skipped", "DraftPlayer", draftPlayerId, "Is Skipping", isSkipping)

    pickPageModel := draft.PickPage {
        Draft: draftModel,
        PickUrl: pickUrl,
        NotifierUrl: notifierUrl,
        IsCurrentPick: isCurrentPick,
        PickError: pickError,
        IsSkipping: isSkipping,
        SkipUrl: skipUrl,
    }
    pickPageIndex := draft.DraftPickIndex(c, pickPageModel)
    username := model.GetUsername(h.Database, userUuid)
    pickPageView := draft.DraftPick(c, " | Draft Picks", true, username, pickPageIndex, draftId)
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
            return
        }

        wsl := WebSocketListener {
            messageQueue: make(chan picking.PickEvent),
        }
        h.DraftManager.AddPickListener(draftId, wsl)
        userTok, err := c.Cookie("sessionToken")
        assert.NoError(err, "Failed to get user token")
        userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
        defer func() {
            err = ws.Close()
            if err != nil {
                slog.Warn("Failed to close pick notifier web socket", "Draft Id", draftId, "User Uuid", userUuid, "Error", err)
            }
        }()
        //TODO Figure out how to unregister the listener
        //defer h.Notifier.UnregiserWatcher(watcher)
        for {
            msg := <- wsl.messageQueue
            slog.Info("Writing pick event to client", "Event", msg)
            draftModel, err := model.GetDraft(h.Database, draftId)
            if err != nil {
                slog.Warn("Attempting to notify draft that does not exist", "Draft Id", draftId)
                continue
            }

            var html strings.Builder
            pickPage := draft.RenderPicks(draftModel, draftModel.NextPick.User.UserUuid == userUuid)
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

func (h *Handler) HandleSkipPickToggle(c echo.Context) error {
    assert := assert.CreateAssertWithContext("Handle Skip Page Toggle")
    userTok, err := c.Cookie("sessionToken")
    assert.NoError(err, "Failed to get user token")
    userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
    draftIdStr := c.Param("id")
    draftId, err := strconv.Atoi(draftIdStr)
    draftPlayerId := model.GetDraftPlayerId(h.Database, draftId, userUuid)

    if err != nil {
        slog.Error("Failed to parse draft id string", "Draft Id String", draftIdStr, "Error", err)
        return err
    }

    body, err := io.ReadAll(c.Request().Body)
    if err != nil {
        slog.Error("Failed to read body of request to toggle skip pick", "Error", err)
    }

    slog.Info("Got request to toggle skip pick", "Body", body)

    // See if we have the skip in the list
    // If we do then mark the player as skipping for the given draft
    // If not then mark them as not skipping
    shouldSkip := strings.Contains(string(body), "skipping")
    slog.Info("Marking should skip", "Should Skip", shouldSkip, "Draft Player Id", draftPlayerId)

    return model.MarkShouldSkipPick(h.Database, draftPlayerId, shouldSkip)
}
