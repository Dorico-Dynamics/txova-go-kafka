// Package consumer provides Kafka consumer implementations for consuming events.
package consumer

import (
	"time"

	"github.com/IBM/sarama"
)

// OffsetReset specifies where to start consuming when no offset is found.
type OffsetReset string

const (
	// OffsetResetEarliest starts from the beginning of the partition.
	OffsetResetEarliest OffsetReset = "earliest"
	// OffsetResetLatest starts from the most recent message.
	OffsetResetLatest OffsetReset = "latest"
)

// Config holds the configuration for a Kafka consumer.
type Config struct {
	// Brokers is the list of Kafka broker addresses.
	Brokers []string
	// GroupID is the consumer group identifier.
	GroupID string
	// Topics is the list of topics to consume from.
	Topics []string
	// OffsetReset specifies where to start consuming when no offset is found.
	// Default: earliest.
	OffsetReset OffsetReset
	// EnableAutoCommit enables automatic offset commits.
	// Default: false (manual commit recommended).
	EnableAutoCommit bool
	// MaxPollRecords is the maximum number of records to fetch per poll.
	// Default: 100.
	MaxPollRecords int
	// SessionTimeout is the timeout for consumer session.
	// Default: 30s.
	SessionTimeout time.Duration
	// HeartbeatInterval is the expected time between heartbeats.
	// Default: 10s.
	HeartbeatInterval time.Duration
	// ProcessingTimeout is the timeout for processing a single message.
	// Default: 30s.
	ProcessingTimeout time.Duration
	// MaxRetries is the number of retry attempts before sending to DLQ.
	// Default: 3.
	MaxRetries int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		OffsetReset:       OffsetResetEarliest,
		EnableAutoCommit:  false,
		MaxPollRecords:    100,
		SessionTimeout:    30 * time.Second,
		HeartbeatInterval: 10 * time.Second,
		ProcessingTimeout: 30 * time.Second,
		MaxRetries:        3,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if len(c.Brokers) == 0 {
		return ErrNoBrokers
	}
	if c.GroupID == "" {
		return ErrNoGroupID
	}
	if len(c.Topics) == 0 {
		return ErrNoTopics
	}
	return nil
}

// toSaramaConfig converts the Config to a sarama.Config.
func (c *Config) toSaramaConfig() *sarama.Config {
	cfg := sarama.NewConfig()

	cfg.Consumer.Group.Session.Timeout = c.SessionTimeout
	cfg.Consumer.Group.Heartbeat.Interval = c.HeartbeatInterval
	cfg.Consumer.MaxProcessingTime = c.ProcessingTimeout

	if c.OffsetReset == OffsetResetLatest {
		cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	} else {
		cfg.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	cfg.Consumer.Offsets.AutoCommit.Enable = c.EnableAutoCommit
	cfg.Consumer.Return.Errors = true

	return cfg
}
