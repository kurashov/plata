package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort string `env:"HTTP_PORT" envDefault:"8080"`

	DBURL string `env:"DB_URL,required"`

	ExchangeAPIKey  string `env:"EXCHANGE_API_KEY,required"`
	ExchangeBaseURL string `env:"EXCHANGE_BASE_URL" envDefault:"https://api.exchangeratesapi.io/v1"`

	WorkerCount   int           `env:"WORKER_COUNT" envDefault:"4"`
	UpdateTimeout time.Duration `env:"UPDATE_TIMEOUT" envDefault:"10s"`
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}
