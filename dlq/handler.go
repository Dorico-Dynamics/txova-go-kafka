package dlq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/IBM/sarama"

	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
	"github.com/Dorico-Dynamics/txova-go-kafka/topics"
)

// Sentinel errors for DLQ operations.
var (
	// ErrNilEnvelope is returned when a nil envelope is passed to Handle.
	ErrNilEnvelope = errors.New("envelope cannot be nil")
)

// Handler handles failed messages by sending them to the dead letter queue.
type Handler struct {
	producer sarama.SyncProducer
	logger   *slog.Logger
}

// NewHandler creates a new DLQ handler with the given producer.
func NewHandler(producer sarama.SyncProducer, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		producer: producer,
		logger:   logger,
	}
}

// Handle sends a failed message to the appropriate DLQ topic.
func (h *Handler) Handle(
	ctx context.Context,
	originalTopic string,
	originalPartition int32,
	originalOffset int64,
	env *envelope.Envelope,
	processingErr error,
	attemptCount int,
) error {
	if env == nil {
		return ErrNilEnvelope
	}

	dlqMsg := NewMessage(
		originalTopic,
		originalPartition,
		originalOffset,
		env,
		processingErr,
		attemptCount,
	)

	data, err := json.Marshal(dlqMsg)
	if err != nil {
		return fmt.Errorf("failed to marshal DLQ message: %w", err)
	}

	dlqTopic := topics.Topic(originalTopic).DLQ()

	// Safely extract error message
	errorMessage := ""
	if processingErr != nil {
		errorMessage = processingErr.Error()
	}

	msg := &sarama.ProducerMessage{
		Topic: dlqTopic.String(),
		Key:   sarama.StringEncoder(env.ID),
		Value: sarama.ByteEncoder(data),
		Headers: []sarama.RecordHeader{
			{Key: []byte("original_topic"), Value: []byte(originalTopic)},
			{Key: []byte("event_type"), Value: []byte(env.Type)},
			{Key: []byte("event_id"), Value: []byte(env.ID)},
			{Key: []byte("error_message"), Value: []byte(errorMessage)},
		},
	}

	partition, offset, err := h.producer.SendMessage(msg)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to send message to DLQ",
			slog.String("dlq_topic", dlqTopic.String()),
			slog.String("event_type", env.Type),
			slog.String("event_id", env.ID),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("failed to send to DLQ: %w", err)
	}

	h.logger.InfoContext(ctx, "sent message to DLQ",
		slog.String("dlq_topic", dlqTopic.String()),
		slog.Int("partition", int(partition)),
		slog.Int64("offset", offset),
		slog.String("event_type", env.Type),
		slog.String("event_id", env.ID),
		slog.String("original_topic", originalTopic),
		slog.Int("attempt_count", attemptCount),
	)

	return nil
}

// CreateConsumerDLQHandler creates a DLQ handler function compatible with
// the consumer.Consumer.DLQHandler callback.
func (h *Handler) CreateConsumerDLQHandler(
	originalTopic string,
	originalPartition int32,
	originalOffset int64,
	attemptCount int,
) func(ctx context.Context, env *envelope.Envelope, err error) {
	return func(ctx context.Context, env *envelope.Envelope, err error) {
		if sendErr := h.Handle(ctx, originalTopic, originalPartition, originalOffset, env, err, attemptCount); sendErr != nil {
			h.logger.ErrorContext(ctx, "failed to handle DLQ",
				slog.String("event_id", env.ID),
				slog.String("error", sendErr.Error()),
			)
		}
	}
}
