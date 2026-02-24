package contracts

import (
	"context"

	"order-outbox/domain"
)

// Mutation represents a single database write. Applied in order within a transaction.
type Mutation interface {
	Apply(ctx context.Context, tx interface{}) error
}

// OrderRepository loads orders and produces mutations for order + outbox in one go.
type OrderRepository interface {
	Retrieve(ctx context.Context, id domain.OrderID) (*domain.Order, error)
	UpdateMut(order *domain.Order) Mutation
	OutboxMuts(order *domain.Order) []Mutation
}
