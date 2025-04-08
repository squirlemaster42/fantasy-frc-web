package handler

import (
	"server/assert"
	"server/model"
	"server/view/draft"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleDraftScore(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Handle Draft Score")
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")

	userId := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userId)

	draftId, err := strconv.Atoi(c.Param("id"))
	assert.NoError(err, "Failed to convert draft id to int")

	userDraftScore := model.GetDraftScore(h.Database, draftId)

	draftIndex := draft.DraftScoreIndex(userDraftScore)
	draft := draft.DraftScore(" | Draft Score", true, username, draftIndex, draftId)
	Render(c, draft)
	return nil
}
