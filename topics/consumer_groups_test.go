package topics

import "testing"

func TestConsumerGroup_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		group ConsumerGroup
		want  string
	}{
		{ConsumerGroupUserService, "user-service-group"},
		{ConsumerGroupDriverService, "driver-service-group"},
		{ConsumerGroupRideService, "ride-service-group"},
		{ConsumerGroupPaymentService, "payment-service-group"},
		{ConsumerGroupNotificationService, "notification-service-group"},
		{ConsumerGroupSafetyService, "safety-service-group"},
		{ConsumerGroupOperationsService, "operations-service-group"},
		{ConsumerGroupAnalyticsService, "analytics-service-group"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.group.String(); got != tt.want {
				t.Errorf("ConsumerGroup.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConsumerGroup_Valid(t *testing.T) {
	t.Parallel()

	validGroups := []ConsumerGroup{
		ConsumerGroupUserService, ConsumerGroupDriverService, ConsumerGroupRideService,
		ConsumerGroupPaymentService, ConsumerGroupNotificationService,
		ConsumerGroupSafetyService, ConsumerGroupOperationsService, ConsumerGroupAnalyticsService,
	}

	for _, group := range validGroups {
		t.Run("valid_"+string(group), func(t *testing.T) {
			t.Parallel()
			if !group.Valid() {
				t.Errorf("ConsumerGroup.Valid() = false for %q, want true", group)
			}
		})
	}

	invalidGroups := []ConsumerGroup{"", "unknown", "some-other-group"}
	for _, group := range invalidGroups {
		t.Run("invalid_"+string(group), func(t *testing.T) {
			t.Parallel()
			if group.Valid() {
				t.Errorf("ConsumerGroup.Valid() = true for %q, want false", group)
			}
		})
	}
}

func TestForService(t *testing.T) {
	t.Parallel()

	tests := []struct {
		serviceName string
		want        ConsumerGroup
	}{
		{"user-service", "user-service-group"},
		{"driver-service", "driver-service-group"},
		{"ride-service", "ride-service-group"},
		{"my-custom-service", "my-custom-service-group"},
	}

	for _, tt := range tests {
		t.Run(tt.serviceName, func(t *testing.T) {
			t.Parallel()
			if got := ForService(tt.serviceName); got != tt.want {
				t.Errorf("ForService(%q) = %q, want %q", tt.serviceName, got, tt.want)
			}
		})
	}
}
