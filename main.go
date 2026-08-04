package main

import (
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"

	"surimbim-chat-api/internal/auth"
	"surimbim-chat-api/internal/config"
	"surimbim-chat-api/internal/database"
	"surimbim-chat-api/internal/router"
	"surimbim-chat-api/internal/websocket"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, using system env")
	}

	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}

	ts := auth.NewTokenStore(24 * time.Hour)
	hub := websocket.NewHub(db, ts)

	handler := router.New(cfg, hub)

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatal(err)
	}
}
