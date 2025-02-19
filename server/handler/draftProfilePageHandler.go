package handler

import (
	"fmt"
	"server/assert"
	"server/model"
	draftView "server/view/draft"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4")

func (h *Handler) HandleViewDraftProfile(c echo.Context) error {
    //TODO Fix: Players are not shown when page is opened from home screen
	h.Logger.Log("Got a request to serve the draft profile page")
	assert := assert.CreateAssertWithContext("Handle update Draft Profile")

	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")

	userId := model.GetUserBySessionToken(h.Database, userTok.Value)
	username := model.GetUsername(h.Database, userId)

	draftId, err := strconv.Atoi(c.Param("id"))
	assert.NoError(err, "Failed to convert draft id to int")
	draftModel := model.GetDraft(h.Database, draftId)

	draftIndex := draftView.DraftProfileIndex(draftModel)
	draftView := draftView.DraftProfile(" | Draft Profile", true, username, draftIndex, draftId)
	err = Render(c, draftView)
	return nil
}

func (h *Handler) HandleUpdateDraftProfile(c echo.Context) error {
	//TODO We need to update the draft settings
	file, err := c.FormFile("profiePic")
	if err != nil {
		fmt.Println(err)
		return err
	}
	src, err := file.Open()
	fmt.Println(src)
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer src.Close()

	return nil
}

func (h *Handler) SearchPlayers(c echo.Context) error {
	splitSource := strings.Split(c.Request().Header["Hx-Current-Url"][0], "/")
	draftId, err := strconv.Atoi(splitSource[len(splitSource)-2])
	assert.NoErrorCF(err, "Failed to parse draft Id")
	searchInput := c.FormValue("search")
	h.Logger.Log("Got request to search users")

	users := model.SearchUsers(h.Database, searchInput, draftId)

	searchResults := draftView.PlayerSearchResults(users, draftId)
	err = Render(c, searchResults)
	return err
}

func (h *Handler) InviteDraftPlayer(c echo.Context) error {
	assert := assert.CreateAssertWithContext("Invite Draft Player")
	userTok, err := c.Cookie("sessionToken")
	assert.NoError(err, "Failed to get user token")
	draftIdStr := c.Param("id")
	invitingPlayer := model.GetUserBySessionToken(h.Database, userTok.Value)
	draftId, err := strconv.Atoi(draftIdStr)
	assert.NoError(err, "Invalid draft id")
	userIdStr := c.FormValue("userId")
	userId, err := strconv.Atoi(userIdStr)
	assert.NoError(err, "Failed to parse user id")

	model.InvitePlayer(h.Database, draftId, invitingPlayer, userId)

	assert.NoError(err, "Failed to parse draft Id")
	searchInput := c.FormValue("search")
	h.Logger.Log("Got request to search users")

	users := model.SearchUsers(h.Database, searchInput, draftId)

    players := model.GetDraft(h.Database, draftId).Players

	updatedPage := draftView.UpdateAfterInvite(users, draftId, players)
	err = Render(c, updatedPage)

	return err
}
