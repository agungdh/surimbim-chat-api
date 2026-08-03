package main

import (
	"log"
	"net/http"

	"github.com/joho/godotenv"

	"surimbim-chat-api/internal/config"
	"surimbim-chat-api/internal/database"
	"surimbim-chat-api/internal/router"
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

	handler := router.New(cfg, db)

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatal(err)
	}
}
