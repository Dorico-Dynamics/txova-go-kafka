package producer

import "errors"

// Producer errors.
var (
	// ErrNoBrokers is returned when no brokers are configured.
	ErrNoBrokers = errors.New("no brokers configured")
	// ErrNoClientID is returned when no client ID is configured.
	ErrNoClientID = errors.New("client ID is required")
	// ErrProducerClosed is returned when the producer is already closed.
	ErrProducerClosed = errors.New("producer is closed")
	// ErrNilEnvelope is returned when attempting to publish a nil envelope.
	ErrNilEnvelope = errors.New("envelope cannot be nil")
	// ErrInvalidEnvelope is returned when the envelope fails validation.
	ErrInvalidEnvelope = errors.New("invalid envelope")
)
