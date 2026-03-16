package dlq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/IBM/sarama"

	"github.com/Dorico-Dynamics/txova-go-kafka/consumer"
	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
)

var (
	// ErrNilReplayHandler is returned when a replayer is created without a handler.
	ErrNilReplayHandler = errors.New("replay handler cannot be nil")
	// ErrNilConsumerGroup is returned when a replayer is created without a consumer group.
	ErrNilConsumerGroup = errors.New("consumer group is required")
	// ErrEmptyDLQTopic is returned when a replayer is created without a DLQ topic.
	ErrEmptyDLQTopic = errors.New("dlq topic is required")
	// ErrReplayerClosed is returned when replay is attempted after close.
	ErrReplayerClosed = errors.New("replayer is closed")
)

// ReplayHandler replays the original event envelope from a DLQ message.
type ReplayHandler func(ctx context.Context, env *envelope.Envelope, msg *Message) error

// ReplayFilter decides whether a DLQ message should be replayed.
type ReplayFilter func(msg *Message) bool

// ReplayStats tracks replayer outcomes.
type ReplayStats struct {
	Consumed int64
	Replayed int64
	Skipped  int64
	Failed   int64
}

// Replayer consumes DLQ messages and replays their original envelopes through
// a caller-provided handler.
type Replayer struct {
	consumerGroup sarama.ConsumerGroup
	topic         string
	handler       ReplayHandler
	logger        *slog.Logger

	mu     sync.RWMutex
	wg     sync.WaitGroup
	closed bool
	stats  ReplayStats
}

// NewReplayer creates a DLQ replayer backed by a Kafka consumer group.
func NewReplayer(cfg *consumer.Config, dlqTopic string, handler ReplayHandler, logger *slog.Logger) (*Replayer, error) {
	if cfg == nil {
		cfg = consumer.DefaultConfig()
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	if dlqTopic == "" {
		return nil, ErrEmptyDLQTopic
	}
	if handler == nil {
		return nil, ErrNilReplayHandler
	}

	group, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, toSaramaConfig(cfg))
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	return NewReplayerWithConsumerGroup(group, dlqTopic, handler, logger)
}

func toSaramaConfig(cfg *consumer.Config) *sarama.Config {
	saramaCfg := sarama.NewConfig()

	saramaCfg.Consumer.Group.Session.Timeout = cfg.SessionTimeout
	saramaCfg.Consumer.Group.Heartbeat.Interval = cfg.HeartbeatInterval
	saramaCfg.Consumer.MaxProcessingTime = cfg.ProcessingTimeout

	if cfg.OffsetReset == consumer.OffsetResetLatest {
		saramaCfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	} else {
		saramaCfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	saramaCfg.Consumer.Offsets.AutoCommit.Enable = cfg.EnableAutoCommit
	saramaCfg.Consumer.Return.Errors = true

	return saramaCfg
}

// NewReplayerWithConsumerGroup creates a replayer with an injected consumer
// group. This is primarily useful for tests.
func NewReplayerWithConsumerGroup(
	group sarama.ConsumerGroup,
	dlqTopic string,
	handler ReplayHandler,
	logger *slog.Logger,
) (*Replayer, error) {
	if group == nil {
		return nil, ErrNilConsumerGroup
	}
	if dlqTopic == "" {
		return nil, ErrEmptyDLQTopic
	}
	if handler == nil {
		return nil, ErrNilReplayHandler
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Replayer{
		consumerGroup: group,
		topic:         dlqTopic,
		handler:       handler,
		logger:        logger,
	}, nil
}

// Replay replays every message from the configured DLQ topic until the context
// is cancelled or the replayer is closed.
func (r *Replayer) Replay(ctx context.Context) error {
	return r.ReplayWithFilter(ctx, nil)
}

// ReplayWithFilter replays only messages that match the provided filter.
func (r *Replayer) ReplayWithFilter(ctx context.Context, filter ReplayFilter) error {
	r.mu.RLock()
	if r.closed {
		r.mu.RUnlock()
		return ErrReplayerClosed
	}
	r.wg.Add(1)
	r.mu.RUnlock()
	defer r.wg.Done()

	handler := &replayerGroupHandler{
		replayer: r,
		filter:   filter,
	}

	for {
		if err := ctx.Err(); err != nil {
			return context.Cause(ctx) //nolint:wrapcheck // terminal context error
		}

		err := r.consumerGroup.Consume(ctx, []string{r.topic}, handler)
		if err != nil {
			if ctx.Err() != nil {
				return context.Cause(ctx) //nolint:wrapcheck // terminal context error
			}
			if r.isClosed() {
				return nil
			}
			return fmt.Errorf("failed to replay dlq topic %s: %w", r.topic, err)
		}

		if r.isClosed() {
			return nil
		}
	}
}

// Stats returns a point-in-time snapshot of replay statistics.
func (r *Replayer) Stats() ReplayStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.stats
}

// Close closes the underlying consumer group and waits for active replay loops
// to exit.
func (r *Replayer) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.mu.Unlock()

	err := r.consumerGroup.Close()
	r.wg.Wait()
	if err != nil {
		return fmt.Errorf("failed to close replayer: %w", err)
	}

	return nil
}

func (r *Replayer) isClosed() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.closed
}

func (r *Replayer) incrementConsumed() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stats.Consumed++
}

func (r *Replayer) incrementReplayed() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stats.Replayed++
}

func (r *Replayer) incrementSkipped() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stats.Skipped++
}

func (r *Replayer) incrementFailed() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stats.Failed++
}

type replayerGroupHandler struct {
	replayer *Replayer
	filter   ReplayFilter
}

func (h *replayerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *replayerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (h *replayerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		if err := h.processMessage(session, msg); err != nil {
			return err
		}
	}

	return nil
}

func (h *replayerGroupHandler) processMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) error {
	ctx := session.Context()
	h.replayer.incrementConsumed()

	var dlqMsg Message
	if err := json.Unmarshal(msg.Value, &dlqMsg); err != nil {
		h.replayer.logger.ErrorContext(ctx, "failed to decode DLQ message",
			slog.String("topic", msg.Topic),
			slog.Int("partition", int(msg.Partition)),
			slog.Int64("offset", msg.Offset),
			slog.String("error", err.Error()),
		)
		h.replayer.incrementSkipped()
		session.MarkMessage(msg, "")

		return nil
	}

	if dlqMsg.Envelope == nil {
		h.replayer.logger.WarnContext(ctx, "skipping DLQ message without envelope",
			slog.String("topic", msg.Topic),
			slog.Int("partition", int(msg.Partition)),
			slog.Int64("offset", msg.Offset),
		)
		h.replayer.incrementSkipped()
		session.MarkMessage(msg, "")

		return nil
	}

	if h.filter != nil && !h.filter(&dlqMsg) {
		h.replayer.incrementSkipped()
		session.MarkMessage(msg, "")

		return nil
	}

	if err := h.replayer.handler(ctx, dlqMsg.Envelope, &dlqMsg); err != nil {
		h.replayer.logger.ErrorContext(ctx, "failed to replay DLQ message",
			slog.String("topic", msg.Topic),
			slog.String("event_id", dlqMsg.Envelope.ID),
			slog.String("event_type", dlqMsg.Envelope.Type),
			slog.String("error", err.Error()),
		)
		h.replayer.incrementFailed()

		return fmt.Errorf("replay handler failed for %s/%d/%d: %w", msg.Topic, msg.Partition, msg.Offset, err)
	}

	h.replayer.incrementReplayed()
	session.MarkMessage(msg, "")

	return nil
}
