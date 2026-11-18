package handler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"server/draft"
	"server/log"
	"server/model"
	"server/picking"
	"server/types"
	draftView "server/view/draft"
	"server/view/team"
	"slices"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v5"
)

func (h *Handler) HandleDraftScore(c *echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	draftId, err := h.parseDraftId(c)
	if err != nil {
		return BadRequest(c, "Failed to convert draft id to int", "Draft id was not an int", "error", err)
	}

	draftModel, err := h.Stores.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NotFound(c, "Draft not found", fmt.Sprintf("Failed to load draft id %d", draftId))
		}
		return InternalServerError(c, "Failed to get draft by id", err, "draftId", draftId)
	}

	isOwner := draftModel.Owner.UserUuid == userUuid

	userDraftScore, err := h.Stores.DraftStore.GetDraftScore(c.Request().Context(), draftId)
	if err != nil {
		return InternalServerError(c, "Failed to get draft score", err)
	}

	sortDraftScore(userDraftScore)

	scoreNotifierUrl := fmt.Sprintf("/u/draft/%d/scoreNotifier", draftId)
	avatarColors := collectDraftAvatarColors(c.Request().Context(), h.Services.AvatarStore, userDraftScore)

	draftIndex := draftView.DraftScoreIndex(userDraftScore, draftId, draftModel.Status, avatarColors, scoreNotifierUrl)
	draftScoreView := draftView.DraftScore("Draft Scores", true, username, draftIndex, types.NewPageData(draftId, draftModel.DisplayName, isOwner))
	return RenderOrLog(c, draftScoreView, "Failed to render draft score page", "draftId", draftId)
}

func (h *Handler) HandleDraftTeamScore(c *echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	draftId, err := h.parseDraftId(c)
	if err != nil {
		return BadRequest(c, "Failed to convert draft id to int", "Draft id was not an int", "error", err)
	}

	draftModel, err := h.Stores.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return NotFound(c, "Draft not found", fmt.Sprintf("Failed to load draft id %d", draftId))
		}
		return InternalServerError(c, "Failed to get draft by id", err, "draftId", draftId)
	}

	isOwner := draftModel.Owner.UserUuid == userUuid

	teamNumber := c.Param("teamNumber")

	scores, err := h.Stores.TeamStore.GetScore(c.Request().Context(), teamPrefix+teamNumber)
	if err != nil {
		return InternalServerError(c, "Failed to get team score", err, "teamNumber", teamNumber)
	}

	// Get qualification matches
	qualificationMatches, err := h.Stores.TeamStore.GetMatchScores(c.Request().Context(), teamPrefix+teamNumber)
	if err != nil {
		return InternalServerError(c, "Failed to get match scores", err, "teamNumber", teamNumber)
	}

	avatarColor := avatarColorForTeam(c.Request().Context(), h.Services.AvatarStore, teamNumber)

	teamScoreReport := team.TeamScoreReport(teamNumber, scores, qualificationMatches, avatarColor)
	draftTeamScore := draftView.DraftTeamScore(" | Score Breakdown", true, username, teamScoreReport, types.NewPageData(draftModel.Id, draftModel.DisplayName, isOwner))
	return RenderOrLog(c, draftTeamScore, "Failed to render draft team score page", "draftId", draftModel.Id, "teamNumber", teamNumber)
}

// ScoreNotifier upgrades the HTTP connection to a WebSocket and streams live
// draft score updates. Connections are only accepted when the draft is in the
// TEAMS_PLAYING state; otherwise the upgrade is rejected with a 400 response.
func (h *Handler) ScoreNotifier(c *echo.Context) error {
	ctx := c.Request().Context()

	userUuid, err := h.requireUserUuid(c)
	if err != nil {
		return err
	}

	draftIdStr := c.Param("id")
	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		log.Error(ctx, "Failed to parse draft id string", "draftIdString", draftIdStr, "error", err)
		return c.NoContent(http.StatusBadRequest)
	}

	_, err = h.Stores.DraftStore.GetDraftPlayerId(ctx, draftId, userUuid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.NoContent(http.StatusForbidden)
		}
		log.Error(ctx, "Failed to verify draft membership", "draftId", draftId, "userUuid", userUuid, "error", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	draftModel, err := h.Stores.DraftStore.GetDraft(ctx, draftId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.NoContent(http.StatusNotFound)
		}
		log.Error(ctx, "Failed to get draft by id", "draftId", draftId, "error", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	if draftModel.Status != model.TEAMS_PLAYING {
		log.Warn(ctx, "Score notifier rejected for draft not in teams playing state", "draftId", draftId, "status", draftModel.Status)
		return c.NoContent(http.StatusBadRequest)
	}

	upgrader := h.newUpgrader()
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		log.Warn(ctx, "Failed to upgrade websocket connection", "error", err)
		return c.NoContent(http.StatusBadRequest)
	}
	conn.SetReadLimit(WsMaxMessageSize())

	watcher := draft.RegisterWatcher(ctx, h.Services.DraftActorMap, draftId)
	if watcher == nil {
		log.Error(ctx, "Failed to register watcher for draft", "draftId", draftId)
		_ = conn.Close()
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := conn.SetReadDeadline(time.Now().Add(WsReadTimeout()))
		if err != nil {
			log.Warn(ctx, "Failed to set context read deadline", "error", err)
		}
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(WsReadTimeout()))
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

	ticker := time.NewTicker(WsPingInterval())
	defer func() {
		ticker.Stop()
		draft.UnregisterWatcher(ctx, h.Services.DraftActorMap, watcher)
		_ = conn.Close()
		<-done
	}()

	for {
		select {
		case <-done:
			log.Debug(ctx, "Client disconnected, closing score notifier", "draftId", draftId, "userUuid", userUuid)
			return nil
		case <-ticker.C:
			err = conn.SetWriteDeadline(time.Now().Add(WsWriteTimeout()))
			if err != nil {
				log.Warn(ctx, "Failed to set context deadline for score notifier", "draftId", draftId, "error", err)
			}
			if err = conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(WsWriteTimeout())); err != nil {
				log.Error(ctx, "Failed to write ping message", "draftId", draftId, "error", err)
				return nil
			}
		case notification := <-watcher.NotifierQueue:
			if notification.Type == picking.DraftStateChangedNotificationType {
				if notification.Status != model.TEAMS_PLAYING {
					log.Debug(ctx, "Draft no longer in teams playing state, closing score notifier", "draftId", draftId, "status", notification.Status)
					return nil
				}
				continue
			}
			if notification.Type != picking.ScoreNotificationType {
				// Ignore pick notifications on the score watcher.
				continue
			}

			log.Debug(ctx, "Received score event notification, re-rendering draft scores", "draftId", draftId)
			html, err := h.renderDraftScoreTable(ctx, draftId)
			if err != nil {
				log.Error(ctx, "Failed to render draft scores for notifier", "draftId", draftId, "error", err)
				continue
			}

			err = conn.SetWriteDeadline(time.Now().Add(WsWriteTimeout()))
			if err != nil {
				log.Warn(ctx, "Failed to set context deadline on websocket context", "draftId", draftId, "error", err)
			}
			err = conn.WriteMessage(websocket.TextMessage, []byte(html))
			if err != nil {
				log.Warn(ctx, "Failed to send message to websocket", "draftId", draftId, "error", err)
				return nil
			}
		}
	}
}

// renderDraftScoreTable returns the rendered DraftScoreTable component for the
// given draft. It applies the same sorting and avatar color collection used by
// the HTTP score page and is pushed over the score notifier WebSocket.
// sortDraftScore sorts players by descending total score and each player's
// picks by descending individual score. It mutates the slice in place.
func sortDraftScore(userDraftScore []model.DraftPlayer) {
	slices.SortFunc(userDraftScore, func(a, b model.DraftPlayer) int {
		return b.Score - a.Score
	})

	for _, draftPlayer := range userDraftScore {
		slices.SortFunc(draftPlayer.Picks, func(a, b model.Pick) int {
			return b.Score - a.Score
		})
	}
}

func (h *Handler) renderDraftScoreTable(ctx context.Context, draftId int) (string, error) {
	userDraftScore, err := h.Stores.DraftStore.GetDraftScore(ctx, draftId)
	if err != nil {
		return "", err
	}

	sortDraftScore(userDraftScore)

	avatarColors := collectDraftAvatarColors(ctx, h.Services.AvatarStore, userDraftScore)

	return RenderToString(ctx, draftView.DraftScoreTable(userDraftScore, draftId, avatarColors))
}
