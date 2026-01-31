package topics

// ConsumerGroup represents a Kafka consumer group ID.
type ConsumerGroup string

// Consumer group constants for each service.
const (
	ConsumerGroupUserService         ConsumerGroup = "user-service-group"
	ConsumerGroupDriverService       ConsumerGroup = "driver-service-group"
	ConsumerGroupRideService         ConsumerGroup = "ride-service-group"
	ConsumerGroupPaymentService      ConsumerGroup = "payment-service-group"
	ConsumerGroupNotificationService ConsumerGroup = "notification-service-group"
	ConsumerGroupSafetyService       ConsumerGroup = "safety-service-group"
	ConsumerGroupOperationsService   ConsumerGroup = "operations-service-group"
	ConsumerGroupAnalyticsService    ConsumerGroup = "analytics-service-group"
)

// String returns the string representation of the consumer group.
func (c ConsumerGroup) String() string {
	return string(c)
}

// Valid returns true if the consumer group is a known group.
func (c ConsumerGroup) Valid() bool {
	switch c {
	case ConsumerGroupUserService, ConsumerGroupDriverService, ConsumerGroupRideService,
		ConsumerGroupPaymentService, ConsumerGroupNotificationService,
		ConsumerGroupSafetyService, ConsumerGroupOperationsService, ConsumerGroupAnalyticsService:
		return true
	default:
		return false
	}
}

// ForService returns the consumer group for the given service name.
func ForService(serviceName string) ConsumerGroup {
	return ConsumerGroup(serviceName + "-group")
}
