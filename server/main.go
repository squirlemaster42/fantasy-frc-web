package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"

    a "server/web"
)

func main() {
    godotenv.Load()
    tbaTok := os.Getenv("TBA_TOKEN")
    fmt.Println(tbaTok)
    a.CreateServer()
}
