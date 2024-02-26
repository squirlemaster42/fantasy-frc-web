package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

    server "server/web"
)

func main() {
    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    fmt.Println(tbaTok)
    server.CreateServer()
}
