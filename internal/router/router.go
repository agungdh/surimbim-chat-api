package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/uptrace/bun"
	"surimbim-chat-api/internal/handler"
)

func New(db *bun.DB) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", handler.Health())

	chat := handler.NewChat(db)
	r.Post("/api/chat", chat.Send())

	return r
}
