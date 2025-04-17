package handler

import (
	"server/assert"
	"server/model"
	"server/view/draft"
	"slices"
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

    //Sort user scores
    slices.SortFunc(userDraftScore, func(a, b model.DraftPlayer) int {
        //This will return a negative number when a score is less than b score
        //a positive number is a score is greater than b score
        //and 0 if they are equal
        return a.Score - b.Score
    })

    for _, draftPlayer := range userDraftScore {
        //Sort team scores
        slices.SortFunc(draftPlayer.Picks, func(a, b model.Pick) int {
            //This will return a negative number when a score is less than b score
            //a positive number is a score is greater than b score
            //and 0 if they are equal
            return a.Score - b.Score
        })
    }

	draftIndex := draft.DraftScoreIndex(userDraftScore)
	draft := draft.DraftScore(" | Draft Score", true, username, draftIndex, draftId)
	Render(c, draft)
	return nil
}
