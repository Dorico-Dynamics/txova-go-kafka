package events

import (
	"github.com/Dorico-Dynamics/txova-go-types/contact"
	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/geo"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
)

// SafetyEmergencyTriggered is the payload for safety.emergency_triggered events.
type SafetyEmergencyTriggered struct {
	RideID        ids.RideID          `json:"ride_id"`
	UserID        ids.UserID          `json:"user_id"`
	Location      geo.Location        `json:"location"`
	EmergencyType enums.EmergencyType `json:"emergency_type"`
}

// SafetyIncidentReported is the payload for safety.incident_reported events.
type SafetyIncidentReported struct {
	IncidentID ids.IncidentID         `json:"incident_id"`
	ReporterID ids.UserID             `json:"reporter_id"`
	RideID     ids.RideID             `json:"ride_id"`
	Severity   enums.IncidentSeverity `json:"severity"`
}

// SafetyTripShared is the payload for safety.trip_shared events.
type SafetyTripShared struct {
	RideID          ids.RideID          `json:"ride_id"`
	SharedWithPhone contact.PhoneNumber `json:"shared_with_phone"`
}
