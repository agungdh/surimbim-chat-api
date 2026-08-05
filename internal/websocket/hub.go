package websocket

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/uptrace/bun"
	"surimbim-chat-api/internal/auth"
	"surimbim-chat-api/internal/model"
)

const (
	defaultHistoryLimit = 50
	maxHistoryLimit     = 100
)

type Hub struct {
	db          *bun.DB
	tokenStore  *auth.TokenStore
	mu          sync.RWMutex
	subscribers map[string]map[*Client]struct{}
	online      map[int64]map[*Client]struct{}
	convMu      sync.RWMutex
	convCache   map[int64][2]int64
}

func NewHub(db *bun.DB, ts *auth.TokenStore) *Hub {
	return &Hub{
		db:          db,
		tokenStore:  ts,
		subscribers: make(map[string]map[*Client]struct{}),
		online:      make(map[int64]map[*Client]struct{}),
		convCache:   make(map[int64][2]int64),
	}
}

func (h *Hub) Subscribe(topic string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.subscribers[topic] == nil {
		h.subscribers[topic] = make(map[*Client]struct{})
	}
	h.subscribers[topic][c] = struct{}{}
}

func (h *Hub) Unsubscribe(topic string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if clients, ok := h.subscribers[topic]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.subscribers, topic)
		}
	}
}

func (h *Hub) RemoveClient(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for topic, clients := range h.subscribers {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.subscribers, topic)
		}
	}

	if c.userID != 0 {
		if clients, ok := h.online[c.userID]; ok {
			delete(clients, c)
			if len(clients) == 0 {
				delete(h.online, c.userID)
				go h.broadcastPresence(c.userID, "offline")
			}
		}
	}
}

func (h *Hub) MarkOnline(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.online[c.userID] == nil {
		h.online[c.userID] = make(map[*Client]struct{})
	}
	wasOffline := len(h.online[c.userID]) == 0
	h.online[c.userID][c] = struct{}{}

	if wasOffline {
		go h.broadcastPresence(c.userID, "online")
	}
}

func (h *Hub) IsOnline(userID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.online[userID]) > 0
}

func (h *Hub) OnlineUserIDs() []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ids := make([]int64, 0, len(h.online))
	for id := range h.online {
		ids = append(ids, id)
	}
	return ids
}

type presenceEvent struct {
	UserID int64  `json:"user_id"`
	Status string `json:"status"`
}

func (h *Hub) broadcastPresence(userID int64, status string) {
	evt := presenceEvent{UserID: userID, Status: status}
	body, err := json.Marshal(evt)
	if err != nil {
		log.Printf("presence marshal error: %v", err)
		return
	}
	h.Broadcast("/topic/presence", MessageFrame("/topic/presence", body))
}

func (h *Hub) Broadcast(topic string, frame *Frame) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := h.subscribers[topic]
	for c := range clients {
		if !trySend(c, frame) {
			go func(client *Client) {
				client.Close()
			}(c)
		}
	}
}

func (h *Hub) HandleFrame(c *Client, frame *Frame) {
	switch frame.Command {
	case "SUBSCRIBE":
		dest := frame.Headers["destination"]
		if dest == "" {
			return
		}

		if convID, ok := conversationIDFromDest(dest); ok {
			if !h.userInConversation(c.userID, convID) {
				c.Send(ErrorFrameFor(frame, "forbidden"))
				return
			}
		}

		h.Subscribe(dest, c)
		c.Send(ReceiptFrame(frame))

	case "UNSUBSCRIBE":
		dest := frame.Headers["destination"]
		if dest == "" {
			return
		}
		h.Unsubscribe(dest, c)

	case "SEND":
		dest := frame.Headers["destination"]
		switch dest {
		case "/app/chat":
			h.routeSend(c, dest, frame)
		case "/app/history":
			h.routeHistory(c, dest, frame)
		}

	case "DISCONNECT":
		h.RemoveClient(c)
	}
}

func (h *Hub) DB() *bun.DB {
	return h.db
}

func (h *Hub) TokenStore() *auth.TokenStore {
	return h.tokenStore
}

func conversationIDFromDest(dest string) (int64, bool) {
	const prefix = "/topic/conversation."

	idStr, ok := strings.CutPrefix(dest, prefix)
	if !ok || idStr == "" {
		return 0, false
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, false
	}

	return id, true
}

func (h *Hub) userInConversation(userID, convID int64) bool {
	h.convMu.RLock()
	participants, ok := h.convCache[convID]
	h.convMu.RUnlock()
	if ok {
		return userID == participants[0] || userID == participants[1]
	}

	var conv model.Conversation
	err := h.db.NewSelect().Model(&conv).Where("id = ?", convID).Scan(context.Background())
	if err != nil {
		return false
	}

	h.convMu.Lock()
	h.convCache[convID] = [2]int64{conv.User1ID, conv.User2ID}
	h.convMu.Unlock()

	return conv.User1ID == userID || conv.User2ID == userID
}

func (h *Hub) routeHistory(c *Client, dest string, frame *Frame) {
	if dest != "/app/history" {
		return
	}

	convID, err := strconv.ParseInt(frame.Headers["conversation-id"], 10, 64)
	if err != nil || convID == 0 {
		c.Send(ErrorFrameFor(frame, "invalid conversation-id"))
		return
	}

	if !h.userInConversation(c.userID, convID) {
		c.Send(ErrorFrameFor(frame, "forbidden"))
		return
	}

	limit := defaultHistoryLimit
	if s := frame.Headers["limit"]; s != "" {
		if l, err := strconv.Atoi(s); err == nil && l > 0 && l <= maxHistoryLimit {
			limit = l
		}
	}

	query := h.db.NewSelect().Model((*model.Message)(nil)).
		Where("conversation_id = ?", convID).
		Order("id DESC").
		Limit(limit + 1)

	if cursor := frame.Headers["cursor"]; cursor != "" {
		id, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			c.Send(ErrorFrameFor(frame, "invalid cursor"))
			return
		}
		query = query.Where("id < ?", id)
	}

	var messages []model.Message
	if err := query.Scan(context.Background()); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("failed to load history: %v", err)
			c.Send(ErrorFrameFor(frame, "failed to load history"))
			return
		}
		messages = []model.Message{}
	}

	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit]
	}

	nextCursor := ""
	if len(messages) > 0 {
		nextCursor = strconv.FormatInt(messages[len(messages)-1].ID, 10)
	}
	// query returns newest first; flip to chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	resp, err := json.Marshal(messages)
	if err != nil {
		c.Send(ErrorFrameFor(frame, "failed to encode history"))
		return
	}

	respFrame := MessageFrame("/app/history", resp)
	respFrame.Headers["conversation-id"] = strconv.FormatInt(convID, 10)
	respFrame.Headers["has-more"] = strconv.FormatBool(hasMore)
	if nextCursor != "" {
		respFrame.Headers["next-cursor"] = nextCursor
	}
	if clientID := frame.Headers["client-id"]; clientID != "" {
		respFrame.Headers["client-id"] = clientID
	}
	c.Send(respFrame)
}
