// Package envelope provides the standard event envelope format for Kafka messages.
// Every event published to Kafka must use this envelope format for consistency
// and traceability across the Txova platform.
package envelope

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	txctx "github.com/Dorico-Dynamics/txova-go-core/context"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
)

// semverRegex validates version strings in semver format (major.minor).
var semverRegex = regexp.MustCompile(`^\d+\.\d+$`)

// ActorType represents the type of actor that triggered an event.
type ActorType string

const (
	// ActorTypeUser indicates the actor is a regular user.
	ActorTypeUser ActorType = "user"
	// ActorTypeSystem indicates the actor is the system itself.
	ActorTypeSystem ActorType = "system"
	// ActorTypeAdmin indicates the actor is an administrator.
	ActorTypeAdmin ActorType = "admin"
)

// Actor represents who triggered an event.
type Actor struct {
	// ID is the actor's user ID.
	ID ids.UserID `json:"id"`
	// Type is the actor type (user, system, admin).
	Type ActorType `json:"type"`
}

// IsZero returns true if the actor is incomplete (missing ID or Type).
// A valid actor must have both an ID and a Type.
func (a Actor) IsZero() bool {
	return a.ID.IsZero() || a.Type == ""
}

// Envelope is the standard wrapper for all Kafka events.
// It provides metadata for tracing, versioning, and event routing.
type Envelope struct {
	// ID is the unique event identifier (UUID).
	ID string `json:"id"`
	// Type is the event type (e.g., "ride.completed").
	Type string `json:"type"`
	// Version is the schema version in semver format (e.g., "1.0").
	Version string `json:"version"`
	// Source is the name of the service that produced the event.
	Source string `json:"source"`
	// Timestamp is when the event was created (UTC).
	Timestamp time.Time `json:"timestamp"`
	// CorrelationID links related events across services.
	CorrelationID string `json:"correlation_id"`
	// CausationID is the ID of the event that caused this event (optional).
	CausationID string `json:"causation_id,omitempty"`
	// Actor is who triggered the event (optional).
	Actor *Actor `json:"actor,omitempty"`
	// Payload contains the event-specific data.
	Payload json.RawMessage `json:"payload"`
}

// Config holds the configuration for creating a new envelope.
type Config struct {
	// Type is the event type (required).
	Type string
	// Version is the schema version (required).
	Version string
	// Source is the producing service name (required).
	Source string
	// CorrelationID links related events (optional, can be extracted from context).
	CorrelationID string
	// CausationID is the ID of the causing event (optional).
	CausationID string
	// Actor is who triggered the event (optional).
	Actor *Actor
	// Payload is the event-specific data (required).
	Payload any
}

// New creates a new envelope with the given configuration.
// It generates a new UUID for the event ID and sets the timestamp to now (UTC).
func New(cfg *Config) (*Envelope, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	eventID, err := generateEventID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate event ID: %w", err)
	}

	payloadBytes, err := json.Marshal(cfg.Payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &Envelope{
		ID:            eventID,
		Type:          cfg.Type,
		Version:       cfg.Version,
		Source:        cfg.Source,
		Timestamp:     time.Now().UTC(),
		CorrelationID: cfg.CorrelationID,
		CausationID:   cfg.CausationID,
		Actor:         cfg.Actor,
		Payload:       payloadBytes,
	}, nil
}

// NewWithContext creates a new envelope, extracting the correlation ID from context
// if not provided in the config.
func NewWithContext(ctx context.Context, cfg *Config) (*Envelope, error) {
	if cfg.CorrelationID == "" {
		cfg.CorrelationID = txctx.CorrelationID(ctx)
	}
	// Fall back to request ID if correlation ID is still empty
	if cfg.CorrelationID == "" {
		cfg.CorrelationID = txctx.RequestID(ctx)
	}

	return New(cfg)
}

// validateConfig validates the envelope configuration.
func validateConfig(cfg *Config) error {
	if cfg.Type == "" {
		return fmt.Errorf("event type is required")
	}
	if cfg.Version == "" {
		return fmt.Errorf("version is required")
	}
	if !semverRegex.MatchString(cfg.Version) {
		return fmt.Errorf("version must be in semver format (major.minor), got: %s", cfg.Version)
	}
	if cfg.Source == "" {
		return fmt.Errorf("source is required")
	}
	if cfg.Payload == nil {
		return fmt.Errorf("payload is required")
	}
	return nil
}

// generateEventID generates a new UUID string for the event ID.
func generateEventID() (string, error) {
	id, err := ids.NewUserID() // Reusing UUID generation from ids package
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// Validate validates that the envelope has all required fields.
func (e *Envelope) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("id is required")
	}
	if e.Type == "" {
		return fmt.Errorf("type is required")
	}
	if e.Version == "" {
		return fmt.Errorf("version is required")
	}
	if !semverRegex.MatchString(e.Version) {
		return fmt.Errorf("version must be in semver format (major.minor), got: %s", e.Version)
	}
	if e.Source == "" {
		return fmt.Errorf("source is required")
	}
	if e.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}
	if e.CorrelationID == "" {
		return fmt.Errorf("correlation_id is required")
	}
	if len(e.Payload) == 0 {
		return fmt.Errorf("payload is required")
	}
	return nil
}

// UnmarshalPayload unmarshals the payload into the provided destination.
func (e *Envelope) UnmarshalPayload(dest any) error {
	if dest == nil {
		return fmt.Errorf("destination cannot be nil")
	}
	if len(e.Payload) == 0 {
		return fmt.Errorf("payload is empty")
	}
	return json.Unmarshal(e.Payload, dest)
}

// WithCorrelationID returns a copy of the envelope with the given correlation ID.
func (e *Envelope) WithCorrelationID(correlationID string) *Envelope {
	dup := *e
	dup.CorrelationID = correlationID
	return &dup
}

// WithCausationID returns a copy of the envelope with the given causation ID.
func (e *Envelope) WithCausationID(causationID string) *Envelope {
	dup := *e
	dup.CausationID = causationID
	return &dup
}

// WithActor returns a copy of the envelope with the given actor.
func (e *Envelope) WithActor(actor *Actor) *Envelope {
	dup := *e
	dup.Actor = actor
	return &dup
}

// EventDomain extracts the domain from the event type (e.g., "ride" from "ride.completed").
func (e *Envelope) EventDomain() string {
	for i, c := range e.Type {
		if c == '.' {
			return e.Type[:i]
		}
	}
	return e.Type
}

// EventAction extracts the action from the event type (e.g., "completed" from "ride.completed").
func (e *Envelope) EventAction() string {
	for i, c := range e.Type {
		if c == '.' {
			return e.Type[i+1:]
		}
	}
	return ""
}
