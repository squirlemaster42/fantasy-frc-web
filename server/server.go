package main

import (
	"log/slog"
	"server/assert"
	"server/authentication"
	"server/handler"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func CreateServer(serverPort string, h handler.Handler) {
    slog.Info("Starting Server")
    assert := assert.CreateAssertWithContext("Create Server")
    auth := authentication.NewAuth(h.Database)
    app := echo.New()
    app.IPExtractor = echo.ExtractIPDirect()
    app.Static("/", "./assets")
    app.Static("/css", "./assets/css")

    app.Use(middleware.Gzip())
    //app.Use(middleware.Recover())

    //Setup Routes
    app.GET("/login", h.HandleViewLogin)
    app.POST("/login", h.HandleLoginPost)
    app.GET("/register", h.HandleViewRegister)
    app.POST("/register", h.HandlerRegisterPost)
    app.POST("/logout", h.HandleLogoutPost)
    app.POST("/tbaWebhook", h.ConsumeTbaWebsocket)

    protected := app.Group("/u", auth.Authenticate)
    protected.GET("/home", h.HandleViewHome)
    protected.GET("/createDraft", h.HandleViewCreateDraft)
    protected.POST("/createDraft", h.HandleCreateDraftPost)
    protected.GET("/draft/:id/profile", h.HandleViewDraftProfile)
    protected.POST("/draft/:id/updateDraft", h.HandleUpdateDraftProfile)
    protected.GET("/draft/:id/pick", h.ServePickPage)
    protected.POST("/draft/:id/makePick", h.HandlerPickRequest)
    protected.GET("/draft/:id/pickNotifier", h.PickNotifier)
    protected.POST("/draft/:id/invitePlayer", h.InviteDraftPlayer)
    protected.GET("/team/score", h.HandleTeamScore)
    protected.POST("/team/score", h.HandleGetTeamScore)
    protected.GET("/draft/:id/draftScore", h.HandleDraftScore)
    protected.POST("/searchPlayers", h.SearchPlayers)
    protected.GET("/viewInvites", h.HandleViewInvites)
    protected.POST("/acceptInvite", h.HandleAcceptInvite)
    protected.POST("/draft/:id/skipPickToggle", h.HandleSkipPickToggle)

    admin := protected.Group("/admin", auth.CheckAdmin)
    admin.GET("/console", h.HandleAdminConsoleGet)
    admin.POST("/processCommand", h.HandleRunCommand)

    err := app.Start(":" + serverPort)
    assert.NoError(err, "Failed to start server")
}
