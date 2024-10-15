package handler

import (
	"database/sql"
	"server/logging"
	"server/tbaHandler"
)

type Handler struct {
    Database *sql.DB
    TbaHandler tbaHandler.TbaHandler
    Logger *logging.Logger
}

