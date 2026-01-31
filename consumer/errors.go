package consumer

import "errors"

// Consumer errors.
var (
	// ErrNoBrokers is returned when no brokers are configured.
	ErrNoBrokers = errors.New("no brokers configured")
	// ErrNoGroupID is returned when no group ID is configured.
	ErrNoGroupID = errors.New("group ID is required")
	// ErrNoTopics is returned when no topics are configured.
	ErrNoTopics = errors.New("at least one topic is required")
	// ErrConsumerClosed is returned when the consumer is already closed.
	ErrConsumerClosed = errors.New("consumer is closed")
	// ErrNoHandler is returned when no handler is registered for an event type.
	ErrNoHandler = errors.New("no handler registered for event type")
	// ErrHandlerPanic is returned when a handler panics.
	ErrHandlerPanic = errors.New("handler panicked")
)
