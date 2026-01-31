package dlq

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"

	"github.com/IBM/sarama"

	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
)

// mockSyncProducer implements sarama.SyncProducer for testing.
type mockSyncProducer struct {
	mu           sync.Mutex
	messages     []*sarama.ProducerMessage
	sendErr      error
	partitionSeq int32
	offsetSeq    int64
}

func newMockSyncProducer() *mockSyncProducer {
	return &mockSyncProducer{
		messages: make([]*sarama.ProducerMessage, 0),
	}
}

func (m *mockSyncProducer) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.sendErr != nil {
		return 0, 0, m.sendErr
	}

	m.messages = append(m.messages, msg)
	m.partitionSeq++
	m.offsetSeq++
	return m.partitionSeq - 1, m.offsetSeq - 1, nil
}

func (m *mockSyncProducer) SendMessages(msgs []*sarama.ProducerMessage) error {
	for _, msg := range msgs {
		if _, _, err := m.SendMessage(msg); err != nil {
			return err
		}
	}
	return nil
}

func (m *mockSyncProducer) Close() error {
	return nil
}

func (m *mockSyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return sarama.ProducerTxnFlagReady
}

func (m *mockSyncProducer) IsTransactional() bool {
	return false
}

func (m *mockSyncProducer) BeginTxn() error {
	return nil
}

func (m *mockSyncProducer) CommitTxn() error {
	return nil
}

func (m *mockSyncProducer) AbortTxn() error {
	return nil
}

func (m *mockSyncProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupID string) error {
	return nil
}

func (m *mockSyncProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupID string, metadata *string) error {
	return nil
}

func TestNewHandler(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	handler := NewHandler(mock, nil)

	if handler == nil {
		t.Fatal("NewHandler() returned nil")
	}
	if handler.producer != mock {
		t.Error("producer should be set")
	}
	if handler.logger == nil {
		t.Error("logger should be set to default")
	}
}

func TestNewHandler_WithLogger(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	logger := slog.Default()
	handler := NewHandler(mock, logger)

	if handler.logger != logger {
		t.Error("logger should be the provided logger")
	}
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	handler := NewHandler(mock, nil)

	env, _ := envelope.New(&envelope.Config{
		Type:          "user.registered",
		Version:       "1.0",
		Source:        "test-service",
		CorrelationID: "corr-123",
		Payload:       map[string]string{"user_id": "user-1"},
	})

	processingErr := errors.New("handler failed")
	ctx := context.Background()

	err := handler.Handle(ctx, "txova.users.v1", 0, 100, env, processingErr, 3)
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()

	if len(mock.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mock.messages))
	}

	msg := mock.messages[0]

	// Verify DLQ topic
	if msg.Topic != "txova.users.v1.dlq" {
		t.Errorf("topic = %q, want %q", msg.Topic, "txova.users.v1.dlq")
	}

	// Verify key is event ID
	key, _ := msg.Key.Encode()
	if string(key) != env.ID {
		t.Errorf("key = %q, want %q", string(key), env.ID)
	}

	// Verify headers
	headers := make(map[string]string)
	for _, h := range msg.Headers {
		headers[string(h.Key)] = string(h.Value)
	}

	if headers["original_topic"] != "txova.users.v1" {
		t.Errorf("original_topic header = %q, want %q", headers["original_topic"], "txova.users.v1")
	}
	if headers["event_type"] != env.Type {
		t.Errorf("event_type header = %q, want %q", headers["event_type"], env.Type)
	}
	if headers["event_id"] != env.ID {
		t.Errorf("event_id header = %q, want %q", headers["event_id"], env.ID)
	}
	if headers["error_message"] != "handler failed" {
		t.Errorf("error_message header = %q, want %q", headers["error_message"], "handler failed")
	}

	// Verify payload is a valid DLQ message
	value, _ := msg.Value.Encode()
	var dlqMsg Message
	if err := json.Unmarshal(value, &dlqMsg); err != nil {
		t.Errorf("failed to unmarshal DLQ message: %v", err)
	}

	if dlqMsg.OriginalTopic != "txova.users.v1" {
		t.Errorf("DLQ message OriginalTopic = %q, want %q", dlqMsg.OriginalTopic, "txova.users.v1")
	}
	if dlqMsg.OriginalOffset != 100 {
		t.Errorf("DLQ message OriginalOffset = %d, want 100", dlqMsg.OriginalOffset)
	}
	if dlqMsg.AttemptCount != 3 {
		t.Errorf("DLQ message AttemptCount = %d, want 3", dlqMsg.AttemptCount)
	}
}

func TestHandler_Handle_SendError(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	mock.sendErr = errors.New("kafka unavailable")
	handler := NewHandler(mock, nil)

	env, _ := envelope.New(&envelope.Config{
		Type:          "user.registered",
		Version:       "1.0",
		Source:        "test-service",
		CorrelationID: "corr-123",
		Payload:       map[string]string{"user_id": "user-1"},
	})

	ctx := context.Background()

	err := handler.Handle(ctx, "txova.users.v1", 0, 100, env, errors.New("original error"), 3)
	if err == nil {
		t.Error("Handle() should return error when send fails")
	}
}

func TestHandler_CreateConsumerDLQHandler(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	handler := NewHandler(mock, nil)

	env, _ := envelope.New(&envelope.Config{
		Type:          "user.registered",
		Version:       "1.0",
		Source:        "test-service",
		CorrelationID: "corr-123",
		Payload:       map[string]string{"user_id": "user-1"},
	})

	dlqHandler := handler.CreateConsumerDLQHandler("txova.users.v1", 0, 100, 3)
	if dlqHandler == nil {
		t.Fatal("CreateConsumerDLQHandler() returned nil")
	}

	// Call the handler
	ctx := context.Background()
	dlqHandler(ctx, env, errors.New("processing error"))

	mock.mu.Lock()
	defer mock.mu.Unlock()

	if len(mock.messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(mock.messages))
	}
}
