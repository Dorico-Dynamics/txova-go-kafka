package events

import (
	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/geo"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
)

// DriverApplicationSubmitted is the payload for driver.application_submitted events.
type DriverApplicationSubmitted struct {
	DriverID ids.DriverID `json:"driver_id"`
	UserID   ids.UserID   `json:"user_id"`
}

// DriverDocumentsSubmitted is the payload for driver.documents_submitted events.
type DriverDocumentsSubmitted struct {
	DriverID      ids.DriverID         `json:"driver_id"`
	DocumentTypes []enums.DocumentType `json:"document_types"`
}

// DriverApproved is the payload for driver.approved events.
type DriverApproved struct {
	DriverID   ids.DriverID `json:"driver_id"`
	ApprovedBy ids.UserID   `json:"approved_by"`
}

// DriverRejected is the payload for driver.rejected events.
type DriverRejected struct {
	DriverID   ids.DriverID `json:"driver_id"`
	Reason     string       `json:"reason"`
	RejectedBy ids.UserID   `json:"rejected_by"`
}

// DriverWentOnline is the payload for driver.went_online events.
type DriverWentOnline struct {
	DriverID  ids.DriverID  `json:"driver_id"`
	Location  geo.Location  `json:"location"`
	VehicleID ids.VehicleID `json:"vehicle_id"`
}

// OfflineReason represents why a driver went offline.
type OfflineReason string

const (
	OfflineReasonManual      OfflineReason = "manual"
	OfflineReasonTimeout     OfflineReason = "timeout"
	OfflineReasonEndOfShift  OfflineReason = "end_of_shift"
	OfflineReasonAppClosed   OfflineReason = "app_closed"
	OfflineReasonLowBattery  OfflineReason = "low_battery"
	OfflineReasonAdminAction OfflineReason = "admin_action"
)

// DriverWentOffline is the payload for driver.went_offline events.
type DriverWentOffline struct {
	DriverID ids.DriverID  `json:"driver_id"`
	Reason   OfflineReason `json:"reason"`
}

// DriverLocationUpdated is the payload for driver.location_updated events.
type DriverLocationUpdated struct {
	DriverID ids.DriverID `json:"driver_id"`
	Location geo.Location `json:"location"`
	Heading  float64      `json:"heading"`
	Speed    float64      `json:"speed"`
}
