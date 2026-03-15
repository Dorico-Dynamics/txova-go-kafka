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
	// ErrConfigRequired is returned when a nil config is provided.
	ErrConfigRequired = errors.New("config is required")
	// ErrCheckerNotInitialized is returned when the health checker has nil client or admin.
	ErrCheckerNotInitialized = errors.New("checker is not initialized")
	// ErrNoPartitionAssignments is returned when the consumer group has no active partition assignments.
	ErrNoPartitionAssignments = errors.New("consumer group has no active partition assignments")
	// ErrGroupNotFound is returned when the consumer group cannot be found.
	ErrGroupNotFound = errors.New("consumer group not found")
	// ErrMissingCommittedOffset is returned when a committed offset block is missing.
	ErrMissingCommittedOffset = errors.New("missing committed offset")
)
