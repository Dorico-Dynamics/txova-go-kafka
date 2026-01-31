// Package events provides typed event definitions for Kafka messages.
// Each event type has a corresponding payload struct with strongly-typed fields.
package events

// EventType represents a Kafka event type string.
type EventType string

// Event type constants for all domains.
const (
	// User events.
	EventTypeUserRegistered     EventType = "user.registered"
	EventTypeUserVerified       EventType = "user.verified"
	EventTypeUserProfileUpdated EventType = "user.profile_updated"
	EventTypeUserSuspended      EventType = "user.suspended"
	EventTypeUserDeleted        EventType = "user.deleted"

	// Driver events.
	EventTypeDriverApplicationSubmitted EventType = "driver.application_submitted"
	EventTypeDriverDocumentsSubmitted   EventType = "driver.documents_submitted"
	EventTypeDriverApproved             EventType = "driver.approved"
	EventTypeDriverRejected             EventType = "driver.rejected"
	EventTypeDriverWentOnline           EventType = "driver.went_online"
	EventTypeDriverWentOffline          EventType = "driver.went_offline"
	EventTypeDriverLocationUpdated      EventType = "driver.location_updated"

	// Ride events.
	EventTypeRideRequested      EventType = "ride.requested"
	EventTypeRideDriverAssigned EventType = "ride.driver_assigned"
	EventTypeRideDriverArrived  EventType = "ride.driver_arrived"
	EventTypeRideStarted        EventType = "ride.started"
	EventTypeRideCompleted      EventType = "ride.completed"
	EventTypeRideCancelled      EventType = "ride.cancelled"
	EventTypeRideRated          EventType = "ride.rated"

	// Payment events.
	EventTypePaymentInitiated EventType = "payment.initiated"
	EventTypePaymentCompleted EventType = "payment.completed"
	EventTypePaymentFailed    EventType = "payment.failed"
	EventTypePaymentRefunded  EventType = "payment.refunded"
	EventTypePayoutRequested  EventType = "payout.requested"
	EventTypePayoutCompleted  EventType = "payout.completed"

	// Safety events.
	EventTypeSafetyEmergencyTriggered EventType = "safety.emergency_triggered"
	EventTypeSafetyIncidentReported   EventType = "safety.incident_reported"
	EventTypeSafetyTripShared         EventType = "safety.trip_shared"
)

// String returns the string representation of the event type.
func (e EventType) String() string {
	return string(e)
}

// Domain returns the domain part of the event type (e.g., "user" from "user.registered").
func (e EventType) Domain() string {
	for i, c := range e {
		if c == '.' {
			return string(e[:i])
		}
	}
	return string(e)
}

// Action returns the action part of the event type (e.g., "registered" from "user.registered").
func (e EventType) Action() string {
	for i, c := range e {
		if c == '.' {
			return string(e[i+1:])
		}
	}
	return ""
}

// Valid returns true if the event type is a known type.
func (e EventType) Valid() bool {
	switch e {
	case EventTypeUserRegistered, EventTypeUserVerified, EventTypeUserProfileUpdated,
		EventTypeUserSuspended, EventTypeUserDeleted,
		EventTypeDriverApplicationSubmitted, EventTypeDriverDocumentsSubmitted,
		EventTypeDriverApproved, EventTypeDriverRejected, EventTypeDriverWentOnline,
		EventTypeDriverWentOffline, EventTypeDriverLocationUpdated,
		EventTypeRideRequested, EventTypeRideDriverAssigned, EventTypeRideDriverArrived,
		EventTypeRideStarted, EventTypeRideCompleted, EventTypeRideCancelled, EventTypeRideRated,
		EventTypePaymentInitiated, EventTypePaymentCompleted, EventTypePaymentFailed,
		EventTypePaymentRefunded, EventTypePayoutRequested, EventTypePayoutCompleted,
		EventTypeSafetyEmergencyTriggered, EventTypeSafetyIncidentReported, EventTypeSafetyTripShared:
		return true
	default:
		return false
	}
}

// VersionLatest is the current schema version for all events.
const VersionLatest = "1.0"
