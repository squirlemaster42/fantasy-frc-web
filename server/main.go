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
    dbPassword := os.Getenv("DB_PASSWORD")
    fmt.Println(tbaTok)
    fmt.Println(dbPassword)
    server.CreateServer()
}
