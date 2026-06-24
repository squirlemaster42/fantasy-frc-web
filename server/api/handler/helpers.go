package apihandler

import (
	"context"
	"database/sql"
	"errors"
	"server/api"
	apimodel "server/api/model"
	"server/api/middleware"
	"server/log"
	"server/model"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func getAuthenticatedUser(c echo.Context) (uuid.UUID, bool) {
	return apimiddleware.GetUserUuid(c)
}

func parseDraftId(c echo.Context) (int, bool) {
	draftIdStr := c.Param("id")
	draftId, err := strconv.Atoi(draftIdStr)
	if err != nil {
		api.BadRequest(c.Response(), "Invalid draft ID")
		return 0, false
	}
	return draftId, true
}

func loadDraft(ctx context.Context, h *Handler, c echo.Context, draftId int) (model.DraftModel, bool) {
	draftModel, err := h.DraftStore.GetDraft(ctx, draftId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			api.NotFound(c.Response(), "Draft not found")
		} else {
			log.Error(ctx, "Failed to get draft", "DraftId", draftId, "Error", err)
			api.InternalError(c.Response())
		}
		return model.DraftModel{}, false
	}
	return draftModel, true
}

func requireDraftOwner(c echo.Context, draftModel model.DraftModel, userUuid uuid.UUID) bool {
	if draftModel.Owner.UserUuid != userUuid {
		api.Forbidden(c.Response(), "You do not have permission to perform this action")
		return false
	}
	return true
}

func requireDraftPlayer(c echo.Context, draftModel model.DraftModel, userUuid uuid.UUID) bool {
	for _, player := range draftModel.Players {
		if player.User.UserUuid == userUuid {
			return true
		}
	}
	api.Forbidden(c.Response(), "You are not a member of this draft")
	return false
}

func mapDraftState(state model.DraftState) apimodel.DraftState {
	switch state {
	case model.FILLING:
		return apimodel.DraftStateFilling
	case model.WAITING_TO_START:
		return apimodel.DraftStateWaitingToStart
	case model.PICKING:
		return apimodel.DraftStatePicking
	case model.TEAMS_PLAYING:
		return apimodel.DraftStateTeamsPlaying
	case model.COMPLETE:
		return apimodel.DraftStateComplete
	}
	return apimodel.DraftState(state)
}

func mapUser(user model.User) apimodel.UserSummary {
	return apimodel.UserSummary{
		Uuid:     user.UserUuid.String(),
		Username: user.Username,
	}
}

func mapPick(p model.Pick) apimodel.PickResponse {
	r := apimodel.PickResponse{
		Id:             p.Id,
		PlayerId:       p.Player,
		AvailableTime:  p.AvailableTime,
		ExpirationTime: p.ExpirationTime,
		Skipped:        p.Skipped,
		Score:          p.Score,
	}
	if p.Pick.Valid {
		r.TeamKey = p.Pick.String
	}
	if p.PickTime.Valid {
		t := p.PickTime.Time
		r.PickTime = &t
	}
	return r
}

func mapDraftPlayer(p model.DraftPlayer) apimodel.DraftPlayerResponse {
	order := int16(-1)
	if p.PlayerOrder.Valid {
		order = p.PlayerOrder.Int16
	}

	picks := make([]apimodel.PickResponse, 0, len(p.Picks))
	for _, pick := range p.Picks {
		picks = append(picks, mapPick(pick))
	}

	return apimodel.DraftPlayerResponse{
		Id:          p.Id,
		User:        mapUser(p.User),
		PlayerOrder: order,
		Pending:     p.Pending,
		Score:       p.Score,
		Picks:       picks,
	}
}

func mapDraft(d model.DraftModel) apimodel.DraftResponse {
	players := make([]apimodel.DraftPlayerResponse, 0, len(d.Players))
	for _, p := range d.Players {
		players = append(players, mapDraftPlayer(p))
	}

	r := apimodel.DraftResponse{
		Id:          d.Id,
		DisplayName: d.DisplayName,
		Description: d.Description,
		Interval:    d.Interval,
		StartTime:   d.StartTime,
		EndTime:     d.EndTime,
		Owner:       mapUser(d.Owner),
		Status:      mapDraftState(d.Status),
		Players:     players,
	}

	if d.NextPick.User.UserUuid != uuid.Nil {
		next := mapDraftPlayer(d.NextPick)
		r.NextPick = &next
	}

	return r
}

func mapInvite(i model.DraftInvite) apimodel.InviteResponse {
	r := apimodel.InviteResponse{
		Id:                 i.Id,
		DraftId:            i.DraftId,
		DraftName:          i.DraftName,
		InvitedUserUuid:    i.InvitedUserUuid.String(),
		InvitingUserUuid:   i.InvitingUserUuid.String(),
		InvitingPlayerName: i.InvitingPlayerName,
		SentTime:           i.SentTime,
		Accepted:           i.Accepted,
	}
	if !i.AcceptedTime.IsZero() {
		t := i.AcceptedTime
		r.AcceptedTime = &t
	}
	return r
}

// respondJSON is a small helper that logs on encoding failure.
// If encoding fails after headers have already been sent, we log the error
// but do not attempt a second write.
func respondJSON(c echo.Context, status int, payload any) error {
	if err := c.JSON(status, payload); err != nil {
		log.Error(c.Request().Context(), "Failed to encode JSON response", "Error", err)
	}
	return nil
}
