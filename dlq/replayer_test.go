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

type mockReplayConsumerGroupSession struct {
	ctx            context.Context
	markedMessages []*sarama.ConsumerMessage
	mu             sync.Mutex
}

func newMockReplayConsumerGroupSession(ctx context.Context) *mockReplayConsumerGroupSession {
	return &mockReplayConsumerGroupSession{
		ctx:            ctx,
		markedMessages: make([]*sarama.ConsumerMessage, 0),
	}
}

func (m *mockReplayConsumerGroupSession) Claims() map[string][]int32 {
	return map[string][]int32{"txova.users.v1.dlq": {0}}
}

func (m *mockReplayConsumerGroupSession) MemberID() string {
	return "test-member"
}

func (m *mockReplayConsumerGroupSession) GenerationID() int32 {
	return 1
}

func (m *mockReplayConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
}

func (m *mockReplayConsumerGroupSession) Commit() {}

func (m *mockReplayConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
}

func (m *mockReplayConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markedMessages = append(m.markedMessages, msg)
}

func (m *mockReplayConsumerGroupSession) Context() context.Context {
	return m.ctx
}

func (m *mockReplayConsumerGroupSession) MarkedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.markedMessages)
}

type mockReplayConsumerGroupClaim struct {
	messages chan *sarama.ConsumerMessage
	topic    string
}

func newMockReplayConsumerGroupClaim(topic string, messages []*sarama.ConsumerMessage) *mockReplayConsumerGroupClaim {
	ch := make(chan *sarama.ConsumerMessage, len(messages))
	for _, msg := range messages {
		ch <- msg
	}
	close(ch)

	return &mockReplayConsumerGroupClaim{
		messages: ch,
		topic:    topic,
	}
}

func (m *mockReplayConsumerGroupClaim) Topic() string {
	return m.topic
}

func (m *mockReplayConsumerGroupClaim) Partition() int32 {
	return 0
}

func (m *mockReplayConsumerGroupClaim) InitialOffset() int64 {
	return 0
}

func (m *mockReplayConsumerGroupClaim) HighWaterMarkOffset() int64 {
	return 100
}

func (m *mockReplayConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	return m.messages
}

type mockReplayConsumerGroup struct {
	consumeFunc func(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error
	closeFunc   func() error
	errChan     chan error
	closed      bool
	mu          sync.Mutex
}

func newMockReplayConsumerGroup() *mockReplayConsumerGroup {
	return &mockReplayConsumerGroup{
		errChan: make(chan error),
	}
}

func (m *mockReplayConsumerGroup) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	if m.consumeFunc != nil {
		return m.consumeFunc(ctx, topics, handler)
	}
	return nil
}

func (m *mockReplayConsumerGroup) Errors() <-chan error {
	return m.errChan
}

func (m *mockReplayConsumerGroup) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockReplayConsumerGroup) Pause(partitions map[string][]int32) {}

func (m *mockReplayConsumerGroup) Resume(partitions map[string][]int32) {}

func (m *mockReplayConsumerGroup) PauseAll() {}

func (m *mockReplayConsumerGroup) ResumeAll() {}

func TestNewReplayerWithConsumerGroup(t *testing.T) {
	t.Parallel()

	group := newMockReplayConsumerGroup()
	handler := func(ctx context.Context, env *envelope.Envelope, msg *Message) error {
		return nil
	}

	replayer, err := NewReplayerWithConsumerGroup(group, "txova.users.v1.dlq", handler, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if replayer == nil {
		t.Fatal("expected replayer, got nil")
	}
	if replayer.logger == nil {
		t.Fatal("expected default logger")
	}
}

func TestNewReplayerWithConsumerGroup_Validation(t *testing.T) {
	t.Parallel()

	handler := func(ctx context.Context, env *envelope.Envelope, msg *Message) error {
		return nil
	}

	tests := []struct {
		name    string
		group   sarama.ConsumerGroup
		topic   string
		handler ReplayHandler
		wantErr error
	}{
		{name: "nil group", topic: "txova.users.v1.dlq", handler: handler},
		{name: "empty topic", group: newMockReplayConsumerGroup(), handler: handler, wantErr: ErrEmptyDLQTopic},
		{name: "nil handler", group: newMockReplayConsumerGroup(), topic: "txova.users.v1.dlq", wantErr: ErrNilReplayHandler},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewReplayerWithConsumerGroup(tt.group, tt.topic, tt.handler, slog.Default())
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestReplayerGroupHandler_ProcessMessage_Success(t *testing.T) {
	t.Parallel()

	replayed := false
	replayer := &Replayer{
		handler: func(ctx context.Context, env *envelope.Envelope, msg *Message) error {
			replayed = true
			return nil
		},
		logger: slog.Default(),
	}

	handler := &replayerGroupHandler{replayer: replayer}
	session := newMockReplayConsumerGroupSession(context.Background())
	msg := buildDLQConsumerMessage(t, "ATXid_123")

	handler.processMessage(session, msg)

	if !replayed {
		t.Fatal("expected replay handler to be called")
	}
	if session.MarkedCount() != 1 {
		t.Fatalf("expected message to be marked once, got %d", session.MarkedCount())
	}

	stats := replayer.Stats()
	if stats.Consumed != 1 || stats.Replayed != 1 || stats.Skipped != 0 || stats.Failed != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestReplayerGroupHandler_ProcessMessage_Filtered(t *testing.T) {
	t.Parallel()

	replayer := &Replayer{
		handler: func(ctx context.Context, env *envelope.Envelope, msg *Message) error {
			t.Fatal("handler should not be called for filtered messages")
			return nil
		},
		logger: slog.Default(),
	}

	handler := &replayerGroupHandler{
		replayer: replayer,
		filter: func(msg *Message) bool {
			return false
		},
	}
	session := newMockReplayConsumerGroupSession(context.Background())

	handler.processMessage(session, buildDLQConsumerMessage(t, "ATXid_123"))

	if session.MarkedCount() != 1 {
		t.Fatalf("expected filtered message to be marked, got %d", session.MarkedCount())
	}

	stats := replayer.Stats()
	if stats.Consumed != 1 || stats.Skipped != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestReplayerGroupHandler_ProcessMessage_InvalidPayload(t *testing.T) {
	t.Parallel()

	replayer := &Replayer{
		handler: func(ctx context.Context, env *envelope.Envelope, msg *Message) error {
			t.Fatal("handler should not be called for invalid payload")
			return nil
		},
		logger: slog.Default(),
	}
	handler := &replayerGroupHandler{replayer: replayer}
	session := newMockReplayConsumerGroupSession(context.Background())
	msg := &sarama.ConsumerMessage{
		Topic: "txova.users.v1.dlq",
		Value: []byte(`not-json`),
	}

	handler.processMessage(session, msg)

	if session.MarkedCount() != 1 {
		t.Fatalf("expected invalid payload to be marked, got %d", session.MarkedCount())
	}

	stats := replayer.Stats()
	if stats.Consumed != 1 || stats.Skipped != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestReplayerGroupHandler_ProcessMessage_HandlerError(t *testing.T) {
	t.Parallel()

	replayer := &Replayer{
		handler: func(ctx context.Context, env *envelope.Envelope, msg *Message) error {
			return errors.New("replay failed")
		},
		logger: slog.Default(),
	}
	handler := &replayerGroupHandler{replayer: replayer}
	session := newMockReplayConsumerGroupSession(context.Background())

	handler.processMessage(session, buildDLQConsumerMessage(t, "ATXid_123"))

	if session.MarkedCount() != 0 {
		t.Fatalf("expected failed replay to remain unmarked, got %d", session.MarkedCount())
	}

	stats := replayer.Stats()
	if stats.Consumed != 1 || stats.Failed != 1 || stats.Replayed != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestReplayer_ReplayWithFilter(t *testing.T) {
	t.Parallel()

	group := newMockReplayConsumerGroup()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	replayed := false
	group.consumeFunc = func(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
		if len(topics) != 1 || topics[0] != "txova.users.v1.dlq" {
			t.Fatalf("unexpected topics: %v", topics)
		}

		session := newMockReplayConsumerGroupSession(ctx)
		claim := newMockReplayConsumerGroupClaim("txova.users.v1.dlq", []*sarama.ConsumerMessage{
			buildDLQConsumerMessage(t, "ATXid_123"),
		})

		if err := handler.Setup(session); err != nil {
			return err
		}
		if err := handler.ConsumeClaim(session, claim); err != nil {
			return err
		}
		if err := handler.Cleanup(session); err != nil {
			return err
		}

		cancel()
		return context.Canceled
	}

	replayer, err := NewReplayerWithConsumerGroup(
		group,
		"txova.users.v1.dlq",
		func(ctx context.Context, env *envelope.Envelope, msg *Message) error {
			replayed = true
			return nil
		},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = replayer.ReplayWithFilter(ctx, func(msg *Message) bool {
		return msg.OriginalTopic == "txova.users.v1"
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if !replayed {
		t.Fatal("expected replay handler to be called")
	}

	stats := replayer.Stats()
	if stats.Consumed != 1 || stats.Replayed != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestReplayer_Close(t *testing.T) {
	t.Parallel()

	closed := false
	group := newMockReplayConsumerGroup()
	group.closeFunc = func() error {
		closed = true
		return nil
	}

	replayer, err := NewReplayerWithConsumerGroup(
		group,
		"txova.users.v1.dlq",
		func(ctx context.Context, env *envelope.Envelope, msg *Message) error { return nil },
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := replayer.Close(); err != nil {
		t.Fatalf("unexpected error closing replayer: %v", err)
	}
	if !closed {
		t.Fatal("expected consumer group close to be called")
	}
	if err := replayer.Replay(context.Background()); !errors.Is(err, ErrReplayerClosed) {
		t.Fatalf("expected ErrReplayerClosed, got %v", err)
	}
}

func buildDLQConsumerMessage(t *testing.T, eventID string) *sarama.ConsumerMessage {
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
	env.ID = eventID

	payload, err := json.Marshal(&Message{
		OriginalTopic:     "txova.users.v1",
		OriginalPartition: 0,
		OriginalOffset:    100,
		ErrorMessage:      "processing failed",
		AttemptCount:      3,
		Envelope:          env,
	})
	if err != nil {
		t.Fatalf("failed to marshal DLQ message: %v", err)
	}

	return &sarama.ConsumerMessage{
		Topic:     "txova.users.v1.dlq",
		Partition: 0,
		Offset:    100,
		Value:     payload,
	}
}
