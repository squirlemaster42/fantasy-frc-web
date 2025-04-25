package handler

import (
	"database/sql"
	"server/background"
	"server/picking"
	"server/tbaHandler"
)

type Handler struct {
    Database *sql.DB
    TbaHandler tbaHandler.TbaHandler
    //TODO I think we can remove this when we shift to the draftPickManager
    Notifier *picking.PickNotifier
    DraftPickManager *picking.DraftPickManager
    DraftDaemon *background.DraftDaemon
}

