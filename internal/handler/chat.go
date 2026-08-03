package handler

import (
	"encoding/json"
	"net/http"

	"github.com/uptrace/bun"
	"surimbim-chat-api/internal/model"
)

type ChatHandler struct {
	db *bun.DB
}

func NewChat(db *bun.DB) *ChatHandler {
	return &ChatHandler{db: db}
}

type chatRequest struct {
	Content string `json:"content"`
}

func (h *ChatHandler) Send() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
			return
		}
		if req.Content == "" {
			http.Error(w, `{"error":"content is required"}`, http.StatusBadRequest)
			return
		}

		msg := &model.Message{
			Content: req.Content,
		}

		ctx := r.Context()
		if _, err := h.db.NewInsert().Model(msg).Exec(ctx); err != nil {
			http.Error(w, `{"error":"failed to save message"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(msg)
	}
}
