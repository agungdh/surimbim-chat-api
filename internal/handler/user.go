package handler

import (
	"net/http"

	"github.com/uptrace/bun"
	mw "surimbim-chat-api/internal/middleware"
	"surimbim-chat-api/internal/model"
	"surimbim-chat-api/internal/websocket"
)

func Me(db *bun.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		myID := mw.UserIDFromCtx(r.Context())

		var user model.User
		err := db.NewSelect().Model(&user).
			Column("id", "username", "name", "created_at").
			Where("id = ?", myID).
			Scan(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		respondJSON(w, http.StatusOK, user)
	}
}

func ListUsers(db *bun.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var users []model.User
		err := db.NewSelect().Model(&users).Column("id", "username", "name", "created_at").Scan(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		respondJSON(w, http.StatusOK, users)
	}
}

func ListActiveUsers(db *bun.DB, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ids := hub.OnlineUserIDs()
		if len(ids) == 0 {
			respondJSON(w, http.StatusOK, []model.User{})
			return
		}

		var users []model.User
		err := db.NewSelect().Model(&users).
			Where("id IN (?)", bun.In(ids)).
			Column("id", "username", "name", "created_at").
			Scan(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		respondJSON(w, http.StatusOK, users)
	}
}
