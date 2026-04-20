// Package domain holds business types shared across the service layers.
package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Status represents the lifecycle state of a quote update.
type Status string

const (
	// StatusPending means the update has been accepted but not yet processed
	// by a worker.
	StatusPending Status = "pending"
	// StatusDone means the worker fetched the price and stored it.
	StatusDone Status = "done"
	// StatusFailed means the worker tried to fetch the price but failed.
	// The reason is recorded in QuoteUpdate.Error.
	StatusFailed Status = "failed"
)

// QuoteUpdate represents a single update request for a currency pair quote.
// A request goes through pending -> (done | failed).
type QuoteUpdate struct {
	ID        uuid.UUID
	Pair      string
	Status    Status
	Price     decimal.NullDecimal // NULL while pending or on failure
	Error     *string             // set only when Status == StatusFailed
	CreatedAt time.Time
	UpdatedAt *time.Time // NULL while pending
}
