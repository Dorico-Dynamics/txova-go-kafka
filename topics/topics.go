// Package topics provides topic definitions and naming conventions for Kafka.
package topics

import "github.com/Dorico-Dynamics/txova-go-kafka/events"

// Topic represents a Kafka topic name.
type Topic string

// Topic constants following the naming convention: txova.{domain}.{version}.
const (
	// Users topic for user-related events.
	Users Topic = "txova.users.v1"
	// Drivers topic for driver-related events.
	Drivers Topic = "txova.drivers.v1"
	// Rides topic for ride-related events.
	Rides Topic = "txova.rides.v1"
	// Payments topic for payment-related events.
	Payments Topic = "txova.payments.v1"
	// Safety topic for safety-related events.
	Safety Topic = "txova.safety.v1"
	// Locations topic for location updates.
	Locations Topic = "txova.locations.v1"
)

// String returns the string representation of the topic.
func (t Topic) String() string {
	return string(t)
}

// DLQ returns the dead letter queue topic name for this topic.
func (t Topic) DLQ() Topic {
	return Topic(string(t) + ".dlq")
}

// Valid returns true if the topic is a known topic.
func (t Topic) Valid() bool {
	switch t {
	case Users, Drivers, Rides, Payments, Safety, Locations:
		return true
	default:
		return false
	}
}

// ForEventType returns the appropriate topic for the given event type.
func ForEventType(eventType events.EventType) Topic {
	switch eventType.Domain() {
	case "user":
		return Users
	case "driver":
		return Drivers
	case "ride":
		return Rides
	case "payment", "payout":
		return Payments
	case "safety":
		return Safety
	default:
		return ""
	}
}

// All returns all known topics.
func All() []Topic {
	return []Topic{
		Users,
		Drivers,
		Rides,
		Payments,
		Safety,
		Locations,
	}
}

// PartitionKeyField describes which field to use as partition key for a topic.
type PartitionKeyField string

const (
	// PartitionKeyUserID uses user_id as partition key.
	PartitionKeyUserID PartitionKeyField = "user_id"
	// PartitionKeyDriverID uses driver_id as partition key.
	PartitionKeyDriverID PartitionKeyField = "driver_id"
	// PartitionKeyRideID uses ride_id as partition key.
	PartitionKeyRideID PartitionKeyField = "ride_id"
	// PartitionKeyPaymentID uses payment_id as partition key.
	PartitionKeyPaymentID PartitionKeyField = "payment_id"
	// PartitionKeyIncidentID uses incident_id as partition key.
	PartitionKeyIncidentID PartitionKeyField = "incident_id"
)

// PartitionKey returns the partition key field for the given topic.
func (t Topic) PartitionKey() PartitionKeyField {
	switch t {
	case Users:
		return PartitionKeyUserID
	case Drivers, Locations:
		return PartitionKeyDriverID
	case Rides:
		return PartitionKeyRideID
	case Payments:
		return PartitionKeyPaymentID
	case Safety:
		return PartitionKeyIncidentID
	default:
		return ""
	}
}

// Config holds configuration for topic creation.
type Config struct {
	// NumPartitions is the number of partitions for the topic.
	NumPartitions int32
	// ReplicationFactor is the replication factor for the topic.
	ReplicationFactor int16
	// RetentionMs is the retention period in milliseconds.
	RetentionMs int64
}

// Default topic configuration values.
const (
	// DefaultNumPartitions is the default number of partitions for production topics.
	DefaultNumPartitions int32 = 12
	// DefaultReplicationFactor is the default replication factor for production topics.
	DefaultReplicationFactor int16 = 3
	// DefaultRetentionMs is the default retention period (7 days in milliseconds).
	DefaultRetentionMs int64 = 7 * 24 * 60 * 60 * 1000
	// DevNumPartitions is the number of partitions for development topics.
	DevNumPartitions int32 = 3
	// DevReplicationFactor is the replication factor for development topics.
	DevReplicationFactor int16 = 1
	// DevRetentionMs is the retention period for development topics (1 day in milliseconds).
	DevRetentionMs int64 = 24 * 60 * 60 * 1000
)

// DefaultConfig returns the default topic configuration.
func DefaultConfig() Config {
	return Config{
		NumPartitions:     DefaultNumPartitions,
		ReplicationFactor: DefaultReplicationFactor,
		RetentionMs:       DefaultRetentionMs,
	}
}

// DevConfig returns a topic configuration suitable for development.
func DevConfig() Config {
	return Config{
		NumPartitions:     DevNumPartitions,
		ReplicationFactor: DevReplicationFactor,
		RetentionMs:       DevRetentionMs,
	}
}
