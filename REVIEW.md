# Code Review: Buggy Order Completion

## 1. Why direct publish breaks reliability

The code updates the order in the database and then publishes the event in a separate step. Those are two independent operations with no atomicity guarantee. If the second step fails, the system is left in an inconsistent state: the order is already marked completed in the database, but no event was published. Downstream consumers (notifications, inventory, analytics) never see the completion, so the business outcome and the event stream diverge. There is no way to "retry" the publish in a well-defined way without either re-applying the order update (wrong) or inventing a separate reconciliation path.

Reliability here means: whenever the order is persisted as completed, a corresponding event must eventually be delivered. Direct publish does not guarantee that, because the two writes are not tied together.

## 2. Exact failure scenario

1. `uc.db.Update(order)` succeeds — the order row is committed (or will be when the transaction commits, depending on driver/tx scope; in the snippet there’s no explicit transaction, so assume auto-commit per call).
2. The process crashes, or the network fails, or the event bus is temporarily unavailable, before `uc.eventBus.Publish(event)` is called or before it succeeds.
3. Result: the order is completed in the database, but no `OrderCompletedEvent` is ever published. Any consumer that depends on that event will have a permanently missing message. The only way to fix it is out-of-band (e.g. scanning for completed orders without events and backfilling), which is complex and error-prone.

Even if both calls are in the same process and the second one is retried, you still have a window where the DB shows "completed" and the event stream does not. So the system is not reliable from the perspective of event-driven consumers.

## 3. Why outbox must be in the same transaction

The outbox row must be written in the **same** database transaction as the aggregate (order) update. That way:

- Either **both** the order update and the outbox insert commit, or **neither** does.
- There is no state where "order is completed" but "no outbox row exists." The outbox becomes the single source of truth for "what must be published."
- A background worker can periodically read unprocessed outbox rows and publish them. If the worker crashes after publishing but before marking the row processed, it will see the same row again on restart and republish (at-least-once). Consumers must handle duplicates; the important guarantee is that no event is lost when the order is committed.

If the outbox were written in a different transaction (or after the order commit), we’d be back to the same problem: the order could be committed and then the outbox write could fail, leaving no record that an event is owed. Same-transaction outbox ensures that whenever the business state is persisted, the obligation to publish is persisted with it.
