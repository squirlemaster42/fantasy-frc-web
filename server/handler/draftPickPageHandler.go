package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"server/assert"
	"server/log"
	"server/model"
	"server/picking"
	"server/view/draft"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

func (h *Handler) ServePickPage(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Server Pick Page")
	log.Debug(c.Request().Context(), "Serving pick page", "Ip", c.RealIP())
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")
	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft id string", "Draft Id String", c.Param("id"), "Error", err)
		return err
	}

	return h.renderPickPage(c, draftId, userUuid, nil, true)
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
	log.Debug(c.Request().Context(), "Attempting to pick team", "Team", pick)
	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	draftId, err := strconv.Atoi(draftIdStr)
	log.Info(c.Request().Context(), "Got request for player to make pick in draft", "User Uuid", userUuid, "Pick", pick, "Draft Id", draftId)
	assert.NoError(err, "Invalid draft id") //Make sure that the pick is valid

	draftModel, err := model.GetDraft(h.Database, draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "User attempted to make pick in invalid draft", "Draft Id", draftId, "User Uuid", userUuid)
		return err
	}

	isCurrentPick := draftModel.NextPick.User.UserUuid == userUuid

	//Make the pick
	draftPlayer, err := model.GetDraftPlayerId(h.Database, draftId, userUuid)
	if err != nil {
		return err
	}
	currPick, err := h.DraftManager.GetCurrentPick(draftId)
	if err != nil {
		return err
	}
	pickStruct := model.Pick{
		Id:     currPick.Id,
		Player: draftPlayer,
		Pick: sql.NullString{
			Valid:  true,
			String: pick,
		},
		PickTime: sql.NullTime{
			Valid: true,
			Time:  time.Now(),
		},
	}

	if pick == "frc" || !isCurrentPick {
		log.Warn(c.Request().Context(), "Could Not Make Pick", "Current Pick", isCurrentPick, "Pick", pick, "User Uuid", userUuid)
        pickError := errors.New("you must be the picking player to make a pick")
        return h.renderPickPage(c, draftId, userUuid, pickError, false)
	}
	pickError := h.DraftManager.MakePick(draftId, pickStruct)
	if pickError != nil {
		log.Warn(c.Request().Context(), "Could Not Make Pick", "Current Pick", isCurrentPick, "Pick", pick, "User Uuid", userUuid, "Error", err)
	}

	return h.renderPickPage(c, draftId, userUuid, pickError, false)
}

func (h *Handler) renderPickPage(c echo.Context, draftId int, userUuid uuid.UUID, pickError error, includeWrapper bool) error {
	cachedDraft, err := h.DraftManager.GetDraft(draftId, false)
	if err != nil {
		log.Warn(c.Request().Context(), "User is attempting to render pick page for invalid draft", "Draft", draftId, "User Uuid", userUuid)
	}
	draftModel := *cachedDraft.Model
	pickUrl := fmt.Sprintf("/u/draft/%d/makePick", draftId)
	notifierUrl := fmt.Sprintf("/u/draft/%d/pickNotifier", draftId)
	skipUrl := fmt.Sprintf("/u/draft/%d/skipPickToggle", draftId)
	isCurrentPick := draftModel.NextPick.User.UserUuid == userUuid
	isOwner := draftModel.Owner.UserUuid == userUuid
	draftPlayerId, err := model.GetDraftPlayerId(h.Database, draftId, userUuid)
	if err != nil {
		log.Warn(c.Request().Context(), "Attempting to get draft player", "Draft", draftId, "User Uuid", userUuid, "Error", err)
		draftPlayerId = -1
	}
	isSkipping := model.ShouldSkipPick(h.Database, draftPlayerId)
	log.Debug(c.Request().Context(), "Loaded if picks should be skipped", "DraftPlayer", draftPlayerId, "Is Skipping", isSkipping)

	pickPageModel := draft.PickPage{
		Draft:         draftModel,
		PickUrl:       pickUrl,
		NotifierUrl:   notifierUrl,
		IsCurrentPick: isCurrentPick,
		PickError:     pickError,
		IsSkipping:    isSkipping,
		SkipUrl:       skipUrl,
	}

	pickPageIndex := draft.DraftPickIndex(pickPageModel)
	if includeWrapper {
		username := model.GetUsername(h.Database, userUuid)
		pickPageView := draft.DraftPick(" | Draft Picks", true, username, pickPageIndex, draftId, isOwner)
		err = Render(c, pickPageView)
	} else {
		err = Render(c, pickPageIndex)
	}

	return err
}

type WebSocketListener struct {
	messageQueue chan picking.PickEvent
}

func (w *WebSocketListener) ReceivePickEvent(pickEvent picking.PickEvent) error {
	select {
	case w.messageQueue <- pickEvent:
		return nil
	case <-time.After(5 * time.Second):
		log.WarnNoContext("Timeout sending pick event to websocket listener", "Listener", w)
		return errors.New("timeout sending to listener")
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *Handler) PickNotifier(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Pick Notifier")
	ctx := c.Request().Context()

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Warn(ctx, "Failed to upgrade websocket connection", "Error", err)
		return nil
	}

	draftIdStr := c.Param("id")
	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		log.Error(ctx, "Failed to parse draft id string", "Draft Id String", draftIdStr, "Error", err)
		conn.Close()
		return nil
	}

	wsl := WebSocketListener{
		messageQueue: make(chan picking.PickEvent, 10),
	}
	h.DraftManager.AddPickListener(draftId, &wsl)

	done := make(chan struct{})
	go func() {
		defer close(done)
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(120 * time.Second))
			return nil
		})
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Warn(ctx, "Websocket unexpected close", "Draft Id", draftId, "Error", err)
				}
				return
			}
		}
	}()

	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")
	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)

	ticker := time.NewTicker(60 * time.Second)
	defer func() {
		ticker.Stop()
		h.DraftManager.RemovePickListener(draftId, &wsl)
		conn.Close()
		<-done
	}()

	for {
		select {
		case <-done:
			log.Info(ctx, "Client disconnected, closing pick notifier", "Draft Id", draftId, "User Uuid", userUuid)
			return nil
		case <-ticker.C:
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				log.Warn(ctx, "Failed to write ping message", "Draft Id", draftId, "Error", err)
				return nil
			}
		case msg := <-wsl.messageQueue:
			log.Info(ctx, "Writing pick event to client", "Event", msg)
			cachedDraft, err := h.DraftManager.GetDraft(draftId, false)
			if err != nil {
				log.Warn(ctx, "Attempting to notify draft that does not exist", "Draft Id", draftId)
				continue
			}
			draftModel := *cachedDraft.Model

			var html strings.Builder
			pickPage := draft.RenderPicks(draftModel, draftModel.NextPick.User.UserUuid == userUuid)
			err = pickPage.Render(context.Background(), &html)
			assert.NoError(err, "Failed to render picks for notifier")

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			err = conn.WriteMessage(websocket.TextMessage, []byte(html.String()))
			if err != nil {
				log.Warn(ctx, "Failed to send message to websocket", "Draft Id", draftId, "Error", err)
				return nil
			}
		}
	}
}

func (h *Handler) HandleSkipPickToggle(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Skip Page Toggle")
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")
	userUuid := model.GetUserBySessionToken(h.Database, userTok.Value)
	draftIdStr := c.Param("id")
	draftId, err := strconv.Atoi(draftIdStr)

	if err != nil {
		log.Error(c.Request().Context(), "Failed to parse draft id string", "Draft Id String", draftIdStr, "Error", err)
		return err
	}

	draftPlayerId, err := model.GetDraftPlayerId(h.Database, draftId, userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get draft player", "User uuid", userUuid, "Draft Id String", draftIdStr, "Error", err)
		return err
	}

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to read body of request to toggle skip pick", "Error", err)
	}

	log.Info(c.Request().Context(), "Got request to toggle skip pick", "Body", body)

	// See if we have the skip in the list
	// If we do then mark the player as skipping for the given draft
	// If not then mark them as not skipping
	shouldSkip := strings.Contains(string(body), "skipping")
	log.Info(c.Request().Context(), "Marking should skip", "Should Skip", shouldSkip, "Draft Player Id", draftPlayerId)

	return model.MarkShouldSkipPick(h.Database, draftPlayerId, shouldSkip)
}
