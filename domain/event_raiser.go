package domain

// DomainEvent is the common contract for events raised by aggregates.
type DomainEvent interface {
	EventType() string
}

// EventRaiser collects events during aggregate operations. Events are not
// published here; they're read by the repository to write outbox entries.
type EventRaiser struct {
	events []DomainEvent
}

// Raise records a domain event. Call from aggregate methods (e.g. Complete).
func (e *EventRaiser) Raise(ev DomainEvent) {
	e.events = append(e.events, ev)
}

// Events returns all raised events and clears the buffer so the same
// aggregate can be used again without re-emitting.
func (e *EventRaiser) Events() []DomainEvent {
	out := e.events
	e.events = nil
	return out
}
