package handler

import (
	"database/sql"
	"server/logging"
)

type Handler struct {
    Database *sql.DB
    Logger *logging.Logger
}

