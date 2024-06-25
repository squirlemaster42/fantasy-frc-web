package main

import (
	"database/sql"
	"server/assert"
	"server/handler"
	"server/logging"

	"github.com/labstack/echo/v4"
)

func CreateServer(database *sql.DB, logger *logging.Logger) {
    assert := assert.CreateAssertWithContext("Create Server")
    app := echo.New()
    app.Static("/", "./assets")

    //Setup Routes
    h := handler.Handler{
        Database: database,
        Logger: logger,
    }
    app.GET("/login", h.HandleViewLogin)
    app.POST("/login", h.HandleLoginPost)

    err := app.Start(":3000")
    assert.NoError(err, "Failed to start server")

}
