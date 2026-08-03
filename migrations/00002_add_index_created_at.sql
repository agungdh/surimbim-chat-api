-- +goose Up
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_messages_created_at;
