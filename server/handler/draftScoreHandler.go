package handler

import (
	"database/sql"
	"errors"
	"fmt"
	"server/model"
	"server/types"
	"server/view/draft"
	"server/view/team"
	"slices"

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
	return RenderOrLog(c, draftView, "Failed to render draft score page", "draftId", draftId)
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
	draftTeamScore := draft.DraftTeamScore(" | Score Breakdown", true, username, teamScoreReport, types.NewPageData(draftModel.Id, draftModel.DisplayName, isOwner))
	return RenderOrLog(c, draftTeamScore, "Failed to render draft team score page", "draftId", draftModel.Id, "teamNumber", teamNumber)
}
