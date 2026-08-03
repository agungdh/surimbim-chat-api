package router

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"surimbim-chat-api/internal/auth"
	"surimbim-chat-api/internal/config"
	"surimbim-chat-api/internal/handler"
	mw "surimbim-chat-api/internal/middleware"
	"surimbim-chat-api/internal/websocket"
)

func New(cfg *config.Config, hub *websocket.Hub) http.Handler {
	db := hub.DB()
	ts := auth.NewTokenStore(24 * time.Hour)

	r := chi.NewRouter()

	if cfg.ENV != "prod" {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)

	r.Get("/health", handler.Health())

	r.Post("/api/login", handler.Login(db, ts))

	r.Get("/ws", handler.WsHandler(hub))

	r.With(mw.RequireAuth(ts)).Get("/api/users", handler.ListUsers(db))
	r.Get("/api/users/active", handler.ListActiveUsers(db, hub))

	return r
}
