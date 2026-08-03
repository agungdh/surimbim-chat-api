package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/uptrace/bun"
)

type Hub struct {
	db          *bun.DB
	mu          sync.RWMutex
	subscribers map[string]map[*Client]struct{}
	online      map[int64]map[*Client]struct{}
}

func NewHub(db *bun.DB) *Hub {
	return &Hub{
		db:          db,
		subscribers: make(map[string]map[*Client]struct{}),
		online:      make(map[int64]map[*Client]struct{}),
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
		select {
		case c.send <- frame:
		default:
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
		h.Subscribe(dest, c)

	case "UNSUBSCRIBE":
		dest := frame.Headers["destination"]
		if dest == "" {
			return
		}
		h.Unsubscribe(dest, c)

	case "SEND":
		dest := frame.Headers["destination"]
		h.routeSend(c, dest, frame)

	case "DISCONNECT":
		h.RemoveClient(c)
	}
}

func (h *Hub) DB() *bun.DB {
	return h.db
}
