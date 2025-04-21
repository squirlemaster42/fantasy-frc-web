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
    Notifier *picking.PickNotifier
    DraftDaemon *background.DraftDaemon
}

