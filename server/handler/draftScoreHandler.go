package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"server/log"
	"server/model"
	"server/types"
	"server/view/draft"
	"server/view/team"
	"slices"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleDraftScore(c echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to convert draft id to int", "draftIdString", c.Param("id"), "error", err)
		return c.String(http.StatusBadRequest, "Draft id was not an int")
	}

	draftModel, err := h.Stores.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn(c.Request().Context(), "Draft not found", "draftId", draftId)
			return c.String(http.StatusNotFound, fmt.Sprintf("Failed to load draft id %d", draftId))
		}
		log.Error(c.Request().Context(), "Failed to get draft by id", "draftId", draftId, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	isOwner := draftModel.Owner.UserUuid == userUuid

	userDraftScore, err := h.Stores.DraftStore.GetDraftScore(c.Request().Context(), draftId)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get draft score", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	slices.SortFunc(userDraftScore, func(a, b model.DraftPlayer) int {
		return b.Score - a.Score
	})

	for _, draftPlayer := range userDraftScore {
		slices.SortFunc(draftPlayer.Picks, func(a, b model.Pick) int {
			return b.Score - a.Score
		})
	}

	avatarColors := collectDraftAvatarColors(c.Request().Context(), h.Services.AvatarStore, userDraftScore)

	draftIndex := draft.DraftScoreIndex(userDraftScore, draftId, draftModel.Status, avatarColors)
	draftView := draft.DraftScore("Draft Scores", true, username, draftIndex, types.NewPageData(draftId, draftModel.DisplayName, isOwner))
	if err := Render(c, draftView); err != nil {
		log.Error(c.Request().Context(), "Failed to render draft score page", "draftId", draftId, "error", err)
		return err
	}
	return nil
}

func (h *Handler) HandleDraftTeamScore(c echo.Context) error {
	userUuid, username, err := h.requireUser(c)
	if err != nil {
		return err
	}

	draftId, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		log.Warn(c.Request().Context(), "Failed to convert draft id to int", "draftIdString", c.Param("id"), "error", err)
		return c.String(http.StatusBadRequest, "Draft id was not an int")
	}

	draftModel, err := h.Stores.DraftStore.GetDraft(c.Request().Context(), draftId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Warn(c.Request().Context(), "Draft not found", "draftId", draftId)
			return c.String(http.StatusNotFound, fmt.Sprintf("Failed to load draft id %d", draftId))
		}
		log.Error(c.Request().Context(), "Failed to get draft by id", "draftId", draftId, "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	isOwner := draftModel.Owner.UserUuid == userUuid

	teamNumber := c.Param("teamNumber")

	scores, err := h.Stores.TeamStore.GetScore(c.Request().Context(), teamPrefix+teamNumber)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get team score", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	// Get qualification matches
	qualificationMatches, err := h.Stores.TeamStore.GetMatchScores(c.Request().Context(), teamPrefix+teamNumber)
	if err != nil {
		log.Error(c.Request().Context(), "Failed to get match scores", "error", err)
		return c.String(http.StatusInternalServerError, "An error occurred")
	}

	avatarColor := avatarColorForTeam(c.Request().Context(), h.Services.AvatarStore, teamNumber)

	teamScoreReport := team.TeamScoreReport(teamNumber, scores, qualificationMatches, avatarColor)
	draftTeamScore := draft.DraftTeamScore(" | Score Breakdown", true, username, teamScoreReport, types.NewPageData(draftId, draftModel.DisplayName, isOwner))
	if err := Render(c, draftTeamScore); err != nil {
		log.Error(c.Request().Context(), "Failed to render draft team score page", "draftId", draftId, "teamNumber", teamNumber, "error", err)
		return err
	}
	return nil
}
