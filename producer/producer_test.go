package producer

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"

	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
	"github.com/Dorico-Dynamics/txova-go-kafka/topics"
)

// mockSyncProducer implements sarama.SyncProducer for testing.
type mockSyncProducer struct {
	mu           sync.Mutex
	messages     []*sarama.ProducerMessage
	sendErr      error
	closeErr     error
	closed       bool
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
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return m.closeErr
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

func newTestProducer(mock *mockSyncProducer) *Producer {
	return &Producer{
		syncProducer: mock,
		config: &Config{
			Brokers:  []string{"localhost:9092"},
			ClientID: "test-client",
		},
		logger: slog.Default(),
	}
}

func createValidEnvelope(t *testing.T) *envelope.Envelope {
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

func TestNew_NilConfig(t *testing.T) {
	t.Parallel()

	// New with nil config should use defaults but fail validation (no brokers)
	_, err := New(nil, nil)
	if err == nil {
		t.Error("New(nil, nil) should return error for missing brokers")
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
			config: &Config{ClientID: "test"},
		},
		{
			name:   "no client ID",
			config: &Config{Brokers: []string{"localhost:9092"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := New(tt.config, nil)
			if err == nil {
				t.Error("New() should return error for invalid config")
			}
		})
	}
}

func TestProducer_PublishToTopic(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	p := newTestProducer(mock)
	defer p.Close()

	env := createValidEnvelope(t)
	ctx := context.Background()

	err := p.PublishToTopic(ctx, topics.Users, env, "user-123")
	if err != nil {
		t.Fatalf("PublishToTopic() error = %v", err)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()

	if len(mock.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mock.messages))
	}

	msg := mock.messages[0]
	if msg.Topic != topics.Users.String() {
		t.Errorf("topic = %q, want %q", msg.Topic, topics.Users.String())
	}

	key, _ := msg.Key.Encode()
	if string(key) != "user-123" {
		t.Errorf("key = %q, want %q", string(key), "user-123")
	}

	// Verify headers
	headers := make(map[string]string)
	for _, h := range msg.Headers {
		headers[string(h.Key)] = string(h.Value)
	}

	if headers["event_type"] != env.Type {
		t.Errorf("event_type header = %q, want %q", headers["event_type"], env.Type)
	}
	if headers["event_id"] != env.ID {
		t.Errorf("event_id header = %q, want %q", headers["event_id"], env.ID)
	}
	if headers["correlation_id"] != env.CorrelationID {
		t.Errorf("correlation_id header = %q, want %q", headers["correlation_id"], env.CorrelationID)
	}

	// Verify payload is valid JSON
	value, _ := msg.Value.Encode()
	var decoded envelope.Envelope
	if err := json.Unmarshal(value, &decoded); err != nil {
		t.Errorf("failed to unmarshal message value: %v", err)
	}
}

func TestProducer_Publish(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	p := newTestProducer(mock)
	defer p.Close()

	env := createValidEnvelope(t)
	ctx := context.Background()

	err := p.Publish(ctx, env, "user-123")
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()

	if len(mock.messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(mock.messages))
	}

	// user.registered should go to users topic
	if mock.messages[0].Topic != topics.Users.String() {
		t.Errorf("topic = %q, want %q", mock.messages[0].Topic, topics.Users.String())
	}
}

func TestProducer_Publish_UnknownEventType(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	p := newTestProducer(mock)
	defer p.Close()

	env := &envelope.Envelope{
		ID:            "evt-123",
		Type:          "unknown.event",
		Version:       "1.0",
		Source:        "test-service",
		Timestamp:     time.Now().UTC(),
		CorrelationID: "corr-123",
		Payload:       json.RawMessage(`{}`),
	}

	ctx := context.Background()

	err := p.Publish(ctx, env, "key-123")
	if err == nil {
		t.Error("Publish() should return error for unknown event type")
	}
}

func TestProducer_PublishToTopic_NilEnvelope(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	p := newTestProducer(mock)
	defer p.Close()

	ctx := context.Background()

	err := p.PublishToTopic(ctx, topics.Users, nil, "key-123")
	if !errors.Is(err, ErrNilEnvelope) {
		t.Errorf("PublishToTopic(nil) error = %v, want %v", err, ErrNilEnvelope)
	}
}

func TestProducer_PublishToTopic_InvalidEnvelope(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	p := newTestProducer(mock)
	defer p.Close()

	env := &envelope.Envelope{
		// Missing required fields
		ID: "evt-123",
	}

	ctx := context.Background()

	err := p.PublishToTopic(ctx, topics.Users, env, "key-123")
	if err == nil {
		t.Error("PublishToTopic() should return error for invalid envelope")
	}
	if !errors.Is(err, ErrInvalidEnvelope) {
		t.Errorf("error should wrap ErrInvalidEnvelope, got: %v", err)
	}
}

func TestProducer_PublishToTopic_SendError(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	mock.sendErr = errors.New("kafka unavailable")
	p := newTestProducer(mock)
	defer p.Close()

	env := createValidEnvelope(t)
	ctx := context.Background()

	err := p.PublishToTopic(ctx, topics.Users, env, "key-123")
	if err == nil {
		t.Error("PublishToTopic() should return error when send fails")
	}
}

func TestProducer_PublishToTopic_Closed(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	p := newTestProducer(mock)

	// Close the producer
	p.Close()

	env := createValidEnvelope(t)
	ctx := context.Background()

	err := p.PublishToTopic(ctx, topics.Users, env, "key-123")
	if !errors.Is(err, ErrProducerClosed) {
		t.Errorf("PublishToTopic() after close error = %v, want %v", err, ErrProducerClosed)
	}
}

func TestProducer_PublishBatch(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	p := newTestProducer(mock)
	defer p.Close()

	envs := []*envelope.Envelope{
		createValidEnvelope(t),
		createValidEnvelope(t),
		createValidEnvelope(t),
	}
	keys := []string{"key-1", "key-2", "key-3"}

	ctx := context.Background()

	err := p.PublishBatch(ctx, envs, keys)
	if err != nil {
		t.Fatalf("PublishBatch() error = %v", err)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()

	if len(mock.messages) != 3 {
		t.Errorf("expected 3 messages, got %d", len(mock.messages))
	}
}

func TestProducer_PublishBatch_MismatchedLengths(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	p := newTestProducer(mock)
	defer p.Close()

	envs := []*envelope.Envelope{createValidEnvelope(t), createValidEnvelope(t)}
	keys := []string{"key-1"} // Different length

	ctx := context.Background()

	err := p.PublishBatch(ctx, envs, keys)
	if err == nil {
		t.Error("PublishBatch() should return error for mismatched lengths")
	}
}

func TestProducer_Close(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	p := newTestProducer(mock)

	err := p.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	mock.mu.Lock()
	closed := mock.closed
	mock.mu.Unlock()

	if !closed {
		t.Error("producer should be closed")
	}

	// Second close should be no-op
	err = p.Close()
	if err != nil {
		t.Errorf("second Close() error = %v, want nil", err)
	}
}

func TestProducer_Close_Error(t *testing.T) {
	t.Parallel()

	mock := newMockSyncProducer()
	mock.closeErr = errors.New("close failed")
	p := newTestProducer(mock)

	err := p.Close()
	if err == nil {
		t.Error("Close() should return error when underlying close fails")
	}
}
