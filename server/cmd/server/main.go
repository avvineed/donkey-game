package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/vineed/games/kazhuta/server/internal/api"
	"github.com/vineed/games/kazhuta/server/internal/game"
)

func main() {
	addr := envOrDefault("ADDR", ":8080")

	manager := game.NewManager(time.Now().UnixNano())
	server := api.NewServer(manager)

	log.Printf("kazhuta server listening on %s", addr)
	if err := http.ListenAndServe(addr, server.Routes()); err != nil {
		log.Fatal(err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
