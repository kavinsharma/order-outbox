# Order completion with outbox

Order completion use case: complete an order in a single transaction with domain events written to an outbox table. A background worker (not included) would poll the outbox and publish events for at-least-once delivery.

- **domain/** — Order aggregate, `Complete(now)`, `EventRaiser`, `OrderCompletedEvent`
- **contracts/** — `OrderRepository`: `Retrieve`, `UpdateMut`, `OutboxMuts`
- **repo/** — Order + outbox mutations; outbox rows use event timestamp for `created_at`
- **usecases/complete_order/** — `Execute` builds a plan (order + outbox mutations), `Apply` runs it in one tx

See `SCHEMA.sql` for the outbox DDL, `REVIEW.md` for the buggy implementation review, and `ANSWERS.md` for the written questions.

### Build and test

```bash
cd /path/to/golang-part-2
go build ./...
go test ./...
```

### Run in your own app

There is no `main` — this is library code. To run the flow yourself:

1. Create a Postgres DB and apply `SCHEMA.sql` (creates `orders` and `outbox` tables).
2. In your application, open the DB, then:
   - `repo := repo.NewOrderRepository(db)`
   - `uc := complete_order.NewInteractor(repo, db)`
   - Load or insert an order (e.g. `INSERT INTO orders (id, customer_id, total_amount_cents, status) VALUES (...)`)
   - `plan, err := uc.Execute(ctx, &complete_order.Request{OrderID: "your-order-id"})`
   - `err = uc.Apply(ctx, plan)`
3. The order is updated and one row is written to `outbox` in a single transaction. A separate worker would poll `outbox` and publish events.
