package handler

import (
	"net/http"

	"github.com/gorilla/websocket"
	"surimbim-chat-api/internal/config"
	mw "surimbim-chat-api/internal/middleware"
	wspkg "surimbim-chat-api/internal/websocket"
)

func WsHandler(hub *wspkg.Hub, cfg *config.Config) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return mw.OriginAllowed(r, cfg.CORSOrigins)
		},
	}

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
