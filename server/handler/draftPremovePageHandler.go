package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"server/log"
	"server/model"
	"server/types"
	draftView "server/view/draft"
	"strconv"
	"strings"

	"uuid"
	"github.com/labstack/echo/v5"
)

// ServePremovePage renders the premove editor for the authenticated user.
func (h *Handler) ServePremovePage(c *echo.Context) error {
	log.Debug(c.Request().Context(), "Serving premove page", "ip", c.RealIP())
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}
	draftId, err := parseDraftId(c)
	if err != nil {
		return err
	}

	return h.renderPremovePage(c, draftId, userUuid, username, "")
}

// HandleSavePremoves replaces the authenticated player's pending premove queue
// with the ordered teams submitted by the editor.
func (h *Handler) HandleSavePremoves(c *echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}
	draftId, err := parseDraftId(c)
	if err != nil {
		return err
	}

	if err := c.Request().ParseForm(); err != nil {
		log.Warn(c.Request().Context(), "Failed to parse premove form", "draftId", draftId, "error", err)
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid form data")
	}

	beacon := c.Request().PostFormValue("beacon") == "1"
	action := c.Request().PostFormValue("action")

	var teams []string
	if action != "clear" {
		teams, err = normalizePremoveTeams(c.Request().PostForm["premoves"])
		if err != nil {
			log.Warn(c.Request().Context(), "Invalid premove queue", "draftId", draftId, "error", err)
			if beacon {
				return c.NoContent(http.StatusBadRequest)
			}
			return h.renderPremovePage(c, draftId, userUuid, username, err.Error())
		}
	}

	draftPlayerId, err := h.Stores.DraftStore.GetDraftPlayerId(c.Request().Context(), draftId, userUuid)
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to get draft player for premoves", "draftId", draftId, "userUuid", userUuid, "error", err)
		return echo.NewHTTPError(http.StatusForbidden, "You are not a player in this draft")
	}

	unavailableTeams, err := h.Stores.DraftStore.ReplacePremoves(c.Request().Context(), draftPlayerId, teams)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to save premoves", "draftId", draftId, "draftPlayerId", draftPlayerId, "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save premoves")
	}

	if len(unavailableTeams) > 0 {
		log.Warn(c.Request().Context(), "Some premove teams are not available", "draftId", draftId, "draftPlayerId", draftPlayerId, "teams", unavailableTeams)
	}

	if beacon {
		return c.NoContent(http.StatusNoContent)
	}

	redirectUrl := fmt.Sprintf("/u/draft/%d/premoves?saved=1", draftId)
	if len(unavailableTeams) > 0 {
		redirectUrl += "&unavailable=" + url.QueryEscape(strings.Join(unavailableTeams, ","))
	}

	return c.Redirect(http.StatusSeeOther, redirectUrl)
}

func (h *Handler) renderPremovePage(c *echo.Context, draftId int, userUuid uuid.UUID, username string, saveError string) error {
	ctx := c.Request().Context()

	draftActor, err := h.Services.DraftActorMap.GetActor(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to get draft actor for premove page", "draftId", draftId, "error", err)
		return err
	}
	draftState := draftActor.GetDraftState()

	draftPlayerId, err := h.Stores.DraftStore.GetDraftPlayerId(ctx, draftId, userUuid)
	if err != nil {
		log.Warn(ctx, "Failed to get draft player for premove page", "draftId", draftId, "userUuid", userUuid, "error", err)
		return echo.NewHTTPError(http.StatusForbidden, "You are not a player in this draft")
	}

	premoves, err := h.Stores.DraftStore.GetPremoves(ctx, draftPlayerId)
	if err != nil {
		log.Error(ctx, "Failed to load premoves", "draftId", draftId, "draftPlayerId", draftPlayerId, "error", err)
		return err
	}

	pickedTeams := make([]string, 0)
	pickedSet := make(map[string]struct{})
	picks, err := h.Stores.DraftStore.GetPicks(ctx, draftId)
	if err != nil {
		log.Warn(ctx, "Failed to load picks for premove page", "draftId", draftId, "error", err)
	} else {
		for _, pick := range picks {
			if !pick.Pick.Valid {
				continue
			}
			if _, exists := pickedSet[pick.Pick.String]; exists {
				continue
			}
			pickedSet[pick.Pick.String] = struct{}{}
			pickedTeams = append(pickedTeams, pick.Pick.String)
		}
	}

	pendingPremoves := make([]model.Premove, 0, len(premoves))
	madePremoves := make([]model.Premove, 0, len(premoves))
	pickedPremoves := make([]model.Premove, 0, len(premoves))
	for _, premove := range premoves {
		if !premove.IsPending() {
			if premove.IsMade() {
				madePremoves = append(madePremoves, premove)
			}
			continue
		}

		if _, picked := pickedSet[premove.TeamTbaId]; picked {
			pickedPremoves = append(pickedPremoves, premove)
			continue
		}

		pendingPremoves = append(pendingPremoves, premove)
	}

	// Completed premoves can never be queued again. Include them in the
	// unavailable set so older cached clients also reject them locally.
	for _, premove := range madePremoves {
		if _, exists := pickedSet[premove.TeamTbaId]; exists {
			continue
		}
		pickedSet[premove.TeamTbaId] = struct{}{}
		pickedTeams = append(pickedTeams, premove.TeamTbaId)
	}
	for _, premove := range pickedPremoves {
		if _, exists := pickedSet[premove.TeamTbaId]; exists {
			continue
		}
		pickedSet[premove.TeamTbaId] = struct{}{}
		pickedTeams = append(pickedTeams, premove.TeamTbaId)
	}

	pendingCount := len(pendingPremoves)

	validTeams := make([]string, 0)
	teams, err := h.Stores.TeamStore.GetTeams(ctx)
	if err != nil {
		log.Warn(ctx, "Failed to load teams for premove page", "draftId", draftId, "error", err)
	} else {
		validTeams = make([]string, 0, len(teams))
		for _, team := range teams {
			validTeams = append(validTeams, strings.TrimPrefix(team.TbaId, teamPrefix))
		}
	}

	pageModel := draftView.PremovePage{
		Draft:          draftState,
		Premoves:       pendingPremoves,
		MadePremoves:   madePremoves,
		PickedPremoves: pickedPremoves,
		ValidTeams:     validTeams,
		PickedTeams:    pickedTeams,
		PendingCount:   pendingCount,
		SaveUrl:        fmt.Sprintf("/u/draft/%d/premoves", draftId),
		MaxPremoves:    model.PicksPerPlayer,
		Saved:          c.QueryParam("saved") == "1",
		SaveError:      saveError,
		SaveWarning:    unavailablePremoveWarning(c.QueryParam("unavailable")),
	}

	pageIndex := draftView.DraftPremoveIndex(pageModel, h.csrfToken(c))
	isOwner := draftState.Owner.UserUuid == userUuid
	pageView := draftView.DraftPremove("Draft Premoves", true, username, pageIndex, types.NewPageData(draftId, draftState.DisplayName, isOwner))
	if err := Render(c, pageView); err != nil {
		log.Error(ctx, "Failed to render premove page", "draftId", draftId, "error", err)
		return err
	}

	return nil
}

func unavailablePremoveWarning(rawTeams string) string {
	if rawTeams == "" {
		return ""
	}

	numbers := make([]string, 0)
	for _, team := range strings.Split(rawTeams, ",") {
		team = strings.TrimSpace(team)
		if team == "" {
			continue
		}
		numbers = append(numbers, strings.TrimPrefix(team, teamPrefix))
	}
	if len(numbers) == 0 {
		return ""
	}

	return fmt.Sprintf("The following teams are not available in this draft: %s.", strings.Join(numbers, ", "))
}

func parseDraftId(c *echo.Context) (int, error) {
	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to parse draft id string", "draftIdString", c.Param("id"), "error", err)
		return 0, fmt.Errorf("failed to parse draft id: %w", err)
	}
	return draftId, nil
}

func normalizePremoveTeams(values []string) ([]string, error) {
	if len(values) > model.PicksPerPlayer {
		return nil, fmt.Errorf("you can queue up to %d premoves", model.PicksPerPlayer)
	}

	teams := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		team, err := normalizeTeamNumber(value)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[team]; exists {
			return nil, fmt.Errorf("team %s is already in your premoves", strings.TrimPrefix(team, teamPrefix))
		}
		seen[team] = struct{}{}
		teams = append(teams, team)
	}

	return teams, nil
}

func normalizeTeamNumber(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	trimmed = strings.TrimPrefix(trimmed, teamPrefix)
	if trimmed == "" {
		return "", errors.New("enter a team number")
	}

	for _, r := range trimmed {
		if r < '0' || r > '9' {
			return "", fmt.Errorf("invalid team number: %s", value)
		}
	}

	number, err := strconv.Atoi(trimmed)
	if err != nil || number <= 0 {
		return "", fmt.Errorf("invalid team number: %s", value)
	}

	return fmt.Sprintf("%s%d", teamPrefix, number), nil
}
