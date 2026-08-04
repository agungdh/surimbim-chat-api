package handler

import (
	"database/sql"
	"errors"
	"net/http"
	"sort"
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
			respondError(w, http.StatusBadRequest, "invalid user_id")
			return
		}

		myID := mw.UserIDFromCtx(r.Context())

		var other model.User
		err = db.NewSelect().Model(&other).
			Column("id", "username", "name", "created_at").
			Where("id = ?", otherID).
			Scan(r.Context())
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				respondError(w, http.StatusNotFound, "user not found")
				return
			}
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		var conv model.Conversation
		err = db.NewSelect().Model(&conv).
			Where("(user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)", myID, otherID, otherID, myID).
			Scan(r.Context())
		if err == nil {
			respondJSON(w, http.StatusOK, conversationResponse{Conversation: conv, User: other})
			return
		}
		if !errors.Is(err, sql.ErrNoRows) {
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		user1, user2 := myID, otherID
		if user1 > user2 {
			user1, user2 = user2, user1
		}

		conv = model.Conversation{User1ID: user1, User2ID: user2}
		_, err = db.NewInsert().Model(&conv).
			On("CONFLICT (user1_id, user2_id) DO NOTHING").
			Exec(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		status := http.StatusOK
		if conv.ID == 0 {
			// another request created it concurrently; re-fetch the row
			err = db.NewSelect().Model(&conv).
				Where("user1_id = ? AND user2_id = ?", user1, user2).
				Scan(r.Context())
			if err != nil {
				respondError(w, http.StatusInternalServerError, "internal server error")
				return
			}
		} else {
			status = http.StatusCreated
		}

		respondJSON(w, status, conversationResponse{Conversation: conv, User: other})
	}
}

type conversationListItem struct {
	model.Conversation
	User        model.User     `json:"user"`
	LastMessage *model.Message `json:"last_message,omitempty"`
}

func ListConversations(db *bun.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		myID := mw.UserIDFromCtx(r.Context())

		var conversations []model.Conversation
		err := db.NewSelect().Model(&conversations).
			Where("user1_id = ? OR user2_id = ?", myID, myID).
			Order("id DESC").
			Scan(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		items := make([]conversationListItem, 0, len(conversations))
		if len(conversations) == 0 {
			respondJSON(w, http.StatusOK, items)
			return
		}

		otherIDs := make([]int64, 0, len(conversations))
		convIDs := make([]int64, 0, len(conversations))
		for _, c := range conversations {
			convIDs = append(convIDs, c.ID)
			if c.User1ID == myID {
				otherIDs = append(otherIDs, c.User2ID)
			} else {
				otherIDs = append(otherIDs, c.User1ID)
			}
		}

		var others []model.User
		err = db.NewSelect().Model(&others).
			Column("id", "username", "name", "created_at").
			Where("id IN (?)", bun.In(otherIDs)).
			Scan(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		userByID := make(map[int64]model.User, len(others))
		for _, u := range others {
			userByID[u.ID] = u
		}

		var lastMessages []model.Message
		err = db.NewSelect().Model(&lastMessages).
			Where("id IN (SELECT MAX(id) FROM messages WHERE conversation_id IN (?) GROUP BY conversation_id)", bun.In(convIDs)).
			Scan(r.Context())
		if err != nil {
			respondError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		lastByConv := make(map[int64]model.Message, len(lastMessages))
		for _, m := range lastMessages {
			lastByConv[m.ConversationID] = m
		}

		for _, c := range conversations {
			item := conversationListItem{Conversation: c}
			if c.User1ID == myID {
				item.User = userByID[c.User2ID]
			} else {
				item.User = userByID[c.User1ID]
			}
			if m, ok := lastByConv[c.ID]; ok {
				item.LastMessage = &m
			}
			items = append(items, item)
		}

		sort.SliceStable(items, func(i, j int) bool {
			ti := int64(0)
			if items[i].LastMessage != nil {
				ti = items[i].LastMessage.ID
			}
			tj := int64(0)
			if items[j].LastMessage != nil {
				tj = items[j].LastMessage.ID
			}
			return ti > tj
		})

		respondJSON(w, http.StatusOK, items)
	}
}
