package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"surimbim-chat-api/internal/config"
	"surimbim-chat-api/internal/handler"
	mw "surimbim-chat-api/internal/middleware"
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

	r.With(mw.RequireAuth).Get("/api/users", handler.ListUsers(hub.DB()))
	r.Get("/api/users/active", handler.ListActiveUsers(hub.DB(), hub))

	return r
}
