package main

import (
	"database/sql"
	"server/assert"
	"server/handler"

	"github.com/labstack/echo/v4"
)

func CreateServer(database *sql.DB) {
    assert := assert.CreateAssertWithContext("Create Server")
    app := echo.New()
    app.Static("/", "./assets")

    //Setup Routes
    app.GET("/login", handler.HandleViewLogin)
    app.POST("/login", handler.HandleLoginPost)

    err := app.Start(":3000")
    assert.NoError(err, "Failed to start server")

}
