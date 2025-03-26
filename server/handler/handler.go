package handler

import (
	"database/sql"
	"server/tbaHandler"
)

type Handler struct {
    Database *sql.DB
    TbaHandler tbaHandler.TbaHandler
    Notifier *PickNotifier
}

