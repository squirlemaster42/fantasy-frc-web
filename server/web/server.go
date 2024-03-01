package web

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
    scoring "server/scoring"
)

type server struct {
    scorer *scoring.Scorer
}

type Draft struct {
    name string
    Players map[string]*Player
}

type Player struct {
    totalScore int
    picks []string
}

func CreateServer(scorer *scoring.Scorer) {
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
}

func (s *server) getScores (w http.ResponseWriter, r *http.Request) {
    fmt.Printf("got / request\n")
    io.WriteString(w, "This is another line\n")
    for _, draft := range getDrafts(s.scorer) {
        io.WriteString(w, fmt.Sprintf("-------- Draft: %s --------", draft.name))
        for name, player := range draft.Players {
            io.WriteString(w, fmt.Sprintf("%s scored %d points", name, player.totalScore))
        }
    }
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
    From Drafts
    Left Join Picks p On p.draftId = d.Id
    Left Join Players pl On pl.Id = d.player
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
            player := Player{totalScore: s.ScoreTeam(pickedTeam), picks: picks}
            draft.Players[playerName] = &player
        } else {
            draft.Players[playerName].picks = append(draft.Players[playerName].picks, pickedTeam)
            draft.Players[playerName].totalScore += s.ScoreTeam(pickedTeam)
        }
    }

    return drafts
}
