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
	db := hub.DB()
	ts := hub.TokenStore()

	r := chi.NewRouter()

	if cfg.ENV != "prod" {
		r.Use(middleware.Logger)
	}
	r.Use(middleware.Recoverer)
	r.Use(mw.CORS)

	r.Get("/health", handler.Health())

	r.Post("/api/login", handler.Login(db, ts, cfg))
	r.Post("/api/logout", handler.Logout(ts))

	r.Get("/ws", handler.WsHandler(hub))

	r.With(mw.RequireAuth(ts)).Get("/api/users", handler.ListUsers(db))
	r.With(mw.RequireAuth(ts)).Get("/api/me", handler.Me(db))
	r.With(mw.RequireAuth(ts)).Get("/api/conversation/{user_id}", handler.GetConversation(db))
	r.Get("/api/users/active", handler.ListActiveUsers(db, hub))

	return r
}
