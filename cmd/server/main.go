package main

import (
	"context"
	"log"
	"time"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
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

// newDBPool creates a pgx connection pool, registers the shopspring/decimal
// type on each new connection, and verifies the database is reachable.
func newDBPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Register decimal.Decimal / decimal.NullDecimal codecs with every new
	// connection so the repo can read/write NUMERIC columns directly.
	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
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
