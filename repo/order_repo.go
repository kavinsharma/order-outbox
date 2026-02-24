package repo

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"order-outbox/contracts"
	"order-outbox/domain"
)

type OrderRepository struct {
	db *sql.DB
}

func NewOrderRepository(db *sql.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Retrieve(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
	var customerID string
	var totalAmount int64
	var status string
	var completedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT customer_id, total_amount_cents, status, completed_at FROM orders WHERE id = $1`,
		string(id),
	).Scan(&customerID, &totalAmount, &status, &completedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("order not found: %s", id)
		}
		return nil, err
	}
	var completed time.Time
	if completedAt.Valid {
		completed = completedAt.Time
	}
	return domain.ReconstituteOrder(
		domain.OrderID(id),
		domain.CustomerID(customerID),
		totalAmount,
		domain.OrderStatus(status),
		completed,
	), nil
}

func (r *OrderRepository) UpdateMut(order *domain.Order) contracts.Mutation {
	return &orderUpdateMut{order: order}
}

func (r *OrderRepository) OutboxMuts(order *domain.Order) []contracts.Mutation {
	events := order.Events.Events()
	if len(events) == 0 {
		return nil
	}
	var muts []contracts.Mutation
	for _, ev := range events {
		entry := OutboxEntry{
			ID:        newID(),
			EventType: ev.EventType(),
			Payload:   toJSON(ev),
			CreatedAt: eventCreatedAt(ev), // use event time, not time.Now()
		}
		muts = append(muts, &outboxInsertMut{entry: entry})
	}
	return muts
}

// eventCreatedAt returns a stable timestamp for the event (e.g. CompletedAt).
func eventCreatedAt(ev domain.DomainEvent) string {
	switch e := ev.(type) {
	case domain.OrderCompletedEvent:
		return e.CompletedAt.UTC().Format(time.RFC3339)
	default:
		return time.Now().UTC().Format(time.RFC3339)
	}
}

type orderUpdateMut struct {
	order *domain.Order
}

func (m *orderUpdateMut) Apply(ctx context.Context, tx interface{}) error {
	txx := tx.(*sql.Tx)
	_, err := txx.ExecContext(ctx,
		`UPDATE orders SET status = $1, completed_at = $2 WHERE id = $3`,
		string(m.order.Status()), m.order.CompletedAt(), string(m.order.ID()),
	)
	return err
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Ensure OrderRepository implements contracts.OrderRepository
var _ contracts.OrderRepository = (*OrderRepository)(nil)
