package consumer

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
	"github.com/Dorico-Dynamics/txova-go-kafka/events"
)

// mockConsumerGroupSession implements sarama.ConsumerGroupSession for testing.
type mockConsumerGroupSession struct {
	ctx            context.Context
	markedMessages []*sarama.ConsumerMessage
	mu             sync.Mutex
}

func newMockConsumerGroupSession(ctx context.Context) *mockConsumerGroupSession {
	return &mockConsumerGroupSession{
		ctx:            ctx,
		markedMessages: make([]*sarama.ConsumerMessage, 0),
	}
}

func (m *mockConsumerGroupSession) Claims() map[string][]int32 {
	return map[string][]int32{"test-topic": {0}}
}

func (m *mockConsumerGroupSession) MemberID() string {
	return "test-member"
}

func (m *mockConsumerGroupSession) GenerationID() int32 {
	return 1
}

func (m *mockConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
}

func (m *mockConsumerGroupSession) Commit() {
}

func (m *mockConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
}

func (m *mockConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markedMessages = append(m.markedMessages, msg)
}

func (m *mockConsumerGroupSession) Context() context.Context {
	return m.ctx
}

func (m *mockConsumerGroupSession) MarkedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.markedMessages)
}

// mockConsumerGroupClaim implements sarama.ConsumerGroupClaim for testing.
type mockConsumerGroupClaim struct {
	messages chan *sarama.ConsumerMessage
	topic    string
}

func newMockConsumerGroupClaim(topic string, messages []*sarama.ConsumerMessage) *mockConsumerGroupClaim {
	ch := make(chan *sarama.ConsumerMessage, len(messages))
	for _, msg := range messages {
		ch <- msg
	}
	close(ch)
	return &mockConsumerGroupClaim{
		messages: ch,
		topic:    topic,
	}
}

func (m *mockConsumerGroupClaim) Topic() string {
	return m.topic
}

func (m *mockConsumerGroupClaim) Partition() int32 {
	return 0
}

func (m *mockConsumerGroupClaim) InitialOffset() int64 {
	return 0
}

func (m *mockConsumerGroupClaim) HighWaterMarkOffset() int64 {
	return 100
}

func (m *mockConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	return m.messages
}

// mockConsumerGroup implements sarama.ConsumerGroup for testing.
type mockConsumerGroup struct {
	consumeFunc func(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error
	closeFunc   func() error
	errChan     chan error
	closed      bool
	mu          sync.Mutex
}

func newMockConsumerGroup() *mockConsumerGroup {
	return &mockConsumerGroup{
		errChan: make(chan error),
	}
}

func (m *mockConsumerGroup) Consume(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
	if m.consumeFunc != nil {
		return m.consumeFunc(ctx, topics, handler)
	}
	return nil
}

func (m *mockConsumerGroup) Errors() <-chan error {
	return m.errChan
}

func (m *mockConsumerGroup) Close() error {
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

func (m *mockConsumerGroup) Pause(partitions map[string][]int32) {
}

func (m *mockConsumerGroup) Resume(partitions map[string][]int32) {
}

func (m *mockConsumerGroup) PauseAll() {
}

func (m *mockConsumerGroup) ResumeAll() {
}

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

func TestConsumerGroupHandler_Setup(t *testing.T) {
	t.Parallel()

	handler := &consumerGroupHandler{
		consumer: &Consumer{},
	}

	session := newMockConsumerGroupSession(context.Background())
	err := handler.Setup(session)
	if err != nil {
		t.Errorf("Setup() error = %v, want nil", err)
	}
}

func TestConsumerGroupHandler_Cleanup(t *testing.T) {
	t.Parallel()

	handler := &consumerGroupHandler{
		consumer: &Consumer{},
	}

	session := newMockConsumerGroupSession(context.Background())
	err := handler.Cleanup(session)
	if err != nil {
		t.Errorf("Cleanup() error = %v, want nil", err)
	}
}

func TestConsumerGroupHandler_ConsumeClaim(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()
	called := false
	registry.Register(events.EventTypeUserRegistered, func(ctx context.Context, env *envelope.Envelope) error {
		called = true
		return nil
	})

	consumer := &Consumer{
		config:   DefaultConfig(),
		registry: registry,
		logger:   slog.Default(),
	}

	handler := &consumerGroupHandler{
		consumer: consumer,
	}

	env := createTestEnvelope(t, string(events.EventTypeUserRegistered))
	envData, _ := json.Marshal(env)

	session := newMockConsumerGroupSession(context.Background())
	claim := newMockConsumerGroupClaim("test-topic", []*sarama.ConsumerMessage{
		{
			Topic:     "test-topic",
			Partition: 0,
			Offset:    1,
			Value:     envData,
		},
	})

	err := handler.ConsumeClaim(session, claim)
	if err != nil {
		t.Errorf("ConsumeClaim() error = %v, want nil", err)
	}
	if !called {
		t.Error("handler should have been called")
	}
	if session.MarkedCount() != 1 {
		t.Errorf("MarkedCount() = %d, want 1", session.MarkedCount())
	}
}

func TestConsumerGroupHandler_processMessage_InvalidJSON(t *testing.T) {
	t.Parallel()

	consumer := &Consumer{
		config:   DefaultConfig(),
		registry: NewHandlerRegistry(),
		logger:   slog.Default(),
	}

	handler := &consumerGroupHandler{
		consumer: consumer,
	}

	session := newMockConsumerGroupSession(context.Background())
	msg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    1,
		Value:     []byte("invalid json"),
	}

	handler.processMessage(session, msg)

	// Message should still be marked to avoid infinite loop
	if session.MarkedCount() != 1 {
		t.Errorf("MarkedCount() = %d, want 1 (invalid messages should be marked)", session.MarkedCount())
	}
}

func TestConsumerGroupHandler_processMessage_NoHandler(t *testing.T) {
	t.Parallel()

	consumer := &Consumer{
		config:   DefaultConfig(),
		registry: NewHandlerRegistry(), // Empty registry
		logger:   slog.Default(),
	}

	handler := &consumerGroupHandler{
		consumer: consumer,
	}

	env := createTestEnvelope(t, string(events.EventTypeUserRegistered))
	envData, _ := json.Marshal(env)

	session := newMockConsumerGroupSession(context.Background())
	msg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    1,
		Value:     envData,
	}

	handler.processMessage(session, msg)

	// Message should be marked even without handler
	if session.MarkedCount() != 1 {
		t.Errorf("MarkedCount() = %d, want 1", session.MarkedCount())
	}
}

func TestConsumerGroupHandler_processMessage_HandlerSuccess(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()
	called := false
	registry.Register(events.EventTypeUserRegistered, func(ctx context.Context, env *envelope.Envelope) error {
		called = true
		return nil
	})

	consumer := &Consumer{
		config:   DefaultConfig(),
		registry: registry,
		logger:   slog.Default(),
	}

	handler := &consumerGroupHandler{
		consumer: consumer,
	}

	env := createTestEnvelope(t, string(events.EventTypeUserRegistered))
	envData, _ := json.Marshal(env)

	session := newMockConsumerGroupSession(context.Background())
	msg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    1,
		Value:     envData,
	}

	handler.processMessage(session, msg)

	if !called {
		t.Error("handler should have been called")
	}
	if session.MarkedCount() != 1 {
		t.Errorf("MarkedCount() = %d, want 1", session.MarkedCount())
	}
}

func TestConsumerGroupHandler_processMessage_HandlerRetry(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()
	attempts := 0
	registry.Register(events.EventTypeUserRegistered, func(ctx context.Context, env *envelope.Envelope) error {
		attempts++
		if attempts < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	config := DefaultConfig()
	config.MaxRetries = 3

	consumer := &Consumer{
		config:   config,
		registry: registry,
		logger:   slog.Default(),
	}

	handler := &consumerGroupHandler{
		consumer: consumer,
	}

	env := createTestEnvelope(t, string(events.EventTypeUserRegistered))
	envData, _ := json.Marshal(env)

	session := newMockConsumerGroupSession(context.Background())
	msg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    1,
		Value:     envData,
	}

	handler.processMessage(session, msg)

	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	if session.MarkedCount() != 1 {
		t.Errorf("MarkedCount() = %d, want 1", session.MarkedCount())
	}
}

func TestConsumerGroupHandler_processMessage_HandlerFailsAllRetries(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()
	attempts := 0
	registry.Register(events.EventTypeUserRegistered, func(ctx context.Context, env *envelope.Envelope) error {
		attempts++
		return errors.New("permanent error")
	})

	config := DefaultConfig()
	config.MaxRetries = 3

	consumer := &Consumer{
		config:   config,
		registry: registry,
		logger:   slog.Default(),
	}

	handler := &consumerGroupHandler{
		consumer: consumer,
	}

	env := createTestEnvelope(t, string(events.EventTypeUserRegistered))
	envData, _ := json.Marshal(env)

	session := newMockConsumerGroupSession(context.Background())
	msg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    1,
		Value:     envData,
	}

	handler.processMessage(session, msg)

	if attempts != 3 {
		t.Errorf("attempts = %d, want 3 (max retries)", attempts)
	}
	// Message should still be marked after failure
	if session.MarkedCount() != 1 {
		t.Errorf("MarkedCount() = %d, want 1", session.MarkedCount())
	}
}

func TestConsumerGroupHandler_processMessage_DLQHandler(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()
	registry.Register(events.EventTypeUserRegistered, func(ctx context.Context, env *envelope.Envelope) error {
		return errors.New("permanent error")
	})

	config := DefaultConfig()
	config.MaxRetries = 1

	dlqCalled := false
	var dlqEnv *envelope.Envelope
	var dlqErr error

	consumer := &Consumer{
		config:   config,
		registry: registry,
		logger:   slog.Default(),
		DLQHandler: func(ctx context.Context, env *envelope.Envelope, err error) {
			dlqCalled = true
			dlqEnv = env
			dlqErr = err
		},
	}

	handler := &consumerGroupHandler{
		consumer: consumer,
	}

	env := createTestEnvelope(t, string(events.EventTypeUserRegistered))
	envData, _ := json.Marshal(env)

	session := newMockConsumerGroupSession(context.Background())
	msg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    1,
		Value:     envData,
	}

	handler.processMessage(session, msg)

	if !dlqCalled {
		t.Error("DLQ handler should have been called")
	}
	if dlqEnv == nil {
		t.Error("DLQ envelope should not be nil")
	}
	if dlqErr == nil {
		t.Error("DLQ error should not be nil")
	}
	if session.MarkedCount() != 1 {
		t.Errorf("MarkedCount() = %d, want 1", session.MarkedCount())
	}
}

func TestConsumerGroupHandler_processMessage_HandlerPanic(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()
	registry.Register(events.EventTypeUserRegistered, func(ctx context.Context, env *envelope.Envelope) error {
		panic("test panic")
	})

	config := DefaultConfig()
	config.MaxRetries = 1

	dlqCalled := false

	consumer := &Consumer{
		config:   config,
		registry: registry,
		logger:   slog.Default(),
		DLQHandler: func(ctx context.Context, env *envelope.Envelope, err error) {
			dlqCalled = true
		},
	}

	handler := &consumerGroupHandler{
		consumer: consumer,
	}

	env := createTestEnvelope(t, string(events.EventTypeUserRegistered))
	envData, _ := json.Marshal(env)

	session := newMockConsumerGroupSession(context.Background())
	msg := &sarama.ConsumerMessage{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    1,
		Value:     envData,
	}

	// Should not panic
	handler.processMessage(session, msg)

	if !dlqCalled {
		t.Error("DLQ handler should have been called after panic")
	}
	if session.MarkedCount() != 1 {
		t.Errorf("MarkedCount() = %d, want 1", session.MarkedCount())
	}
}

func TestConsumer_StartContextCancel(t *testing.T) {
	t.Parallel()

	mockClient := newMockConsumerGroup()
	mockClient.consumeFunc = func(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
		<-ctx.Done()
		return ctx.Err()
	}

	consumer := &Consumer{
		client:   mockClient,
		config:   &Config{Topics: []string{"test-topic"}},
		registry: NewHandlerRegistry(),
		logger:   slog.Default(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := consumer.Start(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Start() error = %v, want context.DeadlineExceeded", err)
	}
}

func TestConsumer_Close(t *testing.T) {
	t.Parallel()

	mockClient := newMockConsumerGroup()
	closeCalled := false
	mockClient.closeFunc = func() error {
		closeCalled = true
		return nil
	}

	consumer := &Consumer{
		client:   mockClient,
		config:   &Config{Topics: []string{"test-topic"}},
		registry: NewHandlerRegistry(),
		logger:   slog.Default(),
	}

	err := consumer.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
	if !closeCalled {
		t.Error("client.Close() should have been called")
	}
}

func TestConsumer_CloseError(t *testing.T) {
	t.Parallel()

	mockClient := newMockConsumerGroup()
	expectedErr := errors.New("close error")
	mockClient.closeFunc = func() error {
		return expectedErr
	}

	consumer := &Consumer{
		client:   mockClient,
		config:   &Config{Topics: []string{"test-topic"}},
		registry: NewHandlerRegistry(),
		logger:   slog.Default(),
	}

	err := consumer.Close()
	if err == nil {
		t.Error("Close() should return error")
	}
}

func TestConsumer_StartWithConsumeError(t *testing.T) {
	t.Parallel()

	callCount := 0
	mockClient := newMockConsumerGroup()
	mockClient.consumeFunc = func(ctx context.Context, topics []string, handler sarama.ConsumerGroupHandler) error {
		callCount++
		if callCount >= 2 {
			// Return context error on second call to break the loop
			return context.Canceled
		}
		return errors.New("consume error")
	}

	consumer := &Consumer{
		client:   mockClient,
		config:   &Config{Topics: []string{"test-topic"}},
		registry: NewHandlerRegistry(),
		logger:   slog.Default(),
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after first error is logged
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := consumer.Start(ctx)
	// Should eventually return context error
	if err == nil {
		t.Error("Start() should return error when context is cancelled")
	}
}

func TestConsumer_MultipleMessages(t *testing.T) {
	t.Parallel()

	registry := NewHandlerRegistry()
	processedCount := 0
	var mu sync.Mutex
	registry.Register(events.EventTypeUserRegistered, func(ctx context.Context, env *envelope.Envelope) error {
		mu.Lock()
		processedCount++
		mu.Unlock()
		return nil
	})

	consumer := &Consumer{
		config:   DefaultConfig(),
		registry: registry,
		logger:   slog.Default(),
	}

	handler := &consumerGroupHandler{
		consumer: consumer,
	}

	// Create multiple messages
	messages := make([]*sarama.ConsumerMessage, 5)
	for i := 0; i < 5; i++ {
		env := createTestEnvelope(t, string(events.EventTypeUserRegistered))
		envData, _ := json.Marshal(env)
		messages[i] = &sarama.ConsumerMessage{
			Topic:     "test-topic",
			Partition: 0,
			Offset:    int64(i),
			Value:     envData,
		}
	}

	session := newMockConsumerGroupSession(context.Background())
	claim := newMockConsumerGroupClaim("test-topic", messages)

	err := handler.ConsumeClaim(session, claim)
	if err != nil {
		t.Errorf("ConsumeClaim() error = %v, want nil", err)
	}

	mu.Lock()
	if processedCount != 5 {
		t.Errorf("processedCount = %d, want 5", processedCount)
	}
	mu.Unlock()

	if session.MarkedCount() != 5 {
		t.Errorf("MarkedCount() = %d, want 5", session.MarkedCount())
	}
}
