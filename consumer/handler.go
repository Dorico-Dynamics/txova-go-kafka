package consumer

import (
	"context"

	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
	"github.com/Dorico-Dynamics/txova-go-kafka/events"
)

// Handler processes an event envelope.
// Return an error to trigger retry/DLQ behavior.
// Handlers must be idempotent as messages may be redelivered.
type Handler func(ctx context.Context, env *envelope.Envelope) error

// HandlerRegistry manages event handlers.
type HandlerRegistry struct {
	handlers        map[events.EventType]Handler
	wildcardHandler Handler
}

// NewHandlerRegistry creates a new handler registry.
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[events.EventType]Handler),
	}
}

// Register registers a handler for a specific event type.
func (r *HandlerRegistry) Register(eventType events.EventType, handler Handler) {
	r.handlers[eventType] = handler
}

// RegisterWildcard registers a handler that matches all event types
// within a domain (e.g., "user.*" matches all user events).
// This is called when no specific handler is found.
func (r *HandlerRegistry) RegisterWildcard(handler Handler) {
	r.wildcardHandler = handler
}

// Get returns the handler for the given event type.
// Returns the specific handler if registered, otherwise the wildcard handler.
// Returns nil if no handler is found.
func (r *HandlerRegistry) Get(eventType events.EventType) Handler {
	if handler, ok := r.handlers[eventType]; ok {
		return handler
	}
	return r.wildcardHandler
}

// Has returns true if a handler is registered for the event type.
func (r *HandlerRegistry) Has(eventType events.EventType) bool {
	if _, ok := r.handlers[eventType]; ok {
		return true
	}
	return r.wildcardHandler != nil
}

// RegisteredTypes returns all registered event types.
func (r *HandlerRegistry) RegisteredTypes() []events.EventType {
	types := make([]events.EventType, 0, len(r.handlers))
	for eventType := range r.handlers {
		types = append(types, eventType)
	}
	return types
}
