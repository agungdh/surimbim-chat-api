package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port           string
	DBPath         string
	DBMaxOpenConns int
	DBMaxIdleConns int
}

func Load() *Config {
	cfg := &Config{
		Port:           "8080",
		DBPath:         "surimbim.db",
		DBMaxOpenConns: 5,
		DBMaxIdleConns: 5,
	}

	if v := os.Getenv("PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("DB_PATH"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("DB_MAX_OPEN_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.DBMaxOpenConns = n
		}
	}
	if v := os.Getenv("DB_MAX_IDLE_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.DBMaxIdleConns = n
		}
	}

	return cfg
}
