package envelope

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	txctx "github.com/Dorico-Dynamics/txova-go-core/context"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
)

// testPayload is a sample payload for testing.
type testPayload struct {
	RideID   string `json:"ride_id"`
	DriverID string `json:"driver_id"`
	Fare     int64  `json:"fare"`
}

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: &Config{
				Type:          "ride.completed",
				Version:       "1.0",
				Source:        "ride-service",
				CorrelationID: "corr-123",
				Payload:       testPayload{RideID: "ride-1", DriverID: "driver-1", Fare: 1500},
			},
			wantErr: false,
		},
		{
			name: "valid config with actor",
			cfg: &Config{
				Type:          "user.registered",
				Version:       "2.1",
				Source:        "user-service",
				CorrelationID: "corr-456",
				Actor: &Actor{
					ID:   ids.MustNewUserID(),
					Type: ActorTypeUser,
				},
				Payload: map[string]string{"user_id": "user-1"},
			},
			wantErr: false,
		},
		{
			name: "valid config with causation ID",
			cfg: &Config{
				Type:          "payment.completed",
				Version:       "1.0",
				Source:        "payment-service",
				CorrelationID: "corr-789",
				CausationID:   "cause-event-123",
				Payload:       map[string]string{"payment_id": "pay-1"},
			},
			wantErr: false,
		},
		{
			name: "missing type",
			cfg: &Config{
				Version:       "1.0",
				Source:        "test-service",
				CorrelationID: "corr-123",
				Payload:       map[string]string{"key": "value"},
			},
			wantErr: true,
			errMsg:  "event type is required",
		},
		{
			name: "missing version",
			cfg: &Config{
				Type:          "test.event",
				Source:        "test-service",
				CorrelationID: "corr-123",
				Payload:       map[string]string{"key": "value"},
			},
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "invalid version format",
			cfg: &Config{
				Type:          "test.event",
				Version:       "v1.0.0",
				Source:        "test-service",
				CorrelationID: "corr-123",
				Payload:       map[string]string{"key": "value"},
			},
			wantErr: true,
			errMsg:  "version must be in semver format",
		},
		{
			name: "missing source",
			cfg: &Config{
				Type:          "test.event",
				Version:       "1.0",
				CorrelationID: "corr-123",
				Payload:       map[string]string{"key": "value"},
			},
			wantErr: true,
			errMsg:  "source is required",
		},
		{
			name: "missing payload",
			cfg: &Config{
				Type:          "test.event",
				Version:       "1.0",
				Source:        "test-service",
				CorrelationID: "corr-123",
			},
			wantErr: true,
			errMsg:  "payload is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env, err := New(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Errorf("New() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("New() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("New() unexpected error: %v", err)
				return
			}

			// Verify envelope fields
			if env.ID == "" {
				t.Error("New() envelope ID should not be empty")
			}
			if env.Type != tt.cfg.Type {
				t.Errorf("New() Type = %q, want %q", env.Type, tt.cfg.Type)
			}
			if env.Version != tt.cfg.Version {
				t.Errorf("New() Version = %q, want %q", env.Version, tt.cfg.Version)
			}
			if env.Source != tt.cfg.Source {
				t.Errorf("New() Source = %q, want %q", env.Source, tt.cfg.Source)
			}
			if env.CorrelationID != tt.cfg.CorrelationID {
				t.Errorf("New() CorrelationID = %q, want %q", env.CorrelationID, tt.cfg.CorrelationID)
			}
			if env.Timestamp.IsZero() {
				t.Error("New() Timestamp should not be zero")
			}
			if len(env.Payload) == 0 {
				t.Error("New() Payload should not be empty")
			}
		})
	}
}

func TestNewWithContext(t *testing.T) {
	t.Parallel()

	t.Run("extracts correlation ID from context", func(t *testing.T) {
		t.Parallel()

		ctx := txctx.WithCorrelationID(context.Background(), "ctx-corr-123")
		env, err := NewWithContext(ctx, &Config{
			Type:    "test.event",
			Version: "1.0",
			Source:  "test-service",
			Payload: map[string]string{"key": "value"},
		})
		if err != nil {
			t.Fatalf("NewWithContext() error: %v", err)
		}
		if env.CorrelationID != "ctx-corr-123" {
			t.Errorf("NewWithContext() CorrelationID = %q, want %q", env.CorrelationID, "ctx-corr-123")
		}
	})

	t.Run("uses config correlation ID over context", func(t *testing.T) {
		t.Parallel()

		ctx := txctx.WithCorrelationID(context.Background(), "ctx-corr-123")
		env, err := NewWithContext(ctx, &Config{
			Type:          "test.event",
			Version:       "1.0",
			Source:        "test-service",
			CorrelationID: "cfg-corr-456",
			Payload:       map[string]string{"key": "value"},
		})
		if err != nil {
			t.Fatalf("NewWithContext() error: %v", err)
		}
		if env.CorrelationID != "cfg-corr-456" {
			t.Errorf("NewWithContext() CorrelationID = %q, want %q", env.CorrelationID, "cfg-corr-456")
		}
	})

	t.Run("falls back to request ID", func(t *testing.T) {
		t.Parallel()

		ctx := txctx.WithRequestID(context.Background(), "req-123")
		env, err := NewWithContext(ctx, &Config{
			Type:    "test.event",
			Version: "1.0",
			Source:  "test-service",
			Payload: map[string]string{"key": "value"},
		})
		if err != nil {
			t.Fatalf("NewWithContext() error: %v", err)
		}
		if env.CorrelationID != "req-123" {
			t.Errorf("NewWithContext() CorrelationID = %q, want %q", env.CorrelationID, "req-123")
		}
	})
}

func TestEnvelope_Validate(t *testing.T) {
	t.Parallel()

	validEnvelope := &Envelope{
		ID:            "evt-123",
		Type:          "ride.completed",
		Version:       "1.0",
		Source:        "ride-service",
		Timestamp:     time.Now().UTC(),
		CorrelationID: "corr-123",
		Payload:       json.RawMessage(`{"ride_id": "ride-1"}`),
	}

	tests := []struct {
		name    string
		modify  func(e *Envelope) *Envelope
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid envelope",
			modify:  func(e *Envelope) *Envelope { return e },
			wantErr: false,
		},
		{
			name:    "missing ID",
			modify:  func(e *Envelope) *Envelope { e.ID = ""; return e },
			wantErr: true,
			errMsg:  "id is required",
		},
		{
			name:    "missing type",
			modify:  func(e *Envelope) *Envelope { e.Type = ""; return e },
			wantErr: true,
			errMsg:  "type is required",
		},
		{
			name:    "missing version",
			modify:  func(e *Envelope) *Envelope { e.Version = ""; return e },
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name:    "invalid version",
			modify:  func(e *Envelope) *Envelope { e.Version = "v1.0.0"; return e },
			wantErr: true,
			errMsg:  "version must be in semver format",
		},
		{
			name:    "missing source",
			modify:  func(e *Envelope) *Envelope { e.Source = ""; return e },
			wantErr: true,
			errMsg:  "source is required",
		},
		{
			name:    "missing timestamp",
			modify:  func(e *Envelope) *Envelope { e.Timestamp = time.Time{}; return e },
			wantErr: true,
			errMsg:  "timestamp is required",
		},
		{
			name:    "missing correlation ID",
			modify:  func(e *Envelope) *Envelope { e.CorrelationID = ""; return e },
			wantErr: true,
			errMsg:  "correlation_id is required",
		},
		{
			name:    "empty payload",
			modify:  func(e *Envelope) *Envelope { e.Payload = nil; return e },
			wantErr: true,
			errMsg:  "payload is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create a copy to avoid race conditions
			envCopy := *validEnvelope
			env := tt.modify(&envCopy)
			err := env.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !containsString(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}

func TestEnvelope_UnmarshalPayload(t *testing.T) {
	t.Parallel()

	t.Run("unmarshal valid payload", func(t *testing.T) {
		t.Parallel()

		env := Envelope{
			Payload: json.RawMessage(`{"ride_id": "ride-1", "driver_id": "driver-1", "fare": 1500}`),
		}

		var payload testPayload
		err := env.UnmarshalPayload(&payload)
		if err != nil {
			t.Fatalf("UnmarshalPayload() error: %v", err)
		}
		if payload.RideID != "ride-1" {
			t.Errorf("UnmarshalPayload() RideID = %q, want %q", payload.RideID, "ride-1")
		}
		if payload.DriverID != "driver-1" {
			t.Errorf("UnmarshalPayload() DriverID = %q, want %q", payload.DriverID, "driver-1")
		}
		if payload.Fare != 1500 {
			t.Errorf("UnmarshalPayload() Fare = %d, want %d", payload.Fare, 1500)
		}
	})

	t.Run("nil destination", func(t *testing.T) {
		t.Parallel()

		env := Envelope{
			Payload: json.RawMessage(`{"key": "value"}`),
		}

		err := env.UnmarshalPayload(nil)
		if err == nil {
			t.Error("UnmarshalPayload() expected error for nil destination")
		}
	})

	t.Run("empty payload", func(t *testing.T) {
		t.Parallel()

		env := Envelope{
			Payload: nil,
		}

		var payload testPayload
		err := env.UnmarshalPayload(&payload)
		if err == nil {
			t.Error("UnmarshalPayload() expected error for empty payload")
		}
	})

	t.Run("invalid JSON payload", func(t *testing.T) {
		t.Parallel()

		env := Envelope{
			Payload: json.RawMessage(`{invalid json}`),
		}

		var payload testPayload
		err := env.UnmarshalPayload(&payload)
		if err == nil {
			t.Error("UnmarshalPayload() expected error for invalid JSON")
		}
	})
}

func TestEnvelope_WithMethods(t *testing.T) {
	t.Parallel()

	env := &Envelope{
		ID:            "evt-123",
		Type:          "test.event",
		Version:       "1.0",
		Source:        "test-service",
		Timestamp:     time.Now().UTC(),
		CorrelationID: "original-corr",
		Payload:       json.RawMessage(`{}`),
	}

	t.Run("WithCorrelationID", func(t *testing.T) {
		t.Parallel()

		newEnv := env.WithCorrelationID("new-corr")
		if newEnv.CorrelationID != "new-corr" {
			t.Errorf("WithCorrelationID() CorrelationID = %q, want %q", newEnv.CorrelationID, "new-corr")
		}
		// Original should be unchanged
		if env.CorrelationID != "original-corr" {
			t.Errorf("WithCorrelationID() modified original envelope")
		}
	})

	t.Run("WithCausationID", func(t *testing.T) {
		t.Parallel()

		newEnv := env.WithCausationID("cause-123")
		if newEnv.CausationID != "cause-123" {
			t.Errorf("WithCausationID() CausationID = %q, want %q", newEnv.CausationID, "cause-123")
		}
	})

	t.Run("WithActor", func(t *testing.T) {
		t.Parallel()

		actor := &Actor{
			ID:   ids.MustNewUserID(),
			Type: ActorTypeAdmin,
		}
		newEnv := env.WithActor(actor)
		if newEnv.Actor == nil {
			t.Error("WithActor() Actor should not be nil")
			return
		}
		if newEnv.Actor.Type != ActorTypeAdmin {
			t.Errorf("WithActor() Actor.Type = %q, want %q", newEnv.Actor.Type, ActorTypeAdmin)
		}
	})
}

func TestEnvelope_EventDomainAndAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		eventType  string
		wantDomain string
		wantAction string
	}{
		{"ride.completed", "ride", "completed"},
		{"user.registered", "user", "registered"},
		{"payment.refunded", "payment", "refunded"},
		{"driver.location_updated", "driver", "location_updated"},
		{"safety.emergency_triggered", "safety", "emergency_triggered"},
		{"nodot", "nodot", ""},
		{"multiple.dots.here", "multiple", "dots.here"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			t.Parallel()

			env := &Envelope{Type: tt.eventType}

			if got := env.EventDomain(); got != tt.wantDomain {
				t.Errorf("EventDomain() = %q, want %q", got, tt.wantDomain)
			}
			if got := env.EventAction(); got != tt.wantAction {
				t.Errorf("EventAction() = %q, want %q", got, tt.wantAction)
			}
		})
	}
}

func TestEnvelope_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	original, err := New(&Config{
		Type:          "ride.completed",
		Version:       "1.0",
		Source:        "ride-service",
		CorrelationID: "corr-123",
		CausationID:   "cause-456",
		Actor: &Actor{
			ID:   ids.MustNewUserID(),
			Type: ActorTypeUser,
		},
		Payload: testPayload{RideID: "ride-1", DriverID: "driver-1", Fare: 1500},
	})
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	// Marshal to JSON
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	// Unmarshal back
	var decoded Envelope
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	// Verify fields
	if decoded.ID != original.ID {
		t.Errorf("JSON round-trip ID = %q, want %q", decoded.ID, original.ID)
	}
	if decoded.Type != original.Type {
		t.Errorf("JSON round-trip Type = %q, want %q", decoded.Type, original.Type)
	}
	if decoded.Version != original.Version {
		t.Errorf("JSON round-trip Version = %q, want %q", decoded.Version, original.Version)
	}
	if decoded.Source != original.Source {
		t.Errorf("JSON round-trip Source = %q, want %q", decoded.Source, original.Source)
	}
	if decoded.CorrelationID != original.CorrelationID {
		t.Errorf("JSON round-trip CorrelationID = %q, want %q", decoded.CorrelationID, original.CorrelationID)
	}
	if decoded.CausationID != original.CausationID {
		t.Errorf("JSON round-trip CausationID = %q, want %q", decoded.CausationID, original.CausationID)
	}
	if decoded.Actor == nil {
		t.Error("JSON round-trip Actor should not be nil")
	} else if decoded.Actor.Type != original.Actor.Type {
		t.Errorf("JSON round-trip Actor.Type = %q, want %q", decoded.Actor.Type, original.Actor.Type)
	}

	// Verify payload can be unmarshaled
	var payload testPayload
	err = decoded.UnmarshalPayload(&payload)
	if err != nil {
		t.Fatalf("UnmarshalPayload() error: %v", err)
	}
	if payload.RideID != "ride-1" {
		t.Errorf("JSON round-trip payload RideID = %q, want %q", payload.RideID, "ride-1")
	}
}

func TestActor_IsZero(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		actor  Actor
		isZero bool
	}{
		{"zero actor", Actor{}, true},
		{"only type", Actor{Type: ActorTypeUser}, true},
		{"only ID", Actor{ID: ids.MustNewUserID()}, true},
		{"full actor", Actor{ID: ids.MustNewUserID(), Type: ActorTypeUser}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.actor.IsZero(); got != tt.isZero {
				t.Errorf("IsZero() = %v, want %v", got, tt.isZero)
			}
		})
	}
}

func TestVersionValidation(t *testing.T) {
	t.Parallel()

	validVersions := []string{"1.0", "2.1", "10.20", "0.1", "99.99"}
	invalidVersions := []string{"v1.0", "1.0.0", "1", "1.0.0-beta", "a.b", "1.0.0.0", ""}

	for _, v := range validVersions {
		t.Run("valid_"+v, func(t *testing.T) {
			t.Parallel()

			_, err := New(&Config{
				Type:          "test.event",
				Version:       v,
				Source:        "test-service",
				CorrelationID: "corr-123",
				Payload:       map[string]string{"key": "value"},
			})
			if err != nil {
				t.Errorf("New() with version %q should not error: %v", v, err)
			}
		})
	}

	for _, v := range invalidVersions {
		t.Run("invalid_"+v, func(t *testing.T) {
			t.Parallel()

			_, err := New(&Config{
				Type:          "test.event",
				Version:       v,
				Source:        "test-service",
				CorrelationID: "corr-123",
				Payload:       map[string]string{"key": "value"},
			})
			if err == nil && v != "" {
				t.Errorf("New() with version %q should error", v)
			}
		})
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || substr == "" ||
		(s != "" && substr != "" && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
