-- Orders table (minimal for the use case)
CREATE TABLE IF NOT EXISTS orders (
    id                VARCHAR(36) PRIMARY KEY,
    customer_id       VARCHAR(36) NOT NULL,
    total_amount_cents BIGINT NOT NULL,
    status            VARCHAR(20) NOT NULL,
    completed_at      TIMESTAMPTZ
);

-- Outbox: one row per domain event, written in same tx as aggregate
CREATE TABLE IF NOT EXISTS outbox (
    id          VARCHAR(36) PRIMARY KEY,
    event_type  VARCHAR(128) NOT NULL,
    payload     JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,
    processed_at TIMESTAMPTZ
);

-- Worker selects unprocessed events, ordered by created_at for FIFO
CREATE INDEX idx_outbox_unprocessed ON outbox (created_at)
    WHERE processed_at IS NULL;

-- Optional: per event_type if workers are sharded by type
CREATE INDEX idx_outbox_event_type_unprocessed ON outbox (event_type, created_at)
    WHERE processed_at IS NULL;
