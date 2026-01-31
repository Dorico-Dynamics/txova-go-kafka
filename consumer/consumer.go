package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/IBM/sarama"

	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
	"github.com/Dorico-Dynamics/txova-go-kafka/events"
)

// Consumer is a Kafka consumer for consuming events.
type Consumer struct {
	client   sarama.ConsumerGroup
	config   *Config
	registry *HandlerRegistry
	logger   *slog.Logger
	closed   bool
	mu       sync.RWMutex
	wg       sync.WaitGroup

	// DLQHandler is called when a message fails processing after max retries.
	// If nil, failed messages are logged and discarded.
	DLQHandler func(ctx context.Context, env *envelope.Envelope, err error)
}

// New creates a new Consumer with the given configuration.
//
//nolint:unparam // Consumer is returned and used by callers.
func New(cfg *Config, registry *HandlerRegistry, logger *slog.Logger) (*Consumer, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if registry == nil {
		registry = NewHandlerRegistry()
	}

	if logger == nil {
		logger = slog.Default()
	}

	saramaCfg := cfg.toSaramaConfig()

	client, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, saramaCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	return &Consumer{
		client:   client,
		config:   cfg,
		registry: registry,
		logger:   logger,
	}, nil
}

// Start begins consuming messages. This method blocks until the context is
// cancelled or Close is called.
func (c *Consumer) Start(ctx context.Context) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrConsumerClosed
	}
	c.mu.RUnlock()

	handler := &consumerGroupHandler{
		consumer: c,
	}

	c.wg.Add(1)
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx) //nolint:wrapcheck // Context errors are terminal and should not be wrapped.
		default:
			err := c.client.Consume(ctx, c.config.Topics, handler)
			if err != nil {
				if ctx.Err() != nil {
					return context.Cause(ctx) //nolint:wrapcheck // Context errors are terminal and should not be wrapped.
				}
				c.logger.ErrorContext(ctx, "error consuming",
					slog.String("error", err.Error()),
				)
			}
		}
	}
}

// Close closes the consumer and waits for all handlers to complete.
func (c *Consumer) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.mu.Unlock()

	// Wait for consume loop to exit
	c.wg.Wait()

	if err := c.client.Close(); err != nil {
		return fmt.Errorf("failed to close consumer: %w", err)
	}

	return nil
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler.
type consumerGroupHandler struct {
	consumer *Consumer
}

func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		h.processMessage(session, msg)
	}
	return nil
}

func (h *consumerGroupHandler) processMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	ctx := session.Context()

	var env envelope.Envelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		h.consumer.logger.ErrorContext(ctx, "failed to unmarshal envelope",
			slog.String("topic", msg.Topic),
			slog.Int("partition", int(msg.Partition)),
			slog.Int64("offset", msg.Offset),
			slog.String("error", err.Error()),
		)
		// Mark as processed to avoid infinite loop on bad messages
		session.MarkMessage(msg, "")
		return
	}

	eventType := events.EventType(env.Type)
	handler := h.consumer.registry.Get(eventType)
	if handler == nil {
		h.consumer.logger.WarnContext(ctx, "no handler for event type",
			slog.String("event_type", env.Type),
			slog.String("event_id", env.ID),
		)
		session.MarkMessage(msg, "")
		return
	}

	// Process with retries
	var lastErr error
	for attempt := 1; attempt <= h.consumer.config.MaxRetries; attempt++ {
		lastErr = h.invokeHandler(ctx, handler, &env)
		if lastErr == nil {
			break
		}

		h.consumer.logger.WarnContext(ctx, "handler failed, retrying",
			slog.String("event_type", env.Type),
			slog.String("event_id", env.ID),
			slog.Int("attempt", attempt),
			slog.Int("max_retries", h.consumer.config.MaxRetries),
			slog.String("error", lastErr.Error()),
		)
	}

	if lastErr != nil {
		h.consumer.logger.ErrorContext(ctx, "handler failed after max retries",
			slog.String("event_type", env.Type),
			slog.String("event_id", env.ID),
			slog.String("error", lastErr.Error()),
		)

		// Send to DLQ if configured
		if h.consumer.DLQHandler != nil {
			h.consumer.DLQHandler(ctx, &env, lastErr)
		}
	}

	// Mark message as processed
	session.MarkMessage(msg, "")
}

func (h *consumerGroupHandler) invokeHandler(ctx context.Context, handler Handler, env *envelope.Envelope) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrHandlerPanic, r)
		}
	}()

	return handler(ctx, env)
}
