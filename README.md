# Order completion with outbox

Order completion use case: complete an order in a single transaction with domain events written to an outbox table. A background worker (not included) would poll the outbox and publish events for at-least-once delivery.

- **domain/** — Order aggregate, `Complete(now)`, `EventRaiser`, `OrderCompletedEvent`
- **contracts/** — `OrderRepository`: `Retrieve`, `UpdateMut`, `OutboxMuts`
- **repo/** — Order + outbox mutations; outbox rows use event timestamp for `created_at`
- **usecases/complete_order/** — `Execute` builds a plan (order + outbox mutations), `Apply` runs it in one tx

See `SCHEMA.sql` for the outbox DDL, `REVIEW.md` for the buggy implementation review, and `ANSWERS.md` for the written questions.

```bash
go build ./...
go test ./...
```
