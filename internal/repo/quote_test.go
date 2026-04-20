//go:build integration

package repo

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kurashov/plata/internal/domain"
)

func TestCreateAndGetByID_Pending(t *testing.T) {
	ctx := context.Background()
	r := NewQuoteRepo(setupDB(t))

	id, err := r.CreateUpdate(ctx, "EUR/MXN")
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, id)

	q, err := r.GetByID(ctx, id)
	require.NoError(t, err)

	assert.Equal(t, id, q.ID)
	assert.Equal(t, "EUR/MXN", q.Pair)
	assert.Equal(t, domain.StatusPending, q.Status)
	assert.False(t, q.Price.Valid)
	assert.Nil(t, q.Error)
	assert.Nil(t, q.UpdatedAt)
	assert.WithinDuration(t, time.Now(), q.CreatedAt, 5*time.Second)
}

func TestGetByID_NotFound(t *testing.T) {
	ctx := context.Background()
	r := NewQuoteRepo(setupDB(t))

	_, err := r.GetByID(ctx, uuid.New())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMarkDone_TransitionsPendingToDone(t *testing.T) {
	ctx := context.Background()
	r := NewQuoteRepo(setupDB(t))

	id, err := r.CreateUpdate(ctx, "USD/MXN")
	require.NoError(t, err)

	price := decimal.RequireFromString("17.12345678")
	updatedAt := time.Now().UTC().Truncate(time.Microsecond)

	require.NoError(t, r.MarkDone(ctx, id, price, updatedAt))

	q, err := r.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusDone, q.Status)
	assert.True(t, q.Price.Valid)
	assert.True(t, price.Equal(q.Price.Decimal), "price: want %s got %s", price, q.Price.Decimal)
	require.NotNil(t, q.UpdatedAt)
	assert.WithinDuration(t, updatedAt, *q.UpdatedAt, time.Second)
	assert.Nil(t, q.Error)
}

func TestMarkDone_IdempotentOnRepeat(t *testing.T) {
	// Second MarkDone on the same id must be a no-op (ErrNotFound because
	// the row is no longer pending). This is what protects against a double
	// worker pickup.
	ctx := context.Background()
	r := NewQuoteRepo(setupDB(t))

	id, err := r.CreateUpdate(ctx, "USD/MXN")
	require.NoError(t, err)

	price := decimal.NewFromInt(42)
	require.NoError(t, r.MarkDone(ctx, id, price, time.Now().UTC()))

	err = r.MarkDone(ctx, id, price, time.Now().UTC())
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMarkFailed_SetsErrorAndStatus(t *testing.T) {
	ctx := context.Background()
	r := NewQuoteRepo(setupDB(t))

	id, err := r.CreateUpdate(ctx, "EUR/USD")
	require.NoError(t, err)

	require.NoError(t, r.MarkFailed(ctx, id, "provider timeout"))

	q, err := r.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusFailed, q.Status)
	assert.False(t, q.Price.Valid)
	require.NotNil(t, q.Error)
	assert.Equal(t, "provider timeout", *q.Error)
	assert.NotNil(t, q.UpdatedAt)
}

func TestMarkFailed_IdempotentOnRepeat(t *testing.T) {
	ctx := context.Background()
	r := NewQuoteRepo(setupDB(t))

	id, err := r.CreateUpdate(ctx, "EUR/USD")
	require.NoError(t, err)
	require.NoError(t, r.MarkFailed(ctx, id, "boom"))

	err = r.MarkFailed(ctx, id, "boom again")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetLatestByPair_ReturnsMostRecentDone(t *testing.T) {
	ctx := context.Background()
	r := NewQuoteRepo(setupDB(t))

	older, err := r.CreateUpdate(ctx, "EUR/MXN")
	require.NoError(t, err)
	require.NoError(t, r.MarkDone(ctx, older, decimal.NewFromInt(10), time.Now().UTC().Add(-1*time.Hour)))

	newer, err := r.CreateUpdate(ctx, "EUR/MXN")
	require.NoError(t, err)
	newerPrice := decimal.RequireFromString("20.5")
	require.NoError(t, r.MarkDone(ctx, newer, newerPrice, time.Now().UTC()))

	// Same pair still pending — should not be picked.
	_, err = r.CreateUpdate(ctx, "EUR/MXN")
	require.NoError(t, err)

	// Different pair done — should not be picked.
	other, err := r.CreateUpdate(ctx, "USD/MXN")
	require.NoError(t, err)
	require.NoError(t, r.MarkDone(ctx, other, decimal.NewFromInt(17), time.Now().UTC()))

	q, err := r.GetLatestByPair(ctx, "EUR/MXN")
	require.NoError(t, err)
	assert.Equal(t, newer, q.ID)
	assert.True(t, newerPrice.Equal(q.Price.Decimal))
}

func TestGetLatestByPair_NoDoneReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	r := NewQuoteRepo(setupDB(t))

	// Only a pending row exists — no successful quote yet.
	_, err := r.CreateUpdate(ctx, "EUR/MXN")
	require.NoError(t, err)

	_, err = r.GetLatestByPair(ctx, "EUR/MXN")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestFetchPending_OldestFirst(t *testing.T) {
	ctx := context.Background()
	r := NewQuoteRepo(setupDB(t))

	id1, err := r.CreateUpdate(ctx, "EUR/MXN")
	require.NoError(t, err)

	// Ensure monotonic created_at even if the clock resolution is coarse.
	time.Sleep(10 * time.Millisecond)

	id2, err := r.CreateUpdate(ctx, "USD/MXN")
	require.NoError(t, err)

	// Done row must NOT be returned.
	done, err := r.CreateUpdate(ctx, "EUR/USD")
	require.NoError(t, err)
	require.NoError(t, r.MarkDone(ctx, done, decimal.NewFromInt(1), time.Now().UTC()))

	items, err := r.FetchPending(ctx, 10)
	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, id1, items[0].ID, "oldest pending must come first")
	assert.Equal(t, id2, items[1].ID)
}

func TestFetchPending_RespectsLimit(t *testing.T) {
	ctx := context.Background()
	r := NewQuoteRepo(setupDB(t))

	for i := 0; i < 3; i++ {
		_, err := r.CreateUpdate(ctx, "EUR/MXN")
		require.NoError(t, err)
	}

	items, err := r.FetchPending(ctx, 2)
	require.NoError(t, err)
	assert.Len(t, items, 2)
}
