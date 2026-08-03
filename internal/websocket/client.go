package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"surimbim-chat-api/internal/model"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

type Client struct {
	id   string
	hub  *Hub
	conn *websocket.Conn
	send chan *Frame
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) Send(f *Frame) {
	select {
	case c.send <- f:
	default:
	}
}

func (c *Client) Close() {
	c.hub.RemoveClient(c)
	c.conn.Close()
	close(c.send)
}

func (c *Client) readPump() {
	defer c.Close()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		frame, err := ParseFrame(message)
		if err != nil {
			log.Printf("stomp parse error: %v", err)
			c.Send(ErrorFrame(err.Error()))
			continue
		}

		c.hub.HandleFrame(c, frame)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case frame, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, frame.Serialize()); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type chatPayload struct {
	ConversationID int64  `json:"conversation_id"`
	SenderID       int64  `json:"sender_id"`
	Content        string `json:"content"`
}

func (h *Hub) routeSend(c *Client, dest string, frame *Frame) {
	if dest != "/app/chat" {
		return
	}

	var payload chatPayload
	if err := json.Unmarshal(frame.Body, &payload); err != nil {
		c.Send(ErrorFrame("invalid json body"))
		return
	}

	if payload.Content == "" {
		c.Send(ErrorFrame("content is required"))
		return
	}

	msg := &model.Message{
		ConversationID: payload.ConversationID,
		SenderID:       payload.SenderID,
		Content:        payload.Content,
	}

	_, err := h.db.NewInsert().Model(msg).Exec(context.Background())
	if err != nil {
		log.Printf("failed to save message: %v", err)
		c.Send(ErrorFrame("failed to save message"))
		return
	}

	resp, err := json.Marshal(msg)
	if err != nil {
		return
	}

	topic := fmt.Sprintf("/topic/conversation.%d", msg.ConversationID)
	h.Broadcast(topic, MessageFrame(topic, resp))
}

func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	c := &Client{
		id:   uuid.New().String(),
		hub:  hub,
		conn: conn,
		send: make(chan *Frame, 64),
	}

	go c.writePump()
	go c.readPump()

	c.Send(ConnectFrame())

	return c
}
