package topics

import (
	"testing"

	"github.com/Dorico-Dynamics/txova-go-kafka/events"
)

func TestTopic_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		topic Topic
		want  string
	}{
		{Users, "txova.users.v1"},
		{Drivers, "txova.drivers.v1"},
		{Rides, "txova.rides.v1"},
		{Payments, "txova.payments.v1"},
		{Safety, "txova.safety.v1"},
		{Locations, "txova.locations.v1"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.topic.String(); got != tt.want {
				t.Errorf("Topic.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTopic_DLQ(t *testing.T) {
	t.Parallel()

	tests := []struct {
		topic   Topic
		wantDLQ Topic
	}{
		{Users, "txova.users.v1.dlq"},
		{Drivers, "txova.drivers.v1.dlq"},
		{Rides, "txova.rides.v1.dlq"},
		{Payments, "txova.payments.v1.dlq"},
		{Safety, "txova.safety.v1.dlq"},
		{Locations, "txova.locations.v1.dlq"},
	}

	for _, tt := range tests {
		t.Run(string(tt.topic), func(t *testing.T) {
			t.Parallel()
			if got := tt.topic.DLQ(); got != tt.wantDLQ {
				t.Errorf("Topic.DLQ() = %q, want %q", got, tt.wantDLQ)
			}
		})
	}
}

func TestTopic_Valid(t *testing.T) {
	t.Parallel()

	validTopics := []Topic{Users, Drivers, Rides, Payments, Safety, Locations}
	for _, topic := range validTopics {
		t.Run("valid_"+string(topic), func(t *testing.T) {
			t.Parallel()
			if !topic.Valid() {
				t.Errorf("Topic.Valid() = false for %q, want true", topic)
			}
		})
	}

	invalidTopics := []Topic{"", "unknown", "txova.invalid.v1"}
	for _, topic := range invalidTopics {
		t.Run("invalid_"+string(topic), func(t *testing.T) {
			t.Parallel()
			if topic.Valid() {
				t.Errorf("Topic.Valid() = true for %q, want false", topic)
			}
		})
	}
}

func TestForEventType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		eventType events.EventType
		wantTopic Topic
	}{
		{events.EventTypeUserRegistered, Users},
		{events.EventTypeUserVerified, Users},
		{events.EventTypeDriverApproved, Drivers},
		{events.EventTypeDriverLocationUpdated, Drivers},
		{events.EventTypeRideRequested, Rides},
		{events.EventTypeRideCompleted, Rides},
		{events.EventTypePaymentInitiated, Payments},
		{events.EventTypePayoutRequested, Payments},
		{events.EventTypeSafetyEmergencyTriggered, Safety},
		{events.EventType("unknown.event"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			t.Parallel()
			if got := ForEventType(tt.eventType); got != tt.wantTopic {
				t.Errorf("ForEventType(%q) = %q, want %q", tt.eventType, got, tt.wantTopic)
			}
		})
	}
}

func TestAll(t *testing.T) {
	t.Parallel()

	topics := All()
	if len(topics) != 6 {
		t.Errorf("All() returned %d topics, want 6", len(topics))
	}

	expected := map[Topic]bool{
		Users:     false,
		Drivers:   false,
		Rides:     false,
		Payments:  false,
		Safety:    false,
		Locations: false,
	}

	for _, topic := range topics {
		if _, ok := expected[topic]; !ok {
			t.Errorf("All() returned unexpected topic: %q", topic)
		}
		expected[topic] = true
	}

	for topic, found := range expected {
		if !found {
			t.Errorf("All() missing topic: %q", topic)
		}
	}
}

func TestTopic_PartitionKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		topic   Topic
		wantKey PartitionKeyField
	}{
		{Users, PartitionKeyUserID},
		{Drivers, PartitionKeyDriverID},
		{Rides, PartitionKeyRideID},
		{Payments, PartitionKeyPaymentID},
		{Safety, PartitionKeyIncidentID},
		{Locations, PartitionKeyDriverID},
		{Topic("unknown"), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.topic), func(t *testing.T) {
			t.Parallel()
			if got := tt.topic.PartitionKey(); got != tt.wantKey {
				t.Errorf("Topic.PartitionKey() = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if cfg.NumPartitions != 12 {
		t.Errorf("DefaultConfig().NumPartitions = %d, want 12", cfg.NumPartitions)
	}
	if cfg.ReplicationFactor != 3 {
		t.Errorf("DefaultConfig().ReplicationFactor = %d, want 3", cfg.ReplicationFactor)
	}
	// 7 days in milliseconds
	expectedRetention := int64(7 * 24 * 60 * 60 * 1000)
	if cfg.RetentionMs != expectedRetention {
		t.Errorf("DefaultConfig().RetentionMs = %d, want %d", cfg.RetentionMs, expectedRetention)
	}
}

func TestDevConfig(t *testing.T) {
	t.Parallel()

	cfg := DevConfig()

	if cfg.NumPartitions != 3 {
		t.Errorf("DevConfig().NumPartitions = %d, want 3", cfg.NumPartitions)
	}
	if cfg.ReplicationFactor != 1 {
		t.Errorf("DevConfig().ReplicationFactor = %d, want 1", cfg.ReplicationFactor)
	}
	// 1 day in milliseconds
	expectedRetention := int64(24 * 60 * 60 * 1000)
	if cfg.RetentionMs != expectedRetention {
		t.Errorf("DevConfig().RetentionMs = %d, want %d", cfg.RetentionMs, expectedRetention)
	}
}
