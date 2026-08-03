package main

import (
	"log"
	"net/http"

	"surimbim-chat-api/internal/config"
	"surimbim-chat-api/internal/database"
	"surimbim-chat-api/internal/router"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}

	handler := router.New(db)

	log.Printf("listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatal(err)
	}
}
