// Package producer provides Kafka producer implementations for publishing events.
package producer

import (
	"time"

	"github.com/IBM/sarama"
)

// Default configuration values.
const (
	// DefaultMaxRetries is the default number of retry attempts for failed messages.
	DefaultMaxRetries = 3
	// DefaultRetryBackoff is the default time to wait between retries.
	DefaultRetryBackoff = 100 * time.Millisecond
	// DefaultBatchSize is the default maximum size of a batch in bytes (16KB).
	DefaultBatchSize = 16 * 1024
	// DefaultLinger is the default time to wait before sending a batch.
	DefaultLinger = 5 * time.Millisecond
	// DefaultTimeout is the default timeout for producing messages.
	DefaultTimeout = 10 * time.Second
)

// Config holds the configuration for a Kafka producer.
type Config struct {
	// Brokers is the list of Kafka broker addresses.
	Brokers []string
	// ClientID is the client identifier sent to Kafka.
	ClientID string
	// RequiredAcks specifies the level of acknowledgment required.
	// Default: WaitForAll (wait for all replicas).
	RequiredAcks sarama.RequiredAcks
	// MaxRetries is the number of retry attempts for failed messages.
	// Default: 3.
	MaxRetries int
	// RetryBackoff is the time to wait between retries.
	// Default: 100ms.
	RetryBackoff time.Duration
	// BatchSize is the maximum size of a batch in bytes.
	// Default: 16KB.
	BatchSize int
	// Linger is the time to wait before sending a batch.
	// Default: 5ms.
	Linger time.Duration
	// Compression is the compression codec to use.
	// Default: Snappy.
	Compression sarama.CompressionCodec
	// Idempotent enables idempotent producer mode.
	// Default: true.
	Idempotent bool
	// Timeout is the timeout for producing messages.
	// Default: 10s.
	Timeout time.Duration
}

// DefaultConfig returns a Config with sensible defaults for production.
func DefaultConfig() *Config {
	return &Config{
		RequiredAcks: sarama.WaitForAll,
		MaxRetries:   DefaultMaxRetries,
		RetryBackoff: DefaultRetryBackoff,
		BatchSize:    DefaultBatchSize,
		Linger:       DefaultLinger,
		Compression:  sarama.CompressionSnappy,
		Idempotent:   true,
		Timeout:      DefaultTimeout,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if len(c.Brokers) == 0 {
		return ErrNoBrokers
	}
	if c.ClientID == "" {
		return ErrNoClientID
	}
	return nil
}

// toSaramaConfig converts the Config to a sarama.Config.
func (c *Config) toSaramaConfig() *sarama.Config {
	cfg := sarama.NewConfig()

	cfg.ClientID = c.ClientID
	cfg.Producer.RequiredAcks = c.RequiredAcks
	cfg.Producer.Retry.Max = c.MaxRetries
	cfg.Producer.Retry.Backoff = c.RetryBackoff
	cfg.Producer.Flush.Bytes = c.BatchSize
	cfg.Producer.Flush.Frequency = c.Linger
	cfg.Producer.Compression = c.Compression
	cfg.Producer.Idempotent = c.Idempotent
	cfg.Producer.Timeout = c.Timeout
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true

	// Required for idempotent producer
	if c.Idempotent {
		cfg.Net.MaxOpenRequests = 1
	}

	return cfg
}
