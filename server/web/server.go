package web

import (
	"cmp"
	"fmt"
	"io"
	"net/http"
	scoring "server/scoring"
	"server/web/handler"
	"slices"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type server struct {
    scorer *scoring.Scorer
}

type Draft struct {
    name string
    Players map[string]*Player
}

type Player struct {
    name string
    totalScore int
    picks []string
}

func CreateServer(scorer *scoring.Scorer, sessionSecret string) {
    app := echo.New()
    app.Static("/", "./assets")
    app.Use(session.Middleware(sessions.NewCookieStore([]byte(sessionSecret))))

    //Setup Session Store

    //Setup base routes
    homeHandler := handler.HomeHandler{}
    app.GET("/", homeHandler.HandleViewHome)
    loginHandler := handler.LoginHandler{
        DbHandler: scorer.DbDriver,
    }
    app.GET("/login", loginHandler.HandleViewLogin)
    app.POST("/login", loginHandler.HandleViewLogin)
    registrationHandler := handler.RegistrationHandler{
        DbHandler: scorer.DbDriver,
    }
    app.GET("/register", registrationHandler.HandleViewRegister)
    app.POST("/register", registrationHandler.HandleViewRegister)

    //Setup protected routes
    //draftGroup := app.Group("/draft", )
    draftHandlder := handler.DraftHandler{DbDriver: scorer.DbDriver}
    app.GET("/draft", draftHandlder.HandleViewDraft)
    app.POST("/draft", draftHandlder.HandleViewDraft)

    //Start Setver
    app.Start(":3000")
    fmt.Println("Started Web Server On Port 3000")
}



//TODO Move this code to a handler
func (s *server) getScores (w http.ResponseWriter, r *http.Request) {
    fmt.Printf("got scores request\n")
    io.WriteString(w, "Welcome to FantasyFRC\n")
    for _, draft := range getDrafts(s.scorer) {
        io.WriteString(w, fmt.Sprintf("-------- %s --------\n", draft.name))
        for i, player := range sortPlayersByScore(draft.Players) {
            io.WriteString(w, fmt.Sprintf("%d. %s scored %d points\n", i + 1, player.name, player.totalScore))
        }
    }
}

func sortPlayersByScore(players map[string]*Player) []*Player{
    var playerList []*Player
    for _, player := range players {
        playerList = append(playerList, player)
    }
    slices.SortFunc(playerList, func(a, b *Player) int {
        return cmp.Compare(b.totalScore, a.totalScore)
    })
    return playerList
}

func getDrafts(s *scoring.Scorer) map[int]*Draft {
    drafts := make(map[int]*Draft)

    driver := s.DbDriver
    rows := driver.RunQuery(`
    Select
    d.Id as DraftId,
    d.Name As DraftName,
    pl.Name As PlayerName,
    p.pickedTeam
    From Drafts d
    Left Join Picks p On p.draftId = d.Id
    Left Join Players pl On pl.Id = p.player
    `)
    defer rows.Close()

    if rows == nil {
        return drafts
    }

    for rows.Next() {
        var draftId int
        var draftName string
        var playerName string
        var pickedTeam string

        err := rows.Scan(&draftId, &draftName, &playerName, &pickedTeam)
        if err != nil {
            return drafts
        }

        draft := drafts[draftId]
        if draft == nil {
            draft = &Draft{name: draftName}
        }

        if draft.Players == nil {
            draft.Players = make(map[string]*Player)
        }

        player := draft.Players[playerName]
        if player == nil {
            picks := make([]string, 8)
            picks[0] = pickedTeam
            player := Player{name: playerName, totalScore: s.ScoreTeam(pickedTeam), picks: picks}
            draft.Players[playerName] = &player
        } else {
            draft.Players[playerName].picks = append(draft.Players[playerName].picks, pickedTeam)
            draft.Players[playerName].totalScore += s.ScoreTeam(pickedTeam)
        }

        drafts[draftId] = draft
    }

    return drafts
}
