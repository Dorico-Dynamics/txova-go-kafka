package dlq

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
)

func createTestEnvelope(t *testing.T) *envelope.Envelope {
	t.Helper()

	env, err := envelope.New(&envelope.Config{
		Type:          "user.registered",
		Version:       "1.0",
		Source:        "test-service",
		CorrelationID: "corr-123",
		Payload:       map[string]string{"user_id": "user-1"},
	})
	if err != nil {
		t.Fatalf("failed to create envelope: %v", err)
	}
	return env
}

func TestNewMessage(t *testing.T) {
	t.Parallel()

	env := createTestEnvelope(t)
	processingErr := errors.New("processing failed")

	msg := NewMessage(
		"txova.users.v1",
		0,
		100,
		env,
		processingErr,
		3,
	)

	if msg.OriginalTopic != "txova.users.v1" {
		t.Errorf("OriginalTopic = %q, want %q", msg.OriginalTopic, "txova.users.v1")
	}
	if msg.OriginalPartition != 0 {
		t.Errorf("OriginalPartition = %d, want 0", msg.OriginalPartition)
	}
	if msg.OriginalOffset != 100 {
		t.Errorf("OriginalOffset = %d, want 100", msg.OriginalOffset)
	}
	if msg.ErrorMessage != "processing failed" {
		t.Errorf("ErrorMessage = %q, want %q", msg.ErrorMessage, "processing failed")
	}
	if msg.AttemptCount != 3 {
		t.Errorf("AttemptCount = %d, want 3", msg.AttemptCount)
	}
	if msg.Envelope != env {
		t.Error("Envelope should be the same as input")
	}
	if msg.FirstFailure.IsZero() {
		t.Error("FirstFailure should be set")
	}
	if msg.LastFailure.IsZero() {
		t.Error("LastFailure should be set")
	}
}

func TestMessage_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	env := createTestEnvelope(t)
	processingErr := errors.New("processing failed")

	original := NewMessage(
		"txova.users.v1",
		2,
		500,
		env,
		processingErr,
		5,
	)

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	// Unmarshal back
	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	// Verify fields
	if decoded.OriginalTopic != original.OriginalTopic {
		t.Errorf("OriginalTopic = %q, want %q", decoded.OriginalTopic, original.OriginalTopic)
	}
	if decoded.OriginalPartition != original.OriginalPartition {
		t.Errorf("OriginalPartition = %d, want %d", decoded.OriginalPartition, original.OriginalPartition)
	}
	if decoded.OriginalOffset != original.OriginalOffset {
		t.Errorf("OriginalOffset = %d, want %d", decoded.OriginalOffset, original.OriginalOffset)
	}
	if decoded.ErrorMessage != original.ErrorMessage {
		t.Errorf("ErrorMessage = %q, want %q", decoded.ErrorMessage, original.ErrorMessage)
	}
	if decoded.AttemptCount != original.AttemptCount {
		t.Errorf("AttemptCount = %d, want %d", decoded.AttemptCount, original.AttemptCount)
	}
	if decoded.Envelope == nil {
		t.Fatal("Envelope should not be nil")
	}
	if decoded.Envelope.ID != original.Envelope.ID {
		t.Errorf("Envelope.ID = %q, want %q", decoded.Envelope.ID, original.Envelope.ID)
	}
}

func TestMessage_FirstFailureLastFailureEqual(t *testing.T) {
	t.Parallel()

	env := createTestEnvelope(t)
	processingErr := errors.New("error")

	msg := NewMessage("topic", 0, 0, env, processingErr, 1)

	// For a new message, FirstFailure and LastFailure should be equal
	if !msg.FirstFailure.Equal(msg.LastFailure) {
		t.Error("FirstFailure and LastFailure should be equal for new message")
	}
}

func TestMessage_TimestampsAreUTC(t *testing.T) {
	t.Parallel()

	env := createTestEnvelope(t)
	processingErr := errors.New("error")

	msg := NewMessage("topic", 0, 0, env, processingErr, 1)

	if msg.FirstFailure.Location() != time.UTC {
		t.Error("FirstFailure should be in UTC")
	}
	if msg.LastFailure.Location() != time.UTC {
		t.Error("LastFailure should be in UTC")
	}
}
