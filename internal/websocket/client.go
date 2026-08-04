package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
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
	id        string
	token     string
	userID    int64
	hub       *Hub
	conn      *websocket.Conn
	send      chan *Frame
	closeOnce sync.Once
}

func (c *Client) ID() string {
	return c.id
}

func (c *Client) UserID() int64 {
	return c.userID
}

func (c *Client) Send(f *Frame) {
	select {
	case c.send <- f:
	default:
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.send)
		c.hub.RemoveClient(c)
	})
}

func (c *Client) readPump() {
	defer func() {
		c.Close()
		c.conn.Close()
	}()

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

		if frame.Command == "CONNECT" || frame.Command == "STOMP" {
			if !c.authenticate(frame) {
				c.forceClose("invalid token")
				return
			}
			c.Send(ConnectFrame(c.userID))
			if c.userID != 0 {
				c.hub.MarkOnline(c)
			}
			continue
		}

		if !c.isAuthorized() {
			c.forceClose("invalid token")
			return
		}

		c.hub.HandleFrame(c, frame)
	}
}

func (c *Client) authenticate(frame *Frame) bool {
	token := frame.Headers["token"]
	if token == "" {
		return false
	}

	userID, ok := c.hub.TokenStore().Validate(token)
	if !ok {
		return false
	}

	c.token = token
	c.userID = userID
	return true
}

func (c *Client) isAuthorized() bool {
	if c.token == "" {
		return false
	}

	userID, ok := c.hub.TokenStore().Validate(c.token)
	if !ok {
		return false
	}

	c.userID = userID
	return true
}

func (c *Client) forceClose(reason string) {
	c.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.ClosePolicyViolation, reason),
		time.Now().Add(writeWait),
	)
	c.Close()
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

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

	if !h.userInConversation(c.userID, payload.ConversationID) {
		c.Send(ErrorFrame("forbidden"))
		return
	}

	msg := &model.Message{
		ConversationID: payload.ConversationID,
		SenderID:       c.userID,
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

	return c
}
