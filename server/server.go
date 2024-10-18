package main

import (
	"database/sql"
	"server/assert"
	"server/handler"
	"server/logging"
	"server/tbaHandler"

	"github.com/labstack/echo/v4"
)

func CreateServer(db *sql.DB, tbaHandler *tbaHandler.TbaHandler, logger *logging.Logger) {
	assert := assert.CreateAssertWithContext("Create Server")
	app := echo.New()
	app.Static("/", "./assets")

	//Setup Routes
	h := handler.Handler{
		Database:   db,
		Logger:     logger,
		TbaHandler: *tbaHandler,
        Notifier: &handler.PickNotifier{
            Database: db,
            Watchers: make(map[int][]handler.Watcher),
        },
	}
	app.GET("/login", h.HandleViewLogin)
	app.POST("/login", h.HandleLoginPost)
	app.GET("/register", h.HandleViewRegister)
	app.POST("/register", h.HandlerRegisterPost)
	app.GET("/home", h.HandleViewHome)
	app.GET("/createDraft", h.HandleViewCreateDraft)
	app.POST("/createDraft", h.HandleCreateDraftPost)
	app.GET("draft/:id/profile", h.HandleViewDraftProfile)
	app.POST("draft/updateDraft", h.HandleUpdateDraftProfile)
	app.GET("draft/:id/pick", h.ServePickPage)
	app.POST("draft/:id/makePick", h.HandlerPickRequest)
	app.GET("draft/:id/pickNotifier", h.PickNotifier)

	err := app.Start(":3000")
	assert.NoError(err, "Failed to start server")

}
