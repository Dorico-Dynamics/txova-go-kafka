package producer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/IBM/sarama"

	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
	"github.com/Dorico-Dynamics/txova-go-kafka/events"
	"github.com/Dorico-Dynamics/txova-go-kafka/topics"
)

// Producer is a Kafka producer for publishing events.
type Producer struct {
	syncProducer sarama.SyncProducer
	config       *Config
	logger       *slog.Logger
	closed       bool
	mu           sync.RWMutex
}

// New creates a new Producer with the given configuration.
//
//nolint:unparam // Producer is returned and used by callers.
func New(cfg *Config, logger *slog.Logger) (*Producer, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if logger == nil {
		logger = slog.Default()
	}

	saramaCfg := cfg.toSaramaConfig()

	syncProducer, err := sarama.NewSyncProducer(cfg.Brokers, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create sync producer: %w", err)
	}

	return &Producer{
		syncProducer: syncProducer,
		config:       cfg,
		logger:       logger,
	}, nil
}

// Publish publishes an envelope to the appropriate topic based on event type.
func (p *Producer) Publish(ctx context.Context, env *envelope.Envelope, partitionKey string) error {
	topic := topics.ForEventType(events.EventType(env.Type))
	if topic == "" {
		return fmt.Errorf("%w: %s", ErrNoTopicForEventType, env.Type)
	}

	return p.PublishToTopic(ctx, topic, env, partitionKey)
}

// PublishToTopic publishes an envelope to a specific topic.
func (p *Producer) PublishToTopic(ctx context.Context, topic topics.Topic, env *envelope.Envelope, partitionKey string) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrProducerClosed
	}
	p.mu.RUnlock()

	if env == nil {
		return ErrNilEnvelope
	}

	if err := env.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidEnvelope, err.Error())
	}

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("failed to marshal envelope: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: topic.String(),
		Key:   sarama.StringEncoder(partitionKey),
		Value: sarama.ByteEncoder(data),
		Headers: []sarama.RecordHeader{
			{Key: []byte("event_type"), Value: []byte(env.Type)},
			{Key: []byte("event_id"), Value: []byte(env.ID)},
			{Key: []byte("correlation_id"), Value: []byte(env.CorrelationID)},
		},
	}

	partition, offset, err := p.syncProducer.SendMessage(msg)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to publish message",
			slog.String("topic", topic.String()),
			slog.String("event_type", env.Type),
			slog.String("event_id", env.ID),
			slog.String("correlation_id", env.CorrelationID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to publish message: %w", err)
	}

	p.logger.DebugContext(ctx, "published message",
		slog.String("topic", topic.String()),
		slog.Int("partition", int(partition)),
		slog.Int64("offset", offset),
		slog.String("event_type", env.Type),
		slog.String("event_id", env.ID),
		slog.String("correlation_id", env.CorrelationID),
	)

	return nil
}

// PublishBatch publishes multiple envelopes to their appropriate topics.
func (p *Producer) PublishBatch(ctx context.Context, envs []*envelope.Envelope, partitionKeys []string) error {
	if len(envs) != len(partitionKeys) {
		return ErrBatchLengthMismatch
	}

	for i, env := range envs {
		if err := p.Publish(ctx, env, partitionKeys[i]); err != nil {
			return fmt.Errorf("failed to publish envelope %d: %w", i, err)
		}
	}

	return nil
}

// Close closes the producer.
func (p *Producer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true

	if err := p.syncProducer.Close(); err != nil {
		return fmt.Errorf("failed to close producer: %w", err)
	}

	return nil
}
