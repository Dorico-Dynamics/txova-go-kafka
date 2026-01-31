package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
	"github.com/Dorico-Dynamics/txova-go-kafka/events"
)

func TestNew_NilConfig(t *testing.T) {
	t.Parallel()

	// New with nil config should use defaults but fail validation (no brokers)
	_, err := New(nil, nil, nil)
	if err == nil {
		t.Error("New(nil, nil, nil) should return error for missing brokers")
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config *Config
	}{
		{
			name:   "no brokers",
			config: &Config{GroupID: "test", Topics: []string{"topic"}},
		},
		{
			name:   "no group ID",
			config: &Config{Brokers: []string{"localhost:9092"}, Topics: []string{"topic"}},
		},
		{
			name:   "no topics",
			config: &Config{Brokers: []string{"localhost:9092"}, GroupID: "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(tt.config, nil, nil)
			if err == nil {
				t.Error("New() should return error for invalid config")
			}
		})
	}
}

func TestConsumerGroupHandler_invokeHandler_Success(t *testing.T) {
	t.Parallel()

	handler := &consumerGroupHandler{
		consumer: &Consumer{},
	}

	called := false
	h := func(ctx context.Context, env *envelope.Envelope) error {
		called = true
		return nil
	}

	err := handler.invokeHandler(context.Background(), h, nil)
	if err != nil {
		t.Errorf("invokeHandler() error = %v, want nil", err)
	}
	if !called {
		t.Error("handler should have been called")
	}
}

func TestConsumerGroupHandler_invokeHandler_Error(t *testing.T) {
	t.Parallel()

	handler := &consumerGroupHandler{
		consumer: &Consumer{},
	}

	expectedErr := errors.New("handler error")
	h := func(ctx context.Context, env *envelope.Envelope) error {
		return expectedErr
	}

	err := handler.invokeHandler(context.Background(), h, nil)
	if err != expectedErr {
		t.Errorf("invokeHandler() error = %v, want %v", err, expectedErr)
	}
}

func TestConsumerGroupHandler_invokeHandler_Panic(t *testing.T) {
	t.Parallel()

	handler := &consumerGroupHandler{
		consumer: &Consumer{},
	}

	h := func(ctx context.Context, env *envelope.Envelope) error {
		panic("test panic")
	}

	err := handler.invokeHandler(context.Background(), h, nil)
	if err == nil {
		t.Error("invokeHandler() should return error on panic")
	}
	if !errors.Is(err, ErrHandlerPanic) {
		t.Errorf("error should wrap ErrHandlerPanic, got: %v", err)
	}
}

func createTestEnvelope(t *testing.T, eventType string) *envelope.Envelope {
	t.Helper()

	env, err := envelope.New(&envelope.Config{
		Type:          eventType,
		Version:       "1.0",
		Source:        "test-service",
		CorrelationID: "corr-123",
		Payload:       map[string]string{"test": "value"},
	})
	if err != nil {
		t.Fatalf("failed to create envelope: %v", err)
	}
	return env
}

func TestEnvelopeUnmarshal(t *testing.T) {
	t.Parallel()

	env := createTestEnvelope(t, string(events.EventTypeUserRegistered))

	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("failed to marshal envelope: %v", err)
	}

	var decoded envelope.Envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal envelope: %v", err)
	}

	if decoded.Type != env.Type {
		t.Errorf("Type = %q, want %q", decoded.Type, env.Type)
	}
	if decoded.ID != env.ID {
		t.Errorf("ID = %q, want %q", decoded.ID, env.ID)
	}
}

func TestOffsetResetConstants(t *testing.T) {
	t.Parallel()

	if OffsetResetEarliest != "earliest" {
		t.Errorf("OffsetResetEarliest = %q, want %q", OffsetResetEarliest, "earliest")
	}
	if OffsetResetLatest != "latest" {
		t.Errorf("OffsetResetLatest = %q, want %q", OffsetResetLatest, "latest")
	}
}

func TestConsumer_StartClosed(t *testing.T) {
	t.Parallel()

	c := &Consumer{
		closed: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := c.Start(ctx)
	if !errors.Is(err, ErrConsumerClosed) {
		t.Errorf("Start() on closed consumer error = %v, want %v", err, ErrConsumerClosed)
	}
}

func TestConsumer_CloseIdempotent(t *testing.T) {
	t.Parallel()

	// Create a consumer with closed=true to test idempotent close
	c := &Consumer{
		closed: true,
	}

	err := c.Close()
	if err != nil {
		t.Errorf("Close() on already closed consumer error = %v, want nil", err)
	}
}
