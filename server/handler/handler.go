package handler

import (
	"database/sql"
	"server/background"
	"server/notifiers"
	"server/tbaHandler"
)

type Handler struct {
    Database *sql.DB
    TbaHandler tbaHandler.TbaHandler
    Notifier *notifiers.PickNotifier
    DraftDaemon *background.DraftDaemon
}

