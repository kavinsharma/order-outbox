package complete_order

import (
	"context"
	"database/sql"
	"time"

	"order-outbox/contracts"
	"order-outbox/domain"
)

type Request struct {
	OrderID domain.OrderID
}

// Plan holds all mutations to apply in a single transaction.
type Plan struct {
	Mutations []contracts.Mutation
}

type Interactor struct {
	repo contracts.OrderRepository
	db   *sql.DB
}

func NewInteractor(repo contracts.OrderRepository, db *sql.DB) *Interactor {
	return &Interactor{repo: repo, db: db}
}

// Execute loads the order, completes it, and persists aggregate + outbox in one transaction.
func (uc *Interactor) Execute(ctx context.Context, req *Request) (*Plan, error) {
	order, err := uc.repo.Retrieve(ctx, req.OrderID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	if err := order.Complete(now); err != nil {
		return nil, err
	}

	plan := &Plan{}
	if order.Changes.Dirty() {
		plan.Mutations = append(plan.Mutations, uc.repo.UpdateMut(order))
	}
	plan.Mutations = append(plan.Mutations, uc.repo.OutboxMuts(order)...)

	return plan, nil
}

// Apply runs the plan inside a single database transaction.
func (uc *Interactor) Apply(ctx context.Context, plan *Plan) error {
	if plan == nil || len(plan.Mutations) == 0 {
		return nil
	}
	tx, err := uc.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, m := range plan.Mutations {
		if err := m.Apply(ctx, tx); err != nil {
			return err
		}
	}
	return tx.Commit()
}
