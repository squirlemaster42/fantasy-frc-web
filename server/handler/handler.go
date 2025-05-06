package handler

import (
	"database/sql"
	"server/background"
	"server/draft"
	"server/tbaHandler"
)

type Handler struct {
    Database *sql.DB
    TbaHandler tbaHandler.TbaHandler
    DraftManager *draft.DraftManager
    DraftDaemon *background.DraftDaemon
}

