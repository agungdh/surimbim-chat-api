package websocket

import (
	"sync"

	"github.com/uptrace/bun"
)

type Hub struct {
	db          *bun.DB
	mu          sync.RWMutex
	subscribers map[string]map[*Client]struct{}
}

func NewHub(db *bun.DB) *Hub {
	return &Hub{
		db:          db,
		subscribers: make(map[string]map[*Client]struct{}),
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
