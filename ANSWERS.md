# Answers

## Q1: At-least-once vs exactly-once; how should consumers handle?

The worker reads an event, publishes it, then crashes before it can set `processed_at`. On restart it reads the same row again and publishes again. So the same event can be delivered more than once — hence **at-least-once**, not exactly-once.

Consumers should handle this by being **idempotent**: processing the same event twice should have the same effect as processing it once. For example:

- Use a unique key (e.g. outbox `id` or a business idempotency key) and record "already processed" in the consumer’s store. If the key is seen again, skip or no-op.
- Or design the operation so that applying it twice is safe (e.g. "set status to X" rather than "increment counter" without a guard).

Exactly-once would require something like a two-phase protocol or transactional outbox with a single atomic "mark processed and commit consumer state" step, which is much harder and often not worth the complexity. At-least-once plus idempotent consumers is the usual approach.

---

## Q2: Outbox table schema

```sql
CREATE TABLE outbox (
    id           VARCHAR(36)  PRIMARY KEY,
    event_type   VARCHAR(128) NOT NULL,
    payload      JSONB        NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL,
    processed_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_unprocessed ON outbox (created_at)
    WHERE processed_at IS NULL;

CREATE INDEX idx_outbox_event_type_unprocessed ON outbox (event_type, created_at)
    WHERE processed_at IS NULL;
```

- **id**: Primary key; unique per event. Use for idempotency (consumer can store this and skip if already seen).
- **event_type**: So the worker or consumers can filter or route.
- **payload**: The serialized event (e.g. JSON).
- **created_at**: When the event was logically created (from the aggregate). Used for ordering and worker polling.
- **processed_at**: NULL = not yet published; set when the worker has successfully published. Partial index only on unprocessed rows keeps the index small and the "next batch" query fast.

**Preventing processing the same event twice:** The worker should only update `processed_at` after a successful publish. If it crashes before that, the row stays unprocessed and will be picked again. To avoid *publishing* the same event twice in a short window, the worker can use a single transaction: "SELECT ... FOR UPDATE SKIP LOCKED" for a batch, publish, then "UPDATE outbox SET processed_at = now() WHERE id IN (...)" and commit. The real deduplication is on the consumer side using `id` (or a business key in the payload) and idempotent handling.

---

## Q3: Out-of-order events (Completed then Cancelled)

**What goes wrong:** The consumer sees first "cancelled" and then "completed." So it may treat the order as cancelled and then later as completed again — wrong final state and possibly wrong side effects (e.g. charging the customer, restocking, then cancelling again).

**How to ensure ordering:**

1. **Single consumer per aggregate (e.g. per order):** Process events for a given order ID in a single stream or partition so that Cancelled always comes after Completed in the same order.
2. **Ordered outbox polling:** Worker selects from outbox ordered by `created_at` (and optionally by order_id or partition key) and publishes to a topic/queue that preserves order (e.g. Kafka partition keyed by order_id). Consumers then see events in order.
3. **Sequence or version in the event:** Include a sequence number or version per order. Consumers ignore an event if they have already processed a higher sequence for that order (e.g. ignore Completed if Cancelled was already applied).

---

## Q4: 10 million unprocessed events (consumer down for a week)

Strategy:

1. **Scale the worker:** Run more worker instances (or more threads) to drain the backlog faster. Ensure the outbox polling query uses the index and limits batch size so you don’t lock too many rows or overload the broker.
2. **Backpressure and prioritise:** If the consumer can’t keep up, consider processing in batches by time or priority; avoid blocking new events for too long.
3. **Temporary bypass or replay:** If the consumer is fixed and can catch up by replay, consider a one-off job that reads large batches (e.g. by `created_at` ranges), publishes in order, and marks processed. Monitor DB and broker load.
4. **SLO and alerting:** Alert on outbox depth and consumer lag so next time you catch it early. Optionally move old processed rows to an archive table to keep the main outbox small.

---

## Q5: Bug with `time.Now()` in OutboxMuts

Using `time.Now()` for `CreatedAt` when building the outbox entry is wrong because it uses the **persistence time** (when the row is being written), not the **event time** (when the order was actually completed). That breaks:

- **Ordering:** Events are ordered by when they were written, not when they occurred. If two events are written in a different order than they occurred (e.g. due to batching or retries), consumers see the wrong order.
- **Correctness:** Downstream may rely on "completed at" being the real completion time; storing "now" in the outbox blurs that.

**Fix:** Use the event’s own timestamp. For `OrderCompletedEvent`, use `event.CompletedAt` (or equivalent) when setting `created_at` on the outbox row. So the outbox row’s `created_at` should be set from the domain event’s time, not from `time.Now()`.
