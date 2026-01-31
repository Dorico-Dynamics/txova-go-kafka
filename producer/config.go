// Package producer provides Kafka producer implementations for publishing events.
package producer

import (
	"time"

	"github.com/IBM/sarama"
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
		MaxRetries:   3,
		RetryBackoff: 100 * time.Millisecond,
		BatchSize:    16 * 1024, // 16KB
		Linger:       5 * time.Millisecond,
		Compression:  sarama.CompressionSnappy,
		Idempotent:   true,
		Timeout:      10 * time.Second,
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
