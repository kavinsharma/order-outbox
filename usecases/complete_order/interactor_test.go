package complete_order

import (
	"context"
	"testing"
	"time"

	"order-outbox/contracts"
	"order-outbox/domain"
)

func TestExecute_ReturnsPlanWithOrderAndOutboxMutations(t *testing.T) {
	order := domain.ReconstituteOrder(
		domain.OrderID("ord-1"),
		domain.CustomerID("cust-1"),
		9999,
		domain.StatusPending,
		time.Time{},
	)
	stub := &stubRepo{order: order}
	uc := NewInteractor(stub, nil)

	plan, err := uc.Execute(context.Background(), &Request{OrderID: "ord-1"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	// One update for order, one insert for OrderCompletedEvent
	if len(plan.Mutations) != 2 {
		t.Errorf("expected 2 mutations, got %d", len(plan.Mutations))
	}
	if order.Status() != domain.StatusCompleted {
		t.Errorf("order status should be completed, got %s", order.Status())
	}
}

func TestExecute_InvalidState_ReturnsError(t *testing.T) {
	order := domain.ReconstituteOrder(
		domain.OrderID("ord-2"),
		domain.CustomerID("cust-2"),
		1000,
		domain.StatusCompleted, // already completed
		time.Now(),
	)
	stub := &stubRepo{order: order}
	uc := NewInteractor(stub, nil)

	_, err := uc.Execute(context.Background(), &Request{OrderID: "ord-2"})
	if err == nil {
		t.Fatal("expected error when completing already-completed order")
	}
	if err != domain.ErrInvalidStateTransition {
		t.Errorf("expected ErrInvalidStateTransition, got %v", err)
	}
}

// --- stubs and helpers ---

type stubRepo struct {
	order *domain.Order
}

func (s *stubRepo) Retrieve(ctx context.Context, id domain.OrderID) (*domain.Order, error) {
	return s.order, nil
}

func (s *stubRepo) UpdateMut(order *domain.Order) contracts.Mutation {
	return &stubMutation{}
}

func (s *stubRepo) OutboxMuts(order *domain.Order) []contracts.Mutation {
	evs := order.Events.Events()
	if len(evs) == 0 {
		return nil
	}
	return []contracts.Mutation{&stubMutation{}}
}

type stubMutation struct{}

func (s *stubMutation) Apply(ctx context.Context, tx interface{}) error { return nil }

