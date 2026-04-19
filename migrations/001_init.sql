CREATE TABLE IF NOT EXISTS quote_updates (
    id         UUID PRIMARY KEY,
    pair       TEXT        NOT NULL,
    status     TEXT        NOT NULL CHECK (status IN ('pending', 'done', 'failed')),
    price      NUMERIC(20, 8),
    error      TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_quote_updates_pair_done_updated_at
    ON quote_updates (pair, updated_at DESC)
    WHERE status = 'done';

CREATE INDEX IF NOT EXISTS idx_quote_updates_pending_created_at
    ON quote_updates (created_at)
    WHERE status = 'pending';
