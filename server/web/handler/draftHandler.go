package handler

import (
	"fmt"
	"log"
	"net/http"
	"server/database"
	"server/web/model"
	"server/web/view/draft"
	"strconv"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type DraftHandler struct {
    DbDriver *database.DatabaseDriver
}

func (d *DraftHandler) HandleViewDraft (c echo.Context) error {
    ses, err := session.Get("session", c)

    if err != nil {
        return err
    }

    sessionToken := ses.Values["token"].(string)
    userId := ses.Values["userId"].(int)

    sessionHandler := SessionHandler{DbHandler: d.DbDriver}
    player, err := model.GetPlayerById(userId, *d.DbDriver)
    isValid := sessionHandler.validateSession(player.Id, sessionToken)

    if err != nil {
        log.Fatalln(err)
    }

    if !isValid {
        log.Println("Invalid Login Detected")
    }

    draftIdParam := c.QueryParam("draftId")

    if draftIdParam == "" {
        return echo.NewHTTPError(http.StatusNotFound, "This draft does not exist")
    }

    draftId, err := strconv.Atoi(draftIdParam)

    //TODO We need to do permission checks here
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "You must specify a draft id")
    }

    if !model.CheckIfDraftExists(draftId, d.DbDriver) {
        return echo.NewHTTPError(http.StatusNotFound, "This draft does not exist")
    }

    draftModel := model.LoadDraftFromDatabase(draftId, d.DbDriver)
    draftIndex := draft.DraftPickIndex(*draftModel)
    draftView := draft.DraftPick(" | Draft", false, draftIndex)

    if c.Request().Method == "POST" {
        pick := "tba" + c.FormValue("pickInput")
        fmt.Println(pick)
        // Validate that the pick is valid (not duplicated and at valid events)
        // Find the pick order and player order and make the pick
        team := model.GetTeam(pick, d.DbDriver)

        if team.Name != "" && team.ValidPick {

        } else {
            log.Println("Invalid Pick")
        }

        err = render(c, draftIndex)
    } else {
        err = render(c, draftView)
    }

    return err
}
