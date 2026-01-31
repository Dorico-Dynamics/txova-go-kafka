package consumer

import (
	"context"
	"errors"
	"testing"

	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
	"github.com/Dorico-Dynamics/txova-go-kafka/events"
)

func TestNewHandlerRegistry(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()
	if registry == nil {
		t.Fatal("NewHandlerRegistry() returned nil")
	}
	if registry.handlers == nil {
		t.Error("handlers map should be initialized")
	}
}

func TestHandlerRegistry_Register(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()

	handler := func(ctx context.Context, env *envelope.Envelope) error {
		return nil
	}

	registry.Register(events.EventTypeUserRegistered, handler)

	if !registry.Has(events.EventTypeUserRegistered) {
		t.Error("handler should be registered for user.registered")
	}
}

func TestHandlerRegistry_Get(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()

	expectedErr := errors.New("test error")
	handler := func(ctx context.Context, env *envelope.Envelope) error {
		return expectedErr
	}

	registry.Register(events.EventTypeUserRegistered, handler)

	got := registry.Get(events.EventTypeUserRegistered)
	if got == nil {
		t.Fatal("Get() returned nil for registered handler")
	}

	// Verify it's the right handler by calling it
	err := got(context.Background(), nil)
	if err != expectedErr {
		t.Errorf("handler returned %v, want %v", err, expectedErr)
	}
}

func TestHandlerRegistry_Get_NotFound(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()

	got := registry.Get(events.EventTypeUserRegistered)
	if got != nil {
		t.Error("Get() should return nil for unregistered handler")
	}
}

func TestHandlerRegistry_RegisterWildcard(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()

	wildcardCalled := false
	wildcard := func(ctx context.Context, env *envelope.Envelope) error {
		wildcardCalled = true
		return nil
	}

	registry.RegisterWildcard(wildcard)

	// Should return wildcard for any event type
	got := registry.Get(events.EventTypeUserRegistered)
	if got == nil {
		t.Fatal("Get() should return wildcard handler")
	}

	_ = got(context.Background(), nil)
	if !wildcardCalled {
		t.Error("wildcard handler should have been called")
	}
}

func TestHandlerRegistry_Get_SpecificOverWildcard(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()

	specificCalled := false
	specific := func(ctx context.Context, env *envelope.Envelope) error {
		specificCalled = true
		return nil
	}

	wildcardCalled := false
	wildcard := func(ctx context.Context, env *envelope.Envelope) error {
		wildcardCalled = true
		return nil
	}

	registry.Register(events.EventTypeUserRegistered, specific)
	registry.RegisterWildcard(wildcard)

	// Should return specific handler, not wildcard
	got := registry.Get(events.EventTypeUserRegistered)
	_ = got(context.Background(), nil)

	if !specificCalled {
		t.Error("specific handler should have been called")
	}
	if wildcardCalled {
		t.Error("wildcard handler should not have been called")
	}
}

func TestHandlerRegistry_Has(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()

	// No handlers registered
	if registry.Has(events.EventTypeUserRegistered) {
		t.Error("Has() should return false for unregistered handler")
	}

	// Register specific handler
	registry.Register(events.EventTypeUserRegistered, func(ctx context.Context, env *envelope.Envelope) error {
		return nil
	})

	if !registry.Has(events.EventTypeUserRegistered) {
		t.Error("Has() should return true for registered handler")
	}

	// Different event type should return false (no wildcard)
	if registry.Has(events.EventTypeDriverApproved) {
		t.Error("Has() should return false for different event type")
	}
}

func TestHandlerRegistry_Has_WithWildcard(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()
	registry.RegisterWildcard(func(ctx context.Context, env *envelope.Envelope) error {
		return nil
	})

	// With wildcard, Has should return true for any event type
	if !registry.Has(events.EventTypeUserRegistered) {
		t.Error("Has() should return true when wildcard is registered")
	}
	if !registry.Has(events.EventTypeDriverApproved) {
		t.Error("Has() should return true when wildcard is registered")
	}
}

func TestHandlerRegistry_RegisteredTypes(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()

	handler := func(ctx context.Context, env *envelope.Envelope) error {
		return nil
	}

	registry.Register(events.EventTypeUserRegistered, handler)
	registry.Register(events.EventTypeDriverApproved, handler)
	registry.Register(events.EventTypeRideCompleted, handler)

	types := registry.RegisteredTypes()

	if len(types) != 3 {
		t.Errorf("RegisteredTypes() returned %d types, want 3", len(types))
	}

	// Check all expected types are present
	typeSet := make(map[events.EventType]bool)
	for _, et := range types {
		typeSet[et] = true
	}

	expectedTypes := []events.EventType{
		events.EventTypeUserRegistered,
		events.EventTypeDriverApproved,
		events.EventTypeRideCompleted,
	}

	for _, expected := range expectedTypes {
		if !typeSet[expected] {
			t.Errorf("RegisteredTypes() missing %q", expected)
		}
	}
}

func TestHandlerRegistry_RegisteredTypes_Empty(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()

	types := registry.RegisteredTypes()
	if len(types) != 0 {
		t.Errorf("RegisteredTypes() returned %d types for empty registry, want 0", len(types))
	}
}
