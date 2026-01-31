package producer

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if cfg.RequiredAcks != sarama.WaitForAll {
		t.Errorf("RequiredAcks = %v, want %v", cfg.RequiredAcks, sarama.WaitForAll)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}
	if cfg.RetryBackoff != 100*time.Millisecond {
		t.Errorf("RetryBackoff = %v, want 100ms", cfg.RetryBackoff)
	}
	if cfg.BatchSize != 16*1024 {
		t.Errorf("BatchSize = %d, want 16384", cfg.BatchSize)
	}
	if cfg.Linger != 5*time.Millisecond {
		t.Errorf("Linger = %v, want 5ms", cfg.Linger)
	}
	if cfg.Compression != sarama.CompressionSnappy {
		t.Errorf("Compression = %v, want %v", cfg.Compression, sarama.CompressionSnappy)
	}
	if !cfg.Idempotent {
		t.Error("Idempotent should be true by default")
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v, want 10s", cfg.Timeout)
	}
	if cfg.Version != sarama.V0_11_0_0 {
		t.Errorf("Version = %v, want %v", cfg.Version, sarama.V0_11_0_0)
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name: "valid config",
			config: &Config{
				Brokers:  []string{"localhost:9092"},
				ClientID: "test-client",
			},
			wantErr: nil,
		},
		{
			name: "missing brokers",
			config: &Config{
				Brokers:  nil,
				ClientID: "test-client",
			},
			wantErr: ErrNoBrokers,
		},
		{
			name: "empty brokers",
			config: &Config{
				Brokers:  []string{},
				ClientID: "test-client",
			},
			wantErr: ErrNoBrokers,
		},
		{
			name: "missing client ID",
			config: &Config{
				Brokers:  []string{"localhost:9092"},
				ClientID: "",
			},
			wantErr: ErrNoClientID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.config.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_toSaramaConfig(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Brokers:      []string{"localhost:9092"},
		ClientID:     "test-client",
		RequiredAcks: sarama.WaitForAll,
		MaxRetries:   5,
		RetryBackoff: 200 * time.Millisecond,
		BatchSize:    32 * 1024,
		Linger:       10 * time.Millisecond,
		Compression:  sarama.CompressionGZIP,
		Idempotent:   true,
		Timeout:      15 * time.Second,
		Version:      sarama.V2_8_0_0,
	}

	saramaCfg, err := cfg.toSaramaConfig()
	if err != nil {
		t.Fatalf("toSaramaConfig() error = %v", err)
	}

	if saramaCfg.ClientID != cfg.ClientID {
		t.Errorf("ClientID = %q, want %q", saramaCfg.ClientID, cfg.ClientID)
	}
	if saramaCfg.Version != cfg.Version {
		t.Errorf("Version = %v, want %v", saramaCfg.Version, cfg.Version)
	}
	if saramaCfg.Producer.RequiredAcks != cfg.RequiredAcks {
		t.Errorf("RequiredAcks = %v, want %v", saramaCfg.Producer.RequiredAcks, cfg.RequiredAcks)
	}
	if saramaCfg.Producer.Retry.Max != cfg.MaxRetries {
		t.Errorf("Retry.Max = %d, want %d", saramaCfg.Producer.Retry.Max, cfg.MaxRetries)
	}
	if saramaCfg.Producer.Retry.Backoff != cfg.RetryBackoff {
		t.Errorf("Retry.Backoff = %v, want %v", saramaCfg.Producer.Retry.Backoff, cfg.RetryBackoff)
	}
	if saramaCfg.Producer.Flush.Bytes != cfg.BatchSize {
		t.Errorf("Flush.Bytes = %d, want %d", saramaCfg.Producer.Flush.Bytes, cfg.BatchSize)
	}
	if saramaCfg.Producer.Flush.Frequency != cfg.Linger {
		t.Errorf("Flush.Frequency = %v, want %v", saramaCfg.Producer.Flush.Frequency, cfg.Linger)
	}
	if saramaCfg.Producer.Compression != cfg.Compression {
		t.Errorf("Compression = %v, want %v", saramaCfg.Producer.Compression, cfg.Compression)
	}
	if saramaCfg.Producer.Idempotent != cfg.Idempotent {
		t.Errorf("Idempotent = %v, want %v", saramaCfg.Producer.Idempotent, cfg.Idempotent)
	}
	if saramaCfg.Producer.Timeout != cfg.Timeout {
		t.Errorf("Timeout = %v, want %v", saramaCfg.Producer.Timeout, cfg.Timeout)
	}
	if !saramaCfg.Producer.Return.Successes {
		t.Error("Return.Successes should be true")
	}
	if !saramaCfg.Producer.Return.Errors {
		t.Error("Return.Errors should be true")
	}
	// Idempotent requires MaxOpenRequests = 1
	if saramaCfg.Net.MaxOpenRequests != 1 {
		t.Errorf("MaxOpenRequests = %d, want 1 for idempotent producer", saramaCfg.Net.MaxOpenRequests)
	}
}

func TestConfig_toSaramaConfig_NonIdempotent(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Brokers:      []string{"localhost:9092"},
		ClientID:     "test-client",
		Idempotent:   false,
		RequiredAcks: sarama.WaitForLocal,
		Timeout:      DefaultTimeout,
		RetryBackoff: DefaultRetryBackoff,
	}

	saramaCfg, err := cfg.toSaramaConfig()
	if err != nil {
		t.Fatalf("toSaramaConfig() error = %v", err)
	}

	// Non-idempotent should not force MaxOpenRequests = 1
	if saramaCfg.Net.MaxOpenRequests == 1 {
		t.Error("MaxOpenRequests should not be 1 for non-idempotent producer")
	}
}
