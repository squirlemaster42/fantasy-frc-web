package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	log.Debug(c.Request().Context(), "Serving pick page", "Ip", c.RealIP())
	userUuid := c.Get("userUuid").(uuid.UUID)
	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft id string", "Draft Id String", c.Param("id"), "Error", err)
		return err
	}

	return h.renderPickPage(c, draftId, userUuid, nil, true)
}

func (h *Handler) HandlerPickRequest(c echo.Context) error {
	//We need to validate that the curent player is allowed to make a pick for the draft
	//they are on. We then need to make that pick at the draft that they are on
	//Get the player, draft id and the pick

	userUuid := c.Get("userUuid").(uuid.UUID)
	draftIdStr := c.Param("id")
	pick := "frc" + c.FormValue("pickInput")
	log.Debug(c.Request().Context(), "Attempting to pick team", "Team", pick)
	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		log.Warn(c.Request().Context(), "Invalid draft id", "Draft Id String", draftIdStr, "Error", err)
		return err
	}
	log.Info(c.Request().Context(), "Got request for player to make pick in draft", "User Uuid", userUuid, "Pick", pick, "Draft Id", draftId)

	draftModel, err := h.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "User attempted to make pick in invalid draft", "Draft Id", draftId, "User Uuid", userUuid)
		return err
	}

	isCurrentPick := draftModel.NextPick.User.UserUuid == userUuid

	//Make the pick
	draftPlayer, err := h.DraftStore.GetDraftPlayerId(c.Request().Context(), draftId, userUuid)
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
	draftPlayerId, err := h.DraftStore.GetDraftPlayerId(c.Request().Context(), draftId, userUuid)
	if err != nil {
		log.Warn(c.Request().Context(), "Attempting to get draft player", "Draft", draftId, "User Uuid", userUuid, "Error", err)
		draftPlayerId = -1
	}
	isSkipping, err := h.DraftStore.ShouldSkipPick(c.Request().Context(), draftPlayerId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to check if pick should be skipped", "DraftPlayer", draftPlayerId, "Error", err)
		isSkipping = false
	}
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

	pickPageIndex := draft.DraftPickIndex(pickPageModel, h.csrfToken(c))
	if includeWrapper {
		username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
		if err != nil {
			log.Warn(c.Request().Context(), "Failed to get username", "Error", err)
			username = ""
		}
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
		log.Warn(context.TODO(), "Timeout sending pick event to websocket listener", "Listener", w)
		return errors.New("timeout sending to listener")
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (h *Handler) PickNotifier(c echo.Context) error {
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

	userUuid := c.Get("userUuid").(uuid.UUID)

	ticker := time.NewTicker(30 * time.Second)
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
			if err != nil {
				log.Warn(ctx, "Failed to render picks for notifier", "Error", err)
				continue
			}

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
	userUuid := c.Get("userUuid").(uuid.UUID)
	draftIdStr := c.Param("id")
	draftId, err := strconv.Atoi(draftIdStr)

	if err != nil {
		log.Error(c.Request().Context(), "Failed to parse draft id string", "Draft Id String", draftIdStr, "Error", err)
		return err
	}

	draftPlayerId, err := h.DraftStore.GetDraftPlayerId(c.Request().Context(), draftId, userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get draft player", "User uuid", userUuid, "Draft Id String", draftIdStr, "Error", err)
		return err
	}

	shouldSkip := c.FormValue("skipping") != ""
	log.Info(c.Request().Context(), "Marking should skip", "Should Skip", shouldSkip, "Draft Player Id", draftPlayerId)

	return h.DraftStore.MarkShouldSkipPick(c.Request().Context(), draftPlayerId, shouldSkip)
}
