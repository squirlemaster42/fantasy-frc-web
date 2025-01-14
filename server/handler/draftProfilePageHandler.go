package handler

import (
	"fmt"
	"server/assert"
	"server/model"
	draftView "server/view/draft"
	"strconv"

	"github.com/labstack/echo/v4"
)

func (h *Handler) HandleViewDraftProfile(c echo.Context) error {
    h.Logger.Log("Got a request to serve the draft profile page")
    assert := assert.CreateAssertWithContext("Handle update Draft Profile")

    userTok, err := c.Cookie("sessionToken")
    //TODO We should have already checked that the user has a token
    //here since they should not be able to access the page otherwise
    //There might be some sort of weird thing here where the middleware
    //validates the session token is good and then it expires a second later
    assert.NoError(err, "Failed to get user token")

    userId := model.GetUserBySessionToken(h.Database, userTok.Value)
    username := model.GetUsername(h.Database, userId)

    draftId, err := strconv.Atoi(c.Param("id"))
    assert.NoError(err, "Failed to convert draft id to int")
    draftModel := model.GetDraft(h.Database, draftId)

    draftIndex := draftView.DraftProfileIndex(draftModel)
    draftView := draftView.DraftProfile(" | Draft Profile", true, username, draftIndex)
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
    searchInput := c.FormValue("search")
    h.Logger.Log("Got request to search users")
    users := model.SearchUsers(h.Database, searchInput)

    searchResults := draftView.PlayerSearchResults(users)
    err := Render(c, searchResults)

    return err
}
