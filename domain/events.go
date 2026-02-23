package domain

import "time"

// OrderCompletedEvent is raised when an order transitions to completed.
// Used by downstream consumers (notifications, analytics, inventory).
type OrderCompletedEvent struct {
	OrderID     string
	CustomerID  string
	TotalAmount int64 // cents
	CompletedAt time.Time
}

func (e OrderCompletedEvent) EventType() string {
	return "order.completed"
}
