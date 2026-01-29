# txova-go-kafka

Event streaming library providing Kafka producer/consumer implementations, event envelope standards, and typed event definitions.

## Overview

`txova-go-kafka` provides a complete event streaming solution for Txova services, including standardized event envelopes, typed event definitions, producer/consumer implementations, and dead letter queue handling.

**Module:** `github.com/txova/txova-go-kafka`

## Features

- **Event Envelope** - Standardized event format with metadata
- **Typed Events** - Pre-defined events for all domains (user, driver, ride, payment, safety)
- **Producer** - Sync/async publishing with retry logic
- **Consumer** - Consumer groups with manual commit
- **Dead Letter Queue** - Failed message handling and replay

## Packages

| Package | Description |
|---------|-------------|
| `envelope` | Standard event envelope format |
| `events` | Typed event definitions |
| `topics` | Topic name constants |
| `producer` | Event publishing |
| `consumer` | Event consumption |
| `dlq` | Dead letter queue handling |

## Installation

```bash
go get github.com/txova/txova-go-kafka
```

## Usage

### Event Envelope

```go
import "github.com/txova/txova-go-kafka/envelope"

event := envelope.New(envelope.Config{
    Type:          "ride.completed",
    Version:       "1.0",
    Source:        "ride-service",
    CorrelationID: requestID,
    Payload: RideCompletedPayload{
        RideID:      rideID,
        DriverID:    driverID,
        Fare:        fare,
        DistanceKM:  distance,
    },
})
```

### Publishing Events

```go
import "github.com/txova/txova-go-kafka/producer"

p := producer.New(producer.Config{
    Brokers: []string{"kafka:9092"},
    Acks:    producer.AcksAll,
    Retries: 3,
})
defer p.Close()

// Sync publish (waits for acknowledgment)
err := p.Publish(ctx, "txova.rides.v1", event)

// Async publish with callback
p.PublishAsync(ctx, "txova.rides.v1", event, func(err error) {
    if err != nil {
        log.Error().Err(err).Msg("Failed to publish")
    }
})
```

### Consuming Events

```go
import "github.com/txova/txova-go-kafka/consumer"

c := consumer.New(consumer.Config{
    Brokers:   []string{"kafka:9092"},
    GroupID:   "payment-service-group",
    Topics:    []string{"txova.rides.v1"},
    AutoReset: consumer.OffsetEarliest,
})

// Register typed handlers
c.Handle("ride.completed", func(ctx context.Context, event envelope.Event) error {
    var payload RideCompletedPayload
    if err := event.UnmarshalPayload(&payload); err != nil {
        return err
    }
    return processRideCompleted(ctx, payload)
})

// Start consuming
c.Start(ctx)
```

### Event Types

**User Events:**
- `user.registered` - New user created
- `user.verified` - User completed verification
- `user.profile_updated` - Profile information changed
- `user.suspended` - Account suspended

**Driver Events:**
- `driver.approved` - Driver application approved
- `driver.went_online` - Driver became available
- `driver.went_offline` - Driver became unavailable
- `driver.location_updated` - Driver position changed

**Ride Events:**
- `ride.requested` - Rider requested ride
- `ride.driver_assigned` - Driver matched to ride
- `ride.started` - Trip began
- `ride.completed` - Trip finished
- `ride.cancelled` - Trip cancelled

**Payment Events:**
- `payment.initiated` - Payment started
- `payment.completed` - Payment successful
- `payment.failed` - Payment failed
- `payment.refunded` - Payment refunded

**Safety Events:**
- `safety.emergency_triggered` - SOS activated
- `safety.incident_reported` - Incident reported

### Topics

| Topic | Partition Key | Description |
|-------|---------------|-------------|
| `txova.users.v1` | user_id | User events |
| `txova.drivers.v1` | driver_id | Driver events |
| `txova.rides.v1` | ride_id | Ride events |
| `txova.payments.v1` | payment_id | Payment events |
| `txova.safety.v1` | incident_id | Safety events |
| `txova.locations.v1` | driver_id | Location updates |

### Dead Letter Queue

```go
import "github.com/txova/txova-go-kafka/dlq"

// Failed messages automatically route to DLQ after max retries
// DLQ topic: txova.{original-topic}.dlq

// Replay DLQ messages
replayer := dlq.NewReplayer(kafkaConfig)
replayer.Replay(ctx, "txova.rides.v1.dlq", func(event envelope.Event) error {
    // Reprocess failed event
    return processEvent(ctx, event)
})
```

## Consumer Groups

| Service | Group ID |
|---------|----------|
| user-service | user-service-group |
| driver-service | driver-service-group |
| ride-service | ride-service-group |
| payment-service | payment-service-group |
| notification-service | notification-service-group |
| safety-service | safety-service-group |
| operations-service | operations-service-group |

## Dependencies

**Internal:**
- `txova-go-types`
- `txova-go-core`

**External:**
- `github.com/IBM/sarama` - Kafka client

## Development

### Requirements

- Go 1.25+
- Kafka 3.x+

### Testing

```bash
go test ./...
```

### Test Coverage Target

> 85%

## License

Proprietary - Dorico Dynamics
