package handler

import (
	"encoding/json"
	"net/http"

	"github.com/uptrace/bun"
	"surimbim-chat-api/internal/model"
	"surimbim-chat-api/internal/websocket"
)

func ListUsers(db *bun.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var users []model.User
		err := db.NewSelect().Model(&users).Column("id", "username", "name", "created_at").Scan(r.Context())
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

func ListActiveUsers(db *bun.DB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids := hub.OnlineUserIDs()
		if len(ids) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]model.User{})
			return
		}

		var users []model.User
		err := db.NewSelect().Model(&users).
			Where("id IN (?)", bun.In(ids)).
			Column("id", "username", "name", "created_at").
			Scan(r.Context())
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}
