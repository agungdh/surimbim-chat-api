package config

import "github.com/caarlos0/env/v11"

type Config struct {
	Port           string `env:"PORT" envDefault:"8080"`
	DBPath         string `env:"DB_PATH" envDefault:"surimbim.db"`
	DBMaxOpenConns int    `env:"DB_MAX_OPEN_CONNS" envDefault:"5"`
	DBMaxIdleConns int    `env:"DB_MAX_IDLE_CONNS" envDefault:"5"`
	ENV            string `env:"ENV" envDefault:"dev"`
}

func Load() *Config {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		panic(err)
	}
	return cfg
}
