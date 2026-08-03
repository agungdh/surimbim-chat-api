package database

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
	"surimbim-chat-api/internal/config"
)

func Connect(cfg *config.Config) (*bun.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)", cfg.DBPath)

	sqldb, err := sql.Open(sqliteshim.ShimName, dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	sqldb.SetMaxOpenConns(cfg.DBMaxOpenConns)
	sqldb.SetMaxIdleConns(cfg.DBMaxIdleConns)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, fmt.Errorf("goose set dialect: %w", err)
	}
	if err := goose.Up(sqldb, "migrations"); err != nil {
		return nil, fmt.Errorf("goose up: %w", err)
	}

	db := bun.NewDB(sqldb, sqlitedialect.New())
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}

	return db, nil
}
