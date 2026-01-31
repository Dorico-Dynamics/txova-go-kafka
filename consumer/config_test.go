package consumer

import (
	"testing"
	"time"

	"github.com/IBM/sarama"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if cfg.OffsetReset != OffsetResetEarliest {
		t.Errorf("OffsetReset = %v, want %v", cfg.OffsetReset, OffsetResetEarliest)
	}
	if cfg.EnableAutoCommit {
		t.Error("EnableAutoCommit should be false by default")
	}
	if cfg.MaxPollRecords != 100 {
		t.Errorf("MaxPollRecords = %d, want 100", cfg.MaxPollRecords)
	}
	if cfg.SessionTimeout != 30*time.Second {
		t.Errorf("SessionTimeout = %v, want 30s", cfg.SessionTimeout)
	}
	if cfg.HeartbeatInterval != 10*time.Second {
		t.Errorf("HeartbeatInterval = %v, want 10s", cfg.HeartbeatInterval)
	}
	if cfg.ProcessingTimeout != 30*time.Second {
		t.Errorf("ProcessingTimeout = %v, want 30s", cfg.ProcessingTimeout)
	}
	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
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
				Brokers: []string{"localhost:9092"},
				GroupID: "test-group",
				Topics:  []string{"test-topic"},
			},
			wantErr: nil,
		},
		{
			name: "missing brokers",
			config: &Config{
				GroupID: "test-group",
				Topics:  []string{"test-topic"},
			},
			wantErr: ErrNoBrokers,
		},
		{
			name: "empty brokers",
			config: &Config{
				Brokers: []string{},
				GroupID: "test-group",
				Topics:  []string{"test-topic"},
			},
			wantErr: ErrNoBrokers,
		},
		{
			name: "missing group ID",
			config: &Config{
				Brokers: []string{"localhost:9092"},
				GroupID: "",
				Topics:  []string{"test-topic"},
			},
			wantErr: ErrNoGroupID,
		},
		{
			name: "missing topics",
			config: &Config{
				Brokers: []string{"localhost:9092"},
				GroupID: "test-group",
				Topics:  nil,
			},
			wantErr: ErrNoTopics,
		},
		{
			name: "empty topics",
			config: &Config{
				Brokers: []string{"localhost:9092"},
				GroupID: "test-group",
				Topics:  []string{},
			},
			wantErr: ErrNoTopics,
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

func TestConfig_toSaramaConfig_Earliest(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Brokers:           []string{"localhost:9092"},
		GroupID:           "test-group",
		Topics:            []string{"test-topic"},
		OffsetReset:       OffsetResetEarliest,
		EnableAutoCommit:  false,
		SessionTimeout:    30 * time.Second,
		HeartbeatInterval: 10 * time.Second,
		ProcessingTimeout: 30 * time.Second,
	}

	saramaCfg := cfg.toSaramaConfig()

	if saramaCfg.Consumer.Offsets.Initial != sarama.OffsetOldest {
		t.Errorf("Offsets.Initial = %v, want OffsetOldest", saramaCfg.Consumer.Offsets.Initial)
	}
	if saramaCfg.Consumer.Offsets.AutoCommit.Enable != cfg.EnableAutoCommit {
		t.Errorf("AutoCommit.Enable = %v, want %v", saramaCfg.Consumer.Offsets.AutoCommit.Enable, cfg.EnableAutoCommit)
	}
	if saramaCfg.Consumer.Group.Session.Timeout != cfg.SessionTimeout {
		t.Errorf("Session.Timeout = %v, want %v", saramaCfg.Consumer.Group.Session.Timeout, cfg.SessionTimeout)
	}
	if saramaCfg.Consumer.Group.Heartbeat.Interval != cfg.HeartbeatInterval {
		t.Errorf("Heartbeat.Interval = %v, want %v", saramaCfg.Consumer.Group.Heartbeat.Interval, cfg.HeartbeatInterval)
	}
	if !saramaCfg.Consumer.Return.Errors {
		t.Error("Return.Errors should be true")
	}
}

func TestConfig_toSaramaConfig_Latest(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		OffsetReset: OffsetResetLatest,
	}

	saramaCfg := cfg.toSaramaConfig()

	if saramaCfg.Consumer.Offsets.Initial != sarama.OffsetNewest {
		t.Errorf("Offsets.Initial = %v, want OffsetNewest", saramaCfg.Consumer.Offsets.Initial)
	}
}
