package repo

import (
	"context"
	"database/sql"
	"encoding/json"
)

// OutboxEntry is the row written to the outbox table in the same tx as the aggregate.
type OutboxEntry struct {
	ID        string
	EventType string
	Payload   []byte
	CreatedAt string // ISO8601 or Unix; use event time, not time.Now()
}

type outboxInsertMut struct {
	entry OutboxEntry
}

func (m *outboxInsertMut) Apply(ctx context.Context, tx interface{}) error {
	txx := tx.(*sql.Tx)
	_, err := txx.ExecContext(ctx,
		`INSERT INTO outbox (id, event_type, payload, created_at) VALUES ($1, $2, $3, $4)`,
		m.entry.ID, m.entry.EventType, m.entry.Payload, m.entry.CreatedAt,
	)
	return err
}

func toJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
