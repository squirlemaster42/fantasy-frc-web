package web

import (
	"cmp"
	"fmt"
	"io"
	"net/http"
	scoring "server/scoring"
	"server/web/handler"
	"slices"

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

func CreateServer(scorer *scoring.Scorer) {
    app := echo.New()

    loginHandler := handler.LoginHandler{}
    app.GET("/login", loginHandler.HandleLoginShow)

    app.Start(":3000")
    fmt.Println("Started Web Server On Port 3000")


    /*
    mux := http.NewServeMux()
    server := server{scorer: scorer}
    mux.HandleFunc("/scores", server.getScores)

    err := http.ListenAndServe(":3333", mux)
    if errors.Is(err, http.ErrServerClosed) {
        fmt.Printf("server closed\n")
    } else if err != nil {
        fmt.Printf("errors starting server: %s\n", err)
        os.Exit(1)
    }
    */
}

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
