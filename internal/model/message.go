package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Message struct {
	bun.BaseModel `bun:"table:messages"`

	ID             int64     `bun:"id,pk,autoincrement" json:"id"`
	ConversationID int64     `bun:"conversation_id,notnull" json:"conversation_id"`
	SenderID       int64     `bun:"sender_id,notnull" json:"sender_id"`
	Content        string    `bun:"content,notnull" json:"content"`
	CreatedAt      time.Time `bun:"created_at,notnull,default:datetime('now')" json:"created_at"`
}
