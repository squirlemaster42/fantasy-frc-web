package web

import (
	"fmt"
	"io"
	"net/http"
	"os"
    "errors"
)

func CreateServer() {
    mux := http.NewServeMux()
    mux.HandleFunc("/scores", getScores)

    err := http.ListenAndServe(":3333", mux)
    if errors.Is(err, http.ErrServerClosed) {
        fmt.Printf("server closed\n")
    } else if err != nil {
        fmt.Printf("errors starting server: %s\n", err)
        os.Exit(1)
    }
}

func getScores(w http.ResponseWriter, r *http.Request) {
    fmt.Printf("got / request\n")
    io.WriteString(w, "This is my website!\n")
}
