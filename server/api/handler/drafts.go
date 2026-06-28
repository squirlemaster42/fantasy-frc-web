package apihandler

import (
	"database/sql"
	"net/http"
	"server/api"
	apimodel "server/api/model"
	"server/draft"
	"server/log"
	"server/model"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

// ListDrafts returns all drafts visible to the authenticated user.
func (h *Handler) ListDrafts(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	drafts, err := h.DraftStore.GetDraftsForUser(ctx, userUuid)
	if err != nil {
		log.Error(ctx, "Failed to list drafts", "UserUuid", userUuid, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	response := make([]apimodel.DraftResponse, 0, len(drafts))
	for _, d := range drafts {
		response = append(response, mapDraft(d))
	}

	return respondJSON(c, http.StatusOK, response)
}

// GetDraft returns a single draft by id.
func (h *Handler) GetDraft(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	draftId, ok := parseDraftId(c)
	if !ok {
		return nil
	}

	draftModel, ok := loadDraft(ctx, h, c, draftId)
	if !ok {
		return nil
	}

	if !requireDraftPlayer(c, draftModel, userUuid) {
		return nil
	}

	return respondJSON(c, http.StatusOK, mapDraft(draftModel))
}

// CreateDraft creates a new draft owned by the authenticated user.
func (h *Handler) CreateDraft(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	var req apimodel.CreateDraftRequest
	if err := c.Bind(&req); err != nil {
		api.BadRequest(c.Response(), "Invalid request body")
		return nil
	}

	if req.DisplayName == "" {
		api.BadRequest(c.Response(), "display_name is required")
		return nil
	}
	if req.Interval <= 0 {
		api.BadRequest(c.Response(), "interval must be greater than 0")
		return nil
	}

	draftModel := model.DraftModel{
		Owner: model.User{
			UserUuid: userUuid,
		},
		DisplayName: req.DisplayName,
		Description: req.Description,
		Interval:    req.Interval,
		Status:      model.FILLING,
	}

	draftId, err := h.DraftStore.CreateDraft(ctx, &draftModel)
	if err != nil {
		log.Error(ctx, "Failed to create draft", "UserUuid", userUuid, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	draftModel.Id = draftId
	return respondJSON(c, http.StatusCreated, mapDraft(draftModel))
}

// UpdateDraft updates draft metadata. Only the owner can update.
func (h *Handler) UpdateDraft(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	draftId, ok := parseDraftId(c)
	if !ok {
		return nil
	}

	draftModel, ok := loadDraft(ctx, h, c, draftId)
	if !ok {
		return nil
	}

	if !requireDraftOwner(c, draftModel, userUuid) {
		return nil
	}

	var req apimodel.UpdateDraftRequest
	if err := c.Bind(&req); err != nil {
		api.BadRequest(c.Response(), "Invalid request body")
		return nil
	}

	update := model.DraftModel{
		Id: draftId,
		Owner: model.User{
			UserUuid: userUuid,
		},
	}

	if req.DisplayName != "" {
		update.DisplayName = req.DisplayName
	} else {
		update.DisplayName = draftModel.DisplayName
	}
	if req.Description != "" {
		update.Description = req.Description
	} else {
		update.Description = draftModel.Description
	}
	if req.Interval > 0 {
		update.Interval = req.Interval
	} else {
		update.Interval = draftModel.Interval
	}

	draftActor, err := h.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "DraftId", draftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	if err := draft.UpdateDraft(ctx, draftActor, update); err != nil {
		log.Error(ctx, "Failed to update draft", "DraftId", draftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	return respondJSON(c, http.StatusOK, mapDraft(draft.GetDraft(draftActor)))
}

// StartDraft transitions a draft from FILLING to WAITING_TO_START.
func (h *Handler) StartDraft(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	draftId, ok := parseDraftId(c)
	if !ok {
		return nil
	}

	draftModel, ok := loadDraft(ctx, h, c, draftId)
	if !ok {
		return nil
	}

	if !requireDraftOwner(c, draftModel, userUuid) {
		return nil
	}

	numAccepted := 0
	for _, p := range draftModel.Players {
		if !p.Pending {
			numAccepted++
		}
	}
	if numAccepted != 8 {
		api.BadRequest(c.Response(), "Draft must have exactly 8 accepted players to start")
		return nil
	}

	if err := h.DraftStore.CancelOutstandingInvites(ctx, draftId); err != nil {
		log.Error(ctx, "Failed to cancel outstanding invites", "DraftId", draftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	draftActor, err := h.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "DraftId", draftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	if err := draft.ExecuteDraftStateTransition(ctx, draftActor, model.WAITING_TO_START); err != nil {
		log.Error(ctx, "Failed to start draft", "DraftId", draftId, "Error", err)
		api.BadRequest(c.Response(), err.Error())
		return nil
	}

	return respondJSON(c, http.StatusOK, mapDraft(draft.GetDraft(draftActor)))
}

// ListPicks returns all picks for a draft.
func (h *Handler) ListPicks(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	draftId, ok := parseDraftId(c)
	if !ok {
		return nil
	}

	draftModel, ok := loadDraft(ctx, h, c, draftId)
	if !ok {
		return nil
	}

	if !requireDraftPlayer(c, draftModel, userUuid) {
		return nil
	}

	picks, err := h.DraftStore.GetPicks(ctx, draftId)
	if err != nil {
		log.Error(ctx, "Failed to get picks", "DraftId", draftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	response := make([]apimodel.PickResponse, 0, len(picks))
	for _, p := range picks {
		response = append(response, mapPick(p))
	}

	return respondJSON(c, http.StatusOK, response)
}

// MakePick submits a pick for the authenticated player.
func (h *Handler) MakePick(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	draftId, ok := parseDraftId(c)
	if !ok {
		return nil
	}

	var req apimodel.MakePickRequest
	if err := c.Bind(&req); err != nil {
		api.BadRequest(c.Response(), "Invalid request body")
		return nil
	}

	if req.TeamNumber <= 0 {
		api.BadRequest(c.Response(), "team_number must be greater than 0")
		return nil
	}

	draftActor, err := h.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor", "DraftId", draftId, "Error", err)
		api.NotFound(c.Response(), "Draft not found")
		return nil
	}

	draftState := draftActor.GetDraftState()
	if draftState.NextPick.User.UserUuid != userUuid {
		api.Forbidden(c.Response(), "It is not your turn to pick")
		return nil
	}

	draftPlayerId, err := h.DraftStore.GetDraftPlayerId(ctx, draftId, userUuid)
	if err != nil {
		log.Warn(ctx, "Failed to get draft player id", "DraftId", draftId, "UserUuid", userUuid, "Error", err)
		api.Forbidden(c.Response(), "You are not a member of this draft")
		return nil
	}

	tbaId := "frc" + strconv.Itoa(req.TeamNumber)
	pick := model.Pick{
		Id: draftState.CurrentPick.Id,
		Player: draftPlayerId,
		Pick: sql.NullString{
			Valid:  true,
			String: tbaId,
		},
		PickTime: sql.NullTime{
			Valid: true,
			Time:  time.Now().UTC(),
		},
	}

	if err := draft.MakePick(ctx, draftActor, pick); err != nil {
		log.Warn(ctx, "Failed to make pick", "DraftId", draftId, "Team", tbaId, "Error", err)
		api.BadRequest(c.Response(), err.Error())
		return nil
	}

	return respondJSON(c, http.StatusOK, mapPick(draft.GetCurrentPick(draftActor)))
}

// ToggleSkip marks whether the authenticated player's picks should be skipped.
func (h *Handler) ToggleSkip(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	draftId, ok := parseDraftId(c)
	if !ok {
		return nil
	}

	var req apimodel.SkipPickRequest
	if err := c.Bind(&req); err != nil {
		api.BadRequest(c.Response(), "Invalid request body")
		return nil
	}

	draftPlayerId, err := h.DraftStore.GetDraftPlayerId(ctx, draftId, userUuid)
	if err != nil {
		log.Warn(ctx, "Failed to get draft player id", "DraftId", draftId, "UserUuid", userUuid, "Error", err)
		api.Forbidden(c.Response(), "You are not a member of this draft")
		return nil
	}

	if err := h.DraftStore.MarkShouldSkipPick(ctx, draftPlayerId, req.Skipping); err != nil {
		log.Error(ctx, "Failed to mark skip pick", "DraftPlayerId", draftPlayerId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	return c.NoContent(http.StatusNoContent)
}

// GetDraftScore returns the current score breakdown for a draft.
func (h *Handler) GetDraftScore(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	draftId, ok := parseDraftId(c)
	if !ok {
		return nil
	}

	draftModel, ok := loadDraft(ctx, h, c, draftId)
	if !ok {
		return nil
	}

	if !requireDraftPlayer(c, draftModel, userUuid) {
		return nil
	}

	players, err := h.DraftStore.GetDraftScore(ctx, draftId)
	if err != nil {
		log.Error(ctx, "Failed to get draft score", "DraftId", draftId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	response := make([]apimodel.DraftPlayerResponse, 0, len(players))
	for _, p := range players {
		response = append(response, mapDraftPlayer(p))
	}

	return respondJSON(c, http.StatusOK, apimodel.DraftScoreResponse{
		DraftId: draftId,
		Status:  mapDraftState(draftModel.Status),
		Players: response,
	})
}

// GetTeamScore returns scores for a specific FRC team.
func (h *Handler) GetTeamScore(c echo.Context) error {
	ctx := c.Request().Context()
	teamNumberStr := c.QueryParam("team_number")
	if teamNumberStr == "" {
		api.BadRequest(c.Response(), "team_number is required")
		return nil
	}

	teamNumber, err := strconv.Atoi(teamNumberStr)
	if err != nil || teamNumber <= 0 {
		api.BadRequest(c.Response(), "team_number must be a positive integer")
		return nil
	}

	tbaId := "frc" + strconv.Itoa(teamNumber)
	scores, err := h.TeamStore.GetScore(ctx, tbaId)
	if err != nil {
		log.Error(ctx, "Failed to get team score", "Team", tbaId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	matches, err := h.TeamStore.GetMatchScores(ctx, tbaId)
	if err != nil {
		log.Error(ctx, "Failed to get match scores", "Team", tbaId, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	matchResponse := make([]apimodel.MatchScore, 0, len(matches))
	for _, m := range matches {
		matchResponse = append(matchResponse, apimodel.MatchScore{
			MatchTbaId: m.MatchTbaId,
			Alliance:   m.Alliance,
			Score:      m.Score,
			IsDqed:     m.IsDqed,
		})
	}

	return respondJSON(c, http.StatusOK, apimodel.TeamScoreResponse{
		TeamNumber: teamNumber,
		Scores:     scores,
		Matches:    matchResponse,
	})
}

// SearchUsers returns users matching the query who are not already in the draft.
func (h *Handler) SearchUsers(c echo.Context) error {
	ctx := c.Request().Context()
	userUuid, ok := getAuthenticatedUser(c)
	if !ok {
		api.Unauthorized(c.Response(), "Authentication required")
		return nil
	}

	draftIdStr := c.QueryParam("draft_id")
	search := c.QueryParam("q")

	draftId := 0
	if draftIdStr != "" {
		var err error
		draftId, err = strconv.Atoi(draftIdStr)
		if err != nil {
			api.BadRequest(c.Response(), "Invalid draft_id")
			return nil
		}
	}

	if draftId == 0 && search == "" {
		api.BadRequest(c.Response(), "draft_id or q is required")
		return nil
	}

	// If a draft id is provided, ensure the requester is the owner.
	if draftId != 0 {
		draftModel, ok := loadDraft(ctx, h, c, draftId)
		if !ok {
			return nil
		}
		if !requireDraftOwner(c, draftModel, userUuid) {
			return nil
		}
	}

	users, err := h.UserStore.SearchUsers(ctx, search, draftId)
	if err != nil {
		log.Error(ctx, "Failed to search users", "DraftId", draftId, "Search", search, "Error", err)
		api.InternalError(c.Response())
		return nil
	}

	response := make([]apimodel.UserSummary, 0, len(users))
	for _, u := range users {
		response = append(response, mapUser(u))
	}

	return respondJSON(c, http.StatusOK, apimodel.UserSearchResponse{Users: response})
}
