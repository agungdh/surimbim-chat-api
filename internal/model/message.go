package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Message struct {
	bun.BaseModel `bun:"table:messages"`

	ID        int64     `bun:"id,pk,autoincrement" json:"id"`
	UserID    int64     `bun:"user_id,notnull,default:0" json:"user_id"`
	Content   string    `bun:"content,notnull" json:"content"`
	CreatedAt time.Time `bun:"created_at,notnull,default:datetime('now')" json:"created_at"`
}
