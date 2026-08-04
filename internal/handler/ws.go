package handler

import (
	"net/http"

	"github.com/gorilla/websocket"
	wspkg "surimbim-chat-api/internal/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WsHandler(hub *wspkg.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		token := ""
		if c, err := r.Cookie("token"); err == nil {
			token = c.Value
		}
		wspkg.NewClient(conn, hub, token)
	}
}
