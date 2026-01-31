package events

import "testing"

func TestEventType_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		eventType EventType
		want      string
	}{
		{EventTypeUserRegistered, "user.registered"},
		{EventTypeDriverApproved, "driver.approved"},
		{EventTypeRideCompleted, "ride.completed"},
		{EventTypePaymentInitiated, "payment.initiated"},
		{EventTypeSafetyEmergencyTriggered, "safety.emergency_triggered"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.eventType.String(); got != tt.want {
				t.Errorf("EventType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEventType_Domain(t *testing.T) {
	t.Parallel()

	tests := []struct {
		eventType  EventType
		wantDomain string
	}{
		{EventTypeUserRegistered, "user"},
		{EventTypeUserVerified, "user"},
		{EventTypeDriverApplicationSubmitted, "driver"},
		{EventTypeDriverWentOnline, "driver"},
		{EventTypeRideRequested, "ride"},
		{EventTypeRideCompleted, "ride"},
		{EventTypePaymentInitiated, "payment"},
		{EventTypePayoutRequested, "payout"},
		{EventTypeSafetyEmergencyTriggered, "safety"},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			t.Parallel()
			if got := tt.eventType.Domain(); got != tt.wantDomain {
				t.Errorf("EventType.Domain() = %q, want %q", got, tt.wantDomain)
			}
		})
	}
}

func TestEventType_Action(t *testing.T) {
	t.Parallel()

	tests := []struct {
		eventType  EventType
		wantAction string
	}{
		{EventTypeUserRegistered, "registered"},
		{EventTypeUserVerified, "verified"},
		{EventTypeDriverApplicationSubmitted, "application_submitted"},
		{EventTypeDriverWentOnline, "went_online"},
		{EventTypeRideRequested, "requested"},
		{EventTypeRideCompleted, "completed"},
		{EventTypePaymentInitiated, "initiated"},
		{EventTypePayoutRequested, "requested"},
		{EventTypeSafetyEmergencyTriggered, "emergency_triggered"},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			t.Parallel()
			if got := tt.eventType.Action(); got != tt.wantAction {
				t.Errorf("EventType.Action() = %q, want %q", got, tt.wantAction)
			}
		})
	}
}

func TestEventType_Valid(t *testing.T) {
	t.Parallel()

	validTypes := []EventType{
		EventTypeUserRegistered, EventTypeUserVerified, EventTypeUserProfileUpdated,
		EventTypeUserSuspended, EventTypeUserDeleted,
		EventTypeDriverApplicationSubmitted, EventTypeDriverDocumentsSubmitted,
		EventTypeDriverApproved, EventTypeDriverRejected, EventTypeDriverWentOnline,
		EventTypeDriverWentOffline, EventTypeDriverLocationUpdated,
		EventTypeRideRequested, EventTypeRideDriverAssigned, EventTypeRideDriverArrived,
		EventTypeRideStarted, EventTypeRideCompleted, EventTypeRideCancelled, EventTypeRideRated,
		EventTypePaymentInitiated, EventTypePaymentCompleted, EventTypePaymentFailed,
		EventTypePaymentRefunded, EventTypePayoutRequested, EventTypePayoutCompleted,
		EventTypeSafetyEmergencyTriggered, EventTypeSafetyIncidentReported, EventTypeSafetyTripShared,
	}

	for _, et := range validTypes {
		t.Run("valid_"+string(et), func(t *testing.T) {
			t.Parallel()
			if !et.Valid() {
				t.Errorf("EventType.Valid() = false for %q, want true", et)
			}
		})
	}

	invalidTypes := []EventType{
		"",
		"invalid",
		"user.unknown",
		"unknown.event",
	}

	for _, et := range invalidTypes {
		t.Run("invalid_"+string(et), func(t *testing.T) {
			t.Parallel()
			if et.Valid() {
				t.Errorf("EventType.Valid() = true for %q, want false", et)
			}
		})
	}
}

func TestEventType_DomainNoDot(t *testing.T) {
	t.Parallel()

	et := EventType("nodot")
	if got := et.Domain(); got != "nodot" {
		t.Errorf("EventType.Domain() = %q, want %q", got, "nodot")
	}
	if got := et.Action(); got != "" {
		t.Errorf("EventType.Action() = %q, want empty string", got)
	}
}

func TestVersionLatest(t *testing.T) {
	t.Parallel()

	if VersionLatest != "1.0" {
		t.Errorf("VersionLatest = %q, want %q", VersionLatest, "1.0")
	}
}
