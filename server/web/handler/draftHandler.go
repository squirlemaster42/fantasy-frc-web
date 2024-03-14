package handler

import (
	"server/database"
	"server/web/model"
	"server/web/view/draft"

	"github.com/labstack/echo/v4"
)

type DraftHandler struct {
    DbDriver *database.DatabaseDriver
}

func (d *DraftHandler) HandleViewDraft (c echo.Context) error {
    draftModel := model.Draft{
        Name: "Test Draft",
        Players: []struct {
            Name  string
            Picks []string
        }{
            {
                Name: "Player 1",
                Picks: []string{"1699", "254", "2168", "1114", "1234", "610", "4414", "1678"},
            },
            {
                Name: "Player 2",
                Picks: []string{"1699", "254", "2168", "1114", "1234", "610", "4414", "1678"},
            },
            {
                Name: "Player 3",
                Picks: []string{"1699", "254", "2168", "1114", "1234", "610", "4414", "1678"},
            },
            {
                Name: "Player 4",
                Picks: []string{"1699", "254", "2168", "1114", "1234", "610", "4414", "1678"},
            },
            {
                Name: "Player 5",
                Picks: []string{"1699", "254", "2168", "1114", "1234", "610", "4414", "1678"},
            },
            {
                Name: "Player 6",
                Picks: []string{"1699", "254", "2168", "1114", "1234", "610", "4414", "1678"},
            },
            {
                Name: "Player 7",
                Picks: []string{"1699", "254", "2168", "1114", "1234", "610", "4414", "1678"},
            },
            {
                Name: "Player 8",
                Picks: []string{"1699", "254", "2168", "1114", "1234", "610", "4414", "1678"},
            },
        },
    }

    draftIndex := draft.DraftPickIndex(draftModel)
    draftView := draft.DraftPick(" | Draft", false, draftIndex)
    err := render(c, draftView)
    return err
}
