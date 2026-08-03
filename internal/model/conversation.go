package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Conversation struct {
	bun.BaseModel `bun:"table:conversations"`

	ID        int64     `bun:"id,pk,autoincrement" json:"id"`
	User1ID   int64     `bun:"user1_id,notnull" json:"user1_id"`
	User2ID   int64     `bun:"user2_id,notnull" json:"user2_id"`
	CreatedAt time.Time `bun:"created_at,notnull,default:datetime('now')" json:"created_at"`
}
