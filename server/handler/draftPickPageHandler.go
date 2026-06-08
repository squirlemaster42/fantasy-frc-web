package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"server/draft"
	"server/log"
	"server/model"
	draftView "server/view/draft"
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

	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "Draft Id", draftId, "Error", err)
		return err
	}
	draftState := draftActor.GetDraftState()

	isCurrentPick := draftState.NextPick.User.UserUuid == userUuid

	// Make the pick
	// TODO we could move this to the actor so we dont have to call the db
	draftPlayer, err := h.DraftStore.GetDraftPlayerId(c.Request().Context(), draftId, userUuid)
	if err != nil {
		return err
	}

	pickStruct := model.Pick{
		Id:     draftState.CurrentPick.Id,
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

	var pickError error
	if pick == "frc" || !isCurrentPick {
		log.Warn(c.Request().Context(), "Could Not Make Pick", "Current Pick", isCurrentPick, "Pick", pick, "User Uuid", userUuid)
        pickError = errors.New("you must be the picking player to make a pick")
        return h.renderPickPage(c, draftId, userUuid, pickError, false)
	}
	replyChan := make(chan draft.Result)
	message := draft.Message {
		Content: draft.PickMessage{
			Pick: pickStruct,
		},
		Reply: replyChan,
	}
	err = draftActor.PostMessage(c.Request().Context(), message)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to post pick message", "Draft Id", draftId, "Error", err)
		pickError = err
	} else {
		select {
		case result := <- message.Reply:
			if result.Error != nil {
				pickError = result.Error
			}
		case <- time.After(5 * time.Second):
			log.Warn(c.Request().Context(), "making pick in draft timed out", "Draft Id", draftId, "Current Pick Id", draftActor.GetDraftState().CurrentPick.Id)
			pickError = errors.New("timeout making pick")
		}
	}
	if pickError != nil {
		log.Warn(c.Request().Context(), "Could Not Make Pick", "Current Pick", isCurrentPick, "Pick", pick, "User Uuid", userUuid, "Error", pickError)
	}

	return h.renderPickPage(c, draftId, userUuid, pickError, false)
}

func (h *Handler) renderPickPage(c echo.Context, draftId int, userUuid uuid.UUID, pickError error, includeWrapper bool) error {
	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "Draft Id", draftId, "Error", err)
		return err
	}
	draftState := draftActor.GetDraftState()
	pickUrl := fmt.Sprintf("/u/draft/%d/makePick", draftId)
	notifierUrl := fmt.Sprintf("/u/draft/%d/pickNotifier", draftId)
	skipUrl := fmt.Sprintf("/u/draft/%d/skipPickToggle", draftId)
	isCurrentPick := draftState.NextPick.User.UserUuid == userUuid
	isOwner := draftState.Owner.UserUuid == userUuid
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

	pickPageModel := draftView.PickPage{
		Draft:         draftState,
		PickUrl:       pickUrl,
		NotifierUrl:   notifierUrl,
		IsCurrentPick: isCurrentPick,
		PickError:     pickError,
		IsSkipping:    isSkipping,
		SkipUrl:       skipUrl,
	}

	pickPageIndex := draftView.DraftPickIndex(pickPageModel, h.csrfToken(c))
	if includeWrapper {
		username, err := h.UserStore.GetUsername(c.Request().Context(), userUuid)
		if err != nil {
			log.Warn(c.Request().Context(), "Failed to get username", "Error", err)
			username = ""
		}
		pickPageView := draftView.DraftPick(" | Draft Picks", true, username, pickPageIndex, draftId, isOwner)
		return Render(c, pickPageView)
	} else {
		return Render(c, pickPageIndex)
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

	draftActor, err := h.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "Draft Id", draftId, "Error", err)
		conn.Close()
		return nil
	}

	watcher := draft.RegisterWatcher(h.DraftActorMap, draftId)
	if watcher == nil {
		log.Error(ctx, "Failed to register watcher for draft", "Draft Id", draftId)
		conn.Close()
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		if err != nil {
			log.Warn(c.Request().Context(), "Failed to set context read deadline", "Error", err)
		}
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(120 * time.Second))
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
		draft.UnregisterWatcher(h.DraftActorMap, watcher)
		conn.Close()
		<-done
	}()

	for {
		select {
		case <-done:
			log.Info(ctx, "Client disconnected, closing pick notifier", "Draft Id", draftId, "User Uuid", userUuid)
			return nil
		case <-ticker.C:
			err = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				log.Warn(ctx, "Failed to set context deadline for pick notifier", "Draft Id", draftId, "Error", err)
			}
			if err = conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				log.Warn(ctx, "Failed to write ping message", "Draft Id", draftId, "Error", err)
				return nil
			}
		case <-watcher.NotifierQueue:
			log.Info(ctx, "Received pick event notification, re-rendering picks", "Draft Id", draftId)
			draftModel := draft.GetDraft(draftActor)

			var html strings.Builder
			pickPage := draftView.RenderPicks(draftModel, draftModel.NextPick.User.UserUuid == userUuid)
			err = pickPage.Render(context.TODO(), &html)
			if err != nil {
				log.Warn(ctx, "Failed to render picks for notifier", "Error", err)
				continue
			}

			err = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				log.Warn(ctx, "failed to set context deadline on websocket context", "Draft Id", draftId, "Error", err)
			}
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
