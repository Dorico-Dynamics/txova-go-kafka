package events

import (
	"time"

	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/geo"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
)

// RideRequested is the payload for ride.requested events.
type RideRequested struct {
	RideID      ids.RideID        `json:"ride_id"`
	RiderID     ids.UserID        `json:"rider_id"`
	Pickup      geo.Location      `json:"pickup"`
	Dropoff     geo.Location      `json:"dropoff"`
	ServiceType enums.ServiceType `json:"service_type"`
}

// RideDriverAssigned is the payload for ride.driver_assigned events.
type RideDriverAssigned struct {
	RideID     ids.RideID    `json:"ride_id"`
	DriverID   ids.DriverID  `json:"driver_id"`
	VehicleID  ids.VehicleID `json:"vehicle_id"`
	ETAMinutes int           `json:"eta_minutes"`
}

// RideDriverArrived is the payload for ride.driver_arrived events.
type RideDriverArrived struct {
	RideID      ids.RideID `json:"ride_id"`
	ArrivalTime time.Time  `json:"arrival_time"`
}

// RideStarted is the payload for ride.started events.
type RideStarted struct {
	RideID        ids.RideID   `json:"ride_id"`
	StartTime     time.Time    `json:"start_time"`
	StartLocation geo.Location `json:"start_location"`
}

// RideCompleted is the payload for ride.completed events.
type RideCompleted struct {
	RideID       ids.RideID `json:"ride_id"`
	EndTime      time.Time  `json:"end_time"`
	DistanceKM   float64    `json:"distance_km"`
	DurationMins int        `json:"duration_mins"`
	Fare         int64      `json:"fare"`
}

// CancelledBy represents who cancelled a ride.
type CancelledBy string

const (
	CancelledByRider  CancelledBy = "rider"
	CancelledByDriver CancelledBy = "driver"
	CancelledBySystem CancelledBy = "system"
)

// RideCancelled is the payload for ride.cancelled events.
type RideCancelled struct {
	RideID          ids.RideID               `json:"ride_id"`
	CancelledBy     CancelledBy              `json:"cancelled_by"`
	Reason          enums.CancellationReason `json:"reason"`
	CancellationFee int64                    `json:"cancellation_fee"`
}

// RatedBy represents who gave the rating.
type RatedBy string

const (
	RatedByRider  RatedBy = "rider"
	RatedByDriver RatedBy = "driver"
)

// RideRated is the payload for ride.rated events.
type RideRated struct {
	RideID  ids.RideID `json:"ride_id"`
	Rating  int        `json:"rating"`
	RatedBy RatedBy    `json:"rated_by"`
	Review  string     `json:"review,omitempty"`
}
