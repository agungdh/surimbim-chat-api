package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/uptrace/bun"
	mw "surimbim-chat-api/internal/middleware"
	"surimbim-chat-api/internal/model"
)

type conversationResponse struct {
	model.Conversation
	User model.User `json:"user"`
}

func GetConversation(db *bun.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		otherID, err := strconv.ParseInt(chi.URLParam(r, "user_id"), 10, 64)
		if err != nil || otherID == 0 {
			http.Error(w, `{"error":"invalid user_id"}`, http.StatusBadRequest)
			return
		}

		myID := mw.UserIDFromCtx(r.Context())
		if myID == otherID {
			http.Error(w, `{"error":"cannot check conversation with yourself"}`, http.StatusBadRequest)
			return
		}

		var other model.User
		err = db.NewSelect().Model(&other).
			Column("id", "username", "name", "created_at").
			Where("id = ?", otherID).
			Scan(r.Context())
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, `{"error":"user not found"}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		var conv model.Conversation
		err = db.NewSelect().Model(&conv).
			Where("(user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)", myID, otherID, otherID, myID).
			Scan(r.Context())
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(conversationResponse{Conversation: conv, User: other})
			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		user1, user2 := myID, otherID
		if user1 > user2 {
			user1, user2 = user2, user1
		}

		conv = model.Conversation{User1ID: user1, User2ID: user2}
		if _, err := db.NewInsert().Model(&conv).Exec(r.Context()); err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(conversationResponse{Conversation: conv, User: other})
	}
}
