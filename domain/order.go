package domain

import (
	"errors"
	"time"
)

var ErrInvalidStateTransition = errors.New("invalid order state transition")

type OrderID string
type CustomerID string

type OrderStatus string

const (
	StatusPending   OrderStatus = "pending"
	StatusCompleted OrderStatus = "completed"
	StatusCancelled OrderStatus = "cancelled"
)

// ChangeTracker marks the aggregate as modified so the repo knows to persist it.
type ChangeTracker struct {
	dirty bool
}

func (c *ChangeTracker) MarkDirty() {
	c.dirty = true
}

func (c *ChangeTracker) Dirty() bool {
	return c.dirty
}

type Order struct {
	id          OrderID
	customerID  CustomerID
	totalAmount int64
	status      OrderStatus
	completedAt time.Time

	Changes ChangeTracker
	Events  EventRaiser
}

func NewOrder(id OrderID, customerID CustomerID, totalAmount int64) *Order {
	return &Order{
		id:          id,
		customerID:  customerID,
		totalAmount: totalAmount,
		status:      StatusPending,
		Changes:     ChangeTracker{},
		Events:      EventRaiser{},
	}
}

// ReconstituteOrder hydrates an order from persistence. Used by the repository.
func ReconstituteOrder(id OrderID, customerID CustomerID, totalAmount int64, status OrderStatus, completedAt time.Time) *Order {
	return &Order{
		id:          id,
		customerID:  customerID,
		totalAmount: totalAmount,
		status:      status,
		completedAt: completedAt,
		Changes:     ChangeTracker{},
		Events:      EventRaiser{},
	}
}

func (o *Order) ID() OrderID           { return o.id }
func (o *Order) CustomerID() CustomerID { return o.customerID }
func (o *Order) TotalAmount() int64     { return o.totalAmount }
func (o *Order) Status() OrderStatus    { return o.status }
func (o *Order) CompletedAt() time.Time { return o.completedAt }

// Complete transitions the order to completed and raises OrderCompletedEvent.
func (o *Order) Complete(now time.Time) error {
	if o.status != StatusPending {
		return ErrInvalidStateTransition
	}
	o.status = StatusCompleted
	o.completedAt = now
	o.Changes.MarkDirty()
	o.Events.Raise(OrderCompletedEvent{
		OrderID:     string(o.id),
		CustomerID:  string(o.customerID),
		TotalAmount: o.totalAmount,
		CompletedAt: now,
	})
	return nil
}
