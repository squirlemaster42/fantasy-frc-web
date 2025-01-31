package main

import (
	"database/sql"
	"server/assert"
	"server/authentication"
	"server/handler"
	"server/logging"
	"server/tbaHandler"

	"github.com/labstack/echo/v4"
)

func CreateServer(db *sql.DB, tbaHandler *tbaHandler.TbaHandler, logger *logging.Logger) {
    logger.Log("Starting Server")
	assert := assert.CreateAssertWithContext("Create Server")
    auth := authentication.NewAuth(db, logger)
	app := echo.New()
    app.IPExtractor = echo.ExtractIPDirect()
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
    app.GET("/", h.HandleViewHome)
	app.GET("/login", h.HandleViewLogin)
	app.POST("/login", h.HandleLoginPost)
	app.GET("/register", h.HandleViewRegister)
	app.POST("/register", h.HandlerRegisterPost)

    protected := app.Group("u", auth.Authenticate)
	protected.GET("/home", h.HandleViewHome)
	protected.GET("/createDraft", h.HandleViewCreateDraft)
    protected.POST("/createDraft", h.HandleCreateDraftPost)
	protected.GET("/draft/:id/profile", h.HandleViewDraftProfile)
	protected.POST("/draft/updateDraft", h.HandleUpdateDraftProfile)
	protected.GET("/draft/:id/pick", h.ServePickPage)
	protected.POST("/draft/:id/makePick", h.HandlerPickRequest)
	protected.GET("/draft/:id/pickNotifier", h.PickNotifier)
    protected.POST("/draft/:id/invitePlayer", h.InviteDraftPlayer)
    protected.GET("/team/score", h.HandleTeamScore)
    protected.POST("/team/score", h.HandleGetTeamScore)
    protected.POST("/searchPlayers", h.SearchPlayers)
    protected.GET("/viewInvites", h.HandleViewInvites)
    protected.POST("/acceptInvite", h.HandleAcceptInvite)

	err := app.Start(":3000")
	assert.NoError(err, "Failed to start server")
}
