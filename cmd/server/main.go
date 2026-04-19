package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kurashov/plata/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf("config loaded: http_port=%s workers=%d update_timeout=%s",
		cfg.HTTPPort, cfg.WorkerCount, cfg.UpdateTimeout)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := newDBPool(ctx, cfg.DBURL)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	log.Printf("db connected")
}

func newDBPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}
