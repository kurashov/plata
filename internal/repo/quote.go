// Package repo provides data-access methods backed by PostgreSQL.
package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"

	"github.com/kurashov/plata/internal/domain"
)

// ErrNotFound is returned when a quote update cannot be located.
var ErrNotFound = errors.New("quote update not found")

// QuoteRepo encapsulates SQL access to the quote_updates table.
type QuoteRepo struct {
	pool *pgxpool.Pool
}

// NewQuoteRepo creates a QuoteRepo bound to the given connection pool.
func NewQuoteRepo(pool *pgxpool.Pool) *QuoteRepo {
	return &QuoteRepo{pool: pool}
}

// CreateUpdate inserts a new pending update for the given currency pair and
// returns the generated identifier.
func (r *QuoteRepo) CreateUpdate(ctx context.Context, pair string) (uuid.UUID, error) {
	id := uuid.New()
	_, err := r.pool.Exec(ctx, `
		INSERT INTO quote_updates (id, pair, status)
		VALUES ($1, $2, $3)
	`, id, pair, domain.StatusPending)
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert quote update: %w", err)
	}
	return id, nil
}

// GetByID returns a quote update by its identifier or ErrNotFound.
func (r *QuoteRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.QuoteUpdate, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, pair, status, price, error, created_at, updated_at
		  FROM quote_updates
		 WHERE id = $1
	`, id)
	return scanQuote(row)
}

// GetLatestByPair returns the most recent successful (done) quote for a pair
// or ErrNotFound if none exists.
func (r *QuoteRepo) GetLatestByPair(ctx context.Context, pair string) (*domain.QuoteUpdate, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, pair, status, price, error, created_at, updated_at
		  FROM quote_updates
		 WHERE pair = $1 AND status = $2
		 ORDER BY updated_at DESC
		 LIMIT 1
	`, pair, domain.StatusDone)
	return scanQuote(row)
}

// MarkDone transitions a pending update to done with the fetched price.
// Returns ErrNotFound if no pending row matches (already processed or missing).
func (r *QuoteRepo) MarkDone(ctx context.Context, id uuid.UUID, price decimal.Decimal, updatedAt time.Time) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE quote_updates
		   SET status = $2, price = $3, updated_at = $4, error = NULL
		 WHERE id = $1 AND status = $5
	`, id, domain.StatusDone, price, updatedAt, domain.StatusPending)
	if err != nil {
		return fmt.Errorf("mark done: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// MarkFailed transitions a pending update to failed with the given error message.
// Returns ErrNotFound if no pending row matches.
func (r *QuoteRepo) MarkFailed(ctx context.Context, id uuid.UUID, errMsg string) error {
	cmd, err := r.pool.Exec(ctx, `
		UPDATE quote_updates
		   SET status = $2, error = $3, updated_at = $4
		 WHERE id = $1 AND status = $5
	`, id, domain.StatusFailed, errMsg, time.Now().UTC(), domain.StatusPending)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FetchPending returns up to `limit` oldest pending updates. Used by the worker
// on startup to recover tasks that were in flight when the previous run exited.
func (r *QuoteRepo) FetchPending(ctx context.Context, limit int) ([]*domain.QuoteUpdate, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, pair, status, price, error, created_at, updated_at
		  FROM quote_updates
		 WHERE status = $1
		 ORDER BY created_at ASC
		 LIMIT $2
	`, domain.StatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch pending: %w", err)
	}
	defer rows.Close()

	var out []*domain.QuoteUpdate
	for rows.Next() {
		q, err := scanQuote(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending: %w", err)
	}
	return out, nil
}

// scanRow abstracts over pgx.Row and pgx.Rows so one helper works for both.
type scanRow interface {
	Scan(dest ...any) error
}

func scanQuote(row scanRow) (*domain.QuoteUpdate, error) {
	var q domain.QuoteUpdate
	err := row.Scan(
		&q.ID, &q.Pair, &q.Status,
		&q.Price, &q.Error, &q.CreatedAt, &q.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan quote update: %w", err)
	}
	return &q, nil
}
