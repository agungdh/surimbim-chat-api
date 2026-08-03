package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"surimbim-chat-api/internal/config"
	"surimbim-chat-api/internal/handler"
	"surimbim-chat-api/internal/websocket"
)

func New(cfg *config.Config, hub *websocket.Hub) http.Handler {
	r := chi.NewRouter()

	if cfg.ENV != "prod" {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)

	r.Get("/health", handler.Health())

	r.Get("/ws", handler.WsHandler(hub))

	return r
}
