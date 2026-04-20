//go:build integration

package repo

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	pgxdecimal "github.com/jackc/pgx-shopspring-decimal"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

const defaultTestDSN = "postgres://plata:plata@localhost:5432/plata?sslmode=disable"

// setupDB connects to the integration Postgres, applies every migration from
// migrations/ (idempotent via IF NOT EXISTS) and truncates the quote_updates
// table so each test starts from a clean slate.
//
// Override the DSN via TEST_DB_URL when running in CI.
func setupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	cfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		pgxdecimal.Register(conn.TypeMap())
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	require.NoError(t, err, "connect to test db (is docker compose up?)")

	require.NoError(t, pool.Ping(ctx), "ping test db")

	applyMigrations(t, pool)

	_, err = pool.Exec(ctx, "TRUNCATE TABLE quote_updates")
	require.NoError(t, err)

	t.Cleanup(pool.Close)
	return pool
}

func applyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	dir, err := filepath.Abs(filepath.Join("..", "..", "migrations"))
	require.NoError(t, err)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err, "read migrations dir")

	ctx := context.Background()
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		sqlBytes, err := os.ReadFile(path)
		require.NoError(t, err, "read %s", path)

		_, err = pool.Exec(ctx, string(sqlBytes))
		require.NoErrorf(t, err, "apply migration %s", e.Name())
	}
}
