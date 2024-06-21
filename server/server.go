package main

import (
	"database/sql"
	"fmt"
	"server/assert"
	"server/handler"

	"github.com/labstack/echo/v4"
)

func CreateServer(database *sql.DB) {
    fmt.Println("Creating Server")
    assert := assert.CreateAssertWithContext("Create Server")
    app := echo.New()
    app.Static("/", "./assets")

    //Setup Routes
    app.GET("/", handler.HandleViewLogin)

    err := app.Start(":3000")
    assert.NoError(err, "Failed to start server")

}
