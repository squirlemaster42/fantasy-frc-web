package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
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
	log.Debug(c.Request().Context(), "Serving pick page", "ip", c.RealIP())
	userUuid := c.Get("userUuid").(uuid.UUID)
	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft id string", "draftIdString", c.Param("id"), "error", err)
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
	log.Debug(c.Request().Context(), "Attempting to pick team", "team", pick)
	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		log.Warn(c.Request().Context(), "Invalid draft id", "draftIdString", draftIdStr, "error", err)
		return err
	}
	log.Debug(c.Request().Context(), "Got request for player to make pick in draft", "userUuid", userUuid, "pick", pick, "draftId", draftId)

	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "draftId", draftId, "error", err)
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
			Time:  time.Now().UTC(),
		},
	}

	var pickError error
	if pick == "frc" || !isCurrentPick {
		log.Warn(c.Request().Context(), "Could Not Make Pick", "isCurrentPick", isCurrentPick, "pick", pick, "userUuid", userUuid)
        pickError = errors.New("you must be the picking player to make a pick")
        return h.renderPickPage(c, draftId, userUuid, pickError, false)
	}
	pickError = draft.MakePick(c.Request().Context(), draftActor, pickStruct)
	if pickError != nil {
		log.Warn(c.Request().Context(), "Could Not Make Pick", "isCurrentPick", isCurrentPick, "pick", pick, "userUuid", userUuid, "error", pickError)
	}

	return h.renderPickPage(c, draftId, userUuid, pickError, false)
}

func (h *Handler) renderPickPage(c echo.Context, draftId int, userUuid uuid.UUID, pickError error, includeWrapper bool) error {
	draftActor, err := h.DraftActorMap.GetActor(c.Request().Context(), draftId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft actor", "draftId", draftId, "error", err)
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
		log.Warn(c.Request().Context(), "Attempting to get draft player", "draftId", draftId, "userUuid", userUuid, "error", err)
		draftPlayerId = -1
	}
	isSkipping, err := h.DraftStore.ShouldSkipPick(c.Request().Context(), draftPlayerId)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to check if pick should be skipped", "draftPlayerId", draftPlayerId, "error", err)
		isSkipping = false
	}
	log.Debug(c.Request().Context(), "Loaded if picks should be skipped", "draftPlayerId", draftPlayerId, "isSkipping", isSkipping)

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
			log.Error(c.Request().Context(), "Failed to get username", "error", err)
			username = ""
		}
		pickPageView := draftView.DraftPick(" | Draft Picks", true, username, pickPageIndex, draftId, isOwner)
		if err := Render(c, pickPageView); err != nil {
			log.Error(c.Request().Context(), "Failed to render pick page", "draftId", draftId, "error", err)
			return err
		}
		return nil
	} else {
		if err := Render(c, pickPageIndex); err != nil {
			log.Error(c.Request().Context(), "Failed to render pick page index", "draftId", draftId, "error", err)
			return err
		}
		return nil
	}
}

func (h *Handler) newUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			if h.AllowedOrigin != "" {
				return origin == h.AllowedOrigin
			}
			host := r.Host
			return strings.HasPrefix(origin, "http://"+host) ||
				strings.HasPrefix(origin, "https://"+host) ||
				strings.HasPrefix(origin, "http://localhost:") ||
				strings.HasPrefix(origin, "https://localhost:")
		},
	}
}

func (h *Handler) PickNotifier(c echo.Context) error {
	ctx := c.Request().Context()

	upgrader := h.newUpgrader()
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Warn(ctx, "Failed to upgrade websocket connection", "error", err)
		return err
	}

	draftIdStr := c.Param("id")
	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		log.Warn(ctx, "Failed to parse draft id string", "draftIdString", draftIdStr, "error", err)
		conn.Close()
		return err
	}

	draftActor, err := h.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "draftId", draftId, "error", err)
		conn.Close()
		return err
	}

	watcher := draft.RegisterWatcher(ctx, h.DraftActorMap, draftId)
	if watcher == nil {
		log.Error(ctx, "Failed to register watcher for draft", "draftId", draftId)
		conn.Close()
		return errors.New("failed to register watcher")
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		if err != nil {
			log.Warn(c.Request().Context(), "Failed to set context read deadline", "error", err)
		}
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		})
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
					log.Warn(ctx, "Websocket unexpected close", "draftId", draftId, "error", err)
				}
				return
			}
		}
	}()

	userUuid := c.Get("userUuid").(uuid.UUID)

	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		draft.UnregisterWatcher(ctx, h.DraftActorMap, watcher)
		conn.Close()
		<-done
	}()

	for {
		select {
		case <-done:
			log.Debug(ctx, "Client disconnected, closing pick notifier", "draftId", draftId, "userUuid", userUuid)
			return nil
		case <-ticker.C:
			err = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				log.Warn(ctx, "Failed to set context deadline for pick notifier", "draftId", draftId, "error", err)
			}
			if err = conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
				log.Error(ctx, "Failed to write ping message", "draftId", draftId, "error", err)
				return err
			}
		case <-watcher.NotifierQueue:
			log.Debug(ctx, "Received pick event notification, re-rendering picks", "draftId", draftId)
			draftModel := draft.GetDraft(draftActor)

			var html strings.Builder
			pickPage := draftView.RenderPicks(draftModel, draftModel.NextPick.User.UserUuid == userUuid)
			err = pickPage.Render(ctx, &html)
			if err != nil {
				log.Error(ctx, "Failed to render picks for notifier", "error", err)
				continue
			}

			err = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err != nil {
				log.Warn(ctx, "failed to set context deadline on websocket context", "draftId", draftId, "error", err)
			}
			err = conn.WriteMessage(websocket.TextMessage, []byte(html.String()))
			if err != nil {
				log.Warn(ctx, "Failed to send message to websocket", "draftId", draftId, "error", err)
				return err
			}
		}
	}
}

func (h *Handler) HandleSkipPickToggle(c echo.Context) error {
	userUuid := c.Get("userUuid").(uuid.UUID)
	draftIdStr := c.Param("id")
	draftId, err := strconv.Atoi(draftIdStr)

	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft id string", "draftIdString", draftIdStr, "error", err)
		return err
	}

	draftPlayerId, err := h.DraftStore.GetDraftPlayerId(c.Request().Context(), draftId, userUuid)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get draft player", "userUuid", userUuid, "draftId", draftId, "error", err)
		return err
	}

	shouldSkip := c.FormValue("skipping") != ""
	log.Debug(c.Request().Context(), "Marking should skip", "shouldSkip", shouldSkip, "draftPlayerId", draftPlayerId)

	return h.DraftStore.MarkShouldSkipPick(c.Request().Context(), draftPlayerId, shouldSkip)
}
