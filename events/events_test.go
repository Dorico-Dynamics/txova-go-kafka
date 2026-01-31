package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Dorico-Dynamics/txova-go-types/contact"
	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/geo"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
)

func TestUserRegistered_JSON(t *testing.T) {
	t.Parallel()

	phone, _ := contact.ParsePhoneNumber("+258841234567")
	email, _ := contact.ParseEmail("test@example.com")
	event := UserRegistered{
		UserID:   ids.MustNewUserID(),
		Phone:    phone,
		UserType: enums.UserTypeRider,
		Email:    &email,
		Name:     "Test User",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var decoded UserRegistered
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if decoded.UserID != event.UserID {
		t.Errorf("UserID = %v, want %v", decoded.UserID, event.UserID)
	}
	if decoded.UserType != event.UserType {
		t.Errorf("UserType = %v, want %v", decoded.UserType, event.UserType)
	}
}

func TestUserVerified_JSON(t *testing.T) {
	t.Parallel()

	event := UserVerified{
		UserID:           ids.MustNewUserID(),
		VerificationType: VerificationTypePhone,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var decoded UserVerified
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if decoded.VerificationType != VerificationTypePhone {
		t.Errorf("VerificationType = %v, want %v", decoded.VerificationType, VerificationTypePhone)
	}
}

func TestDriverApplicationSubmitted_JSON(t *testing.T) {
	t.Parallel()

	event := DriverApplicationSubmitted{
		DriverID: ids.MustNewDriverID(),
		UserID:   ids.MustNewUserID(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var decoded DriverApplicationSubmitted
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if decoded.DriverID != event.DriverID {
		t.Errorf("DriverID = %v, want %v", decoded.DriverID, event.DriverID)
	}
}

func TestDriverLocationUpdated_JSON(t *testing.T) {
	t.Parallel()

	loc := geo.MustNewLocation(-25.9692, 32.5732)
	event := DriverLocationUpdated{
		DriverID: ids.MustNewDriverID(),
		Location: loc,
		Heading:  180.5,
		Speed:    45.2,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var decoded DriverLocationUpdated
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if decoded.Heading != event.Heading {
		t.Errorf("Heading = %v, want %v", decoded.Heading, event.Heading)
	}
	if decoded.Speed != event.Speed {
		t.Errorf("Speed = %v, want %v", decoded.Speed, event.Speed)
	}
}

func TestRideRequested_JSON(t *testing.T) {
	t.Parallel()

	pickup := geo.MustNewLocation(-25.9692, 32.5732)
	dropoff := geo.MustNewLocation(-25.9532, 32.5892)
	event := RideRequested{
		RideID:      ids.MustNewRideID(),
		RiderID:     ids.MustNewUserID(),
		Pickup:      pickup,
		Dropoff:     dropoff,
		ServiceType: enums.ServiceTypeStandard,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var decoded RideRequested
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if decoded.ServiceType != enums.ServiceTypeStandard {
		t.Errorf("ServiceType = %v, want %v", decoded.ServiceType, enums.ServiceTypeStandard)
	}
}

func TestRideCompleted_JSON(t *testing.T) {
	t.Parallel()

	event := RideCompleted{
		RideID:       ids.MustNewRideID(),
		EndTime:      time.Now().UTC(),
		DistanceKM:   5.5,
		DurationMins: 15,
		Fare:         15000,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var decoded RideCompleted
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if decoded.DistanceKM != event.DistanceKM {
		t.Errorf("DistanceKM = %v, want %v", decoded.DistanceKM, event.DistanceKM)
	}
	if decoded.Fare != event.Fare {
		t.Errorf("Fare = %v, want %v", decoded.Fare, event.Fare)
	}
}

func TestRideCancelled_JSON(t *testing.T) {
	t.Parallel()

	event := RideCancelled{
		RideID:          ids.MustNewRideID(),
		CancelledBy:     CancelledByRider,
		Reason:          enums.CancellationReasonRiderCancelled,
		CancellationFee: 500,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var decoded RideCancelled
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if decoded.CancelledBy != CancelledByRider {
		t.Errorf("CancelledBy = %v, want %v", decoded.CancelledBy, CancelledByRider)
	}
}

func TestPaymentInitiated_JSON(t *testing.T) {
	t.Parallel()

	event := PaymentInitiated{
		PaymentID: ids.MustNewPaymentID(),
		RideID:    ids.MustNewRideID(),
		Amount:    15000,
		Method:    enums.PaymentMethodMPesa,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var decoded PaymentInitiated
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if decoded.Method != enums.PaymentMethodMPesa {
		t.Errorf("Method = %v, want %v", decoded.Method, enums.PaymentMethodMPesa)
	}
}

func TestPayoutID(t *testing.T) {
	t.Parallel()

	id, err := NewPayoutID()
	if err != nil {
		t.Fatalf("NewPayoutID() error: %v", err)
	}

	if id.IsZero() {
		t.Error("NewPayoutID() returned zero ID")
	}

	// Test MustNewPayoutID doesn't panic
	mustID := MustNewPayoutID()
	if mustID.IsZero() {
		t.Error("MustNewPayoutID() returned zero ID")
	}
}

func TestSafetyEmergencyTriggered_JSON(t *testing.T) {
	t.Parallel()

	loc := geo.MustNewLocation(-25.9692, 32.5732)
	event := SafetyEmergencyTriggered{
		RideID:        ids.MustNewRideID(),
		UserID:        ids.MustNewUserID(),
		Location:      loc,
		EmergencyType: enums.EmergencyTypeAccident,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var decoded SafetyEmergencyTriggered
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if decoded.EmergencyType != enums.EmergencyTypeAccident {
		t.Errorf("EmergencyType = %v, want %v", decoded.EmergencyType, enums.EmergencyTypeAccident)
	}
}

func TestSafetyIncidentReported_JSON(t *testing.T) {
	t.Parallel()

	event := SafetyIncidentReported{
		IncidentID: ids.MustNewIncidentID(),
		ReporterID: ids.MustNewUserID(),
		RideID:     ids.MustNewRideID(),
		Severity:   enums.IncidentSeverityHigh,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var decoded SafetyIncidentReported
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if decoded.Severity != enums.IncidentSeverityHigh {
		t.Errorf("Severity = %v, want %v", decoded.Severity, enums.IncidentSeverityHigh)
	}
}

func TestDriverWentOffline_Reasons(t *testing.T) {
	t.Parallel()

	reasons := []OfflineReason{
		OfflineReasonManual,
		OfflineReasonTimeout,
		OfflineReasonEndOfShift,
		OfflineReasonAppClosed,
		OfflineReasonLowBattery,
		OfflineReasonAdminAction,
	}

	for _, reason := range reasons {
		t.Run(string(reason), func(t *testing.T) {
			t.Parallel()

			event := DriverWentOffline{
				DriverID: ids.MustNewDriverID(),
				Reason:   reason,
			}

			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("json.Marshal() error: %v", err)
			}

			var decoded DriverWentOffline
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error: %v", err)
			}

			if decoded.Reason != reason {
				t.Errorf("Reason = %v, want %v", decoded.Reason, reason)
			}
		})
	}
}

func TestVerificationTypes(t *testing.T) {
	t.Parallel()

	types := []VerificationType{
		VerificationTypePhone,
		VerificationTypeEmail,
		VerificationTypeID,
	}

	for _, vt := range types {
		t.Run(string(vt), func(t *testing.T) {
			t.Parallel()

			event := UserVerified{
				UserID:           ids.MustNewUserID(),
				VerificationType: vt,
			}

			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("json.Marshal() error: %v", err)
			}

			var decoded UserVerified
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error: %v", err)
			}

			if decoded.VerificationType != vt {
				t.Errorf("VerificationType = %v, want %v", decoded.VerificationType, vt)
			}
		})
	}
}

func TestDeletionTypes(t *testing.T) {
	t.Parallel()

	types := []DeletionType{
		DeletionTypeUserRequested,
		DeletionTypeAdminAction,
		DeletionTypeInactivity,
	}

	for _, dt := range types {
		t.Run(string(dt), func(t *testing.T) {
			t.Parallel()

			event := UserDeleted{
				UserID:       ids.MustNewUserID(),
				DeletionType: dt,
			}

			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("json.Marshal() error: %v", err)
			}

			var decoded UserDeleted
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error: %v", err)
			}

			if decoded.DeletionType != dt {
				t.Errorf("DeletionType = %v, want %v", decoded.DeletionType, dt)
			}
		})
	}
}

func TestCancelledByTypes(t *testing.T) {
	t.Parallel()

	types := []CancelledBy{
		CancelledByRider,
		CancelledByDriver,
		CancelledBySystem,
	}

	for _, cb := range types {
		t.Run(string(cb), func(t *testing.T) {
			t.Parallel()

			event := RideCancelled{
				RideID:          ids.MustNewRideID(),
				CancelledBy:     cb,
				Reason:          enums.CancellationReasonOther,
				CancellationFee: 0,
			}

			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("json.Marshal() error: %v", err)
			}

			var decoded RideCancelled
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error: %v", err)
			}

			if decoded.CancelledBy != cb {
				t.Errorf("CancelledBy = %v, want %v", decoded.CancelledBy, cb)
			}
		})
	}
}

func TestRatedByTypes(t *testing.T) {
	t.Parallel()

	types := []RatedBy{
		RatedByRider,
		RatedByDriver,
	}

	for _, rb := range types {
		t.Run(string(rb), func(t *testing.T) {
			t.Parallel()

			event := RideRated{
				RideID:  ids.MustNewRideID(),
				Rating:  5,
				RatedBy: rb,
				Review:  "Great ride!",
			}

			data, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("json.Marshal() error: %v", err)
			}

			var decoded RideRated
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal() error: %v", err)
			}

			if decoded.RatedBy != rb {
				t.Errorf("RatedBy = %v, want %v", decoded.RatedBy, rb)
			}
		})
	}
}
