# txova-go-kafka

Event streaming library providing Kafka producer/consumer implementations, event envelope standards, and typed event definitions for the Txova platform.

## Overview

`txova-go-kafka` provides a complete event streaming solution for Txova services, including:

- Standardized event envelopes with metadata and tracing
- Typed event definitions for all domains (user, driver, ride, payment, safety)
- Producer with idempotent delivery and retry logic
- Consumer groups with manual commit and retry handling
- Dead letter queue (DLQ) for failed message handling

**Module:** `github.com/Dorico-Dynamics/txova-go-kafka`

## Features

| Feature | Description |
|---------|-------------|
| Event Envelope | Standardized format with correlation IDs, versioning, and actor tracking |
| Typed Events | 28 pre-defined event types across 5 domains |
| Producer | Sync publishing with idempotent mode, compression, and batching |
| Consumer | Consumer groups with handler registry and automatic retries |
| DLQ | Dead letter queue with failure metadata and replay support |
| Topics | Pre-defined topics with partition key mappings |

## Packages

| Package | Description |
|---------|-------------|
| `envelope` | Standard event envelope format with validation |
| `events` | Typed event definitions and payloads |
| `topics` | Topic constants and consumer group definitions |
| `producer` | Kafka producer with configuration and error handling |
| `consumer` | Consumer groups with handler registry |
| `dlq` | Dead letter queue message handling |

## Installation

```bash
go get github.com/Dorico-Dynamics/txova-go-kafka
```

## Quick Start

### Publishing Events

```go
import (
    "github.com/Dorico-Dynamics/txova-go-kafka/envelope"
    "github.com/Dorico-Dynamics/txova-go-kafka/events"
    "github.com/Dorico-Dynamics/txova-go-kafka/producer"
)

// Create producer
cfg := producer.DefaultConfig()
cfg.Brokers = []string{"localhost:9092"}
cfg.ClientID = "ride-service"

prod, err := producer.New(cfg, logger)
if err != nil {
    return err
}
defer prod.Close()

// Create envelope
env, err := envelope.NewWithContext(ctx, &envelope.Config{
    Type:    events.EventTypeRideCompleted.String(),
    Version: events.VersionLatest,
    Source:  "ride-service",
    Payload: &events.RideCompleted{
        RideID:       rideID,
        EndTime:      time.Now(),
        DistanceKM:   12.5,
        DurationMins: 25,
        Fare:         5000,
    },
})
if err != nil {
    return err
}

// Publish (auto-routes to txova.rides.v1)
err = prod.Publish(ctx, env, rideID.String())
```

### Consuming Events

```go
import (
    "github.com/Dorico-Dynamics/txova-go-kafka/consumer"
    "github.com/Dorico-Dynamics/txova-go-kafka/envelope"
    "github.com/Dorico-Dynamics/txova-go-kafka/events"
    "github.com/Dorico-Dynamics/txova-go-kafka/topics"
)

// Create handler registry
registry := consumer.NewHandlerRegistry()

registry.Register(events.EventTypeRideCompleted, func(ctx context.Context, env *envelope.Envelope) error {
    var payload events.RideCompleted
    if err := env.UnmarshalPayload(&payload); err != nil {
        return err
    }
    return processRideCompleted(ctx, &payload)
})

// Create consumer
cfg := consumer.DefaultConfig()
cfg.Brokers = []string{"localhost:9092"}
cfg.GroupID = topics.ConsumerGroupRideService.String()
cfg.Topics = []string{topics.Rides.String()}

cons, err := consumer.New(cfg, registry, logger)
if err != nil {
    return err
}

// Start consuming (blocks until context cancelled)
err = cons.Start(ctx)
```

## Event Types

### User Events
| Event | Description |
|-------|-------------|
| `user.registered` | New user created |
| `user.verified` | User completed verification |
| `user.profile_updated` | Profile information changed |
| `user.suspended` | Account suspended |
| `user.deleted` | Account deleted |

### Driver Events
| Event | Description |
|-------|-------------|
| `driver.application_submitted` | Driver application submitted |
| `driver.documents_submitted` | Documents uploaded |
| `driver.approved` | Application approved |
| `driver.rejected` | Application rejected |
| `driver.went_online` | Driver became available |
| `driver.went_offline` | Driver became unavailable |
| `driver.location_updated` | Position updated |

### Ride Events
| Event | Description |
|-------|-------------|
| `ride.requested` | Rider requested ride |
| `ride.driver_assigned` | Driver matched to ride |
| `ride.driver_arrived` | Driver arrived at pickup |
| `ride.started` | Trip began |
| `ride.completed` | Trip finished |
| `ride.cancelled` | Trip cancelled |
| `ride.rated` | Rating submitted |

### Payment Events
| Event | Description |
|-------|-------------|
| `payment.initiated` | Payment started |
| `payment.completed` | Payment successful |
| `payment.failed` | Payment failed |
| `payment.refunded` | Payment refunded |
| `payout.requested` | Driver payout requested |
| `payout.completed` | Driver payout completed |

### Safety Events
| Event | Description |
|-------|-------------|
| `safety.emergency_triggered` | SOS activated |
| `safety.incident_reported` | Incident reported |
| `safety.trip_shared` | Trip shared with contact |

## Topics

| Topic | Partition Key | Description |
|-------|---------------|-------------|
| `txova.users.v1` | user_id | User lifecycle events |
| `txova.drivers.v1` | driver_id | Driver lifecycle events |
| `txova.rides.v1` | ride_id | Ride lifecycle events |
| `txova.payments.v1` | payment_id | Payment events |
| `txova.safety.v1` | incident_id | Safety events |
| `txova.locations.v1` | driver_id | Location updates |

DLQ topics follow the pattern: `{topic}.dlq` (e.g., `txova.rides.v1.dlq`)

## Consumer Groups

| Service | Group ID |
|---------|----------|
| user-service | `user-service-group` |
| driver-service | `driver-service-group` |
| ride-service | `ride-service-group` |
| payment-service | `payment-service-group` |
| notification-service | `notification-service-group` |
| safety-service | `safety-service-group` |
| operations-service | `operations-service-group` |
| analytics-service | `analytics-service-group` |

## Configuration Defaults

### Producer
| Setting | Default |
|---------|---------|
| RequiredAcks | WaitForAll |
| MaxRetries | 3 |
| RetryBackoff | 100ms |
| BatchSize | 16KB |
| Linger | 5ms |
| Compression | Snappy |
| Idempotent | true |
| Timeout | 10s |

### Consumer
| Setting | Default |
|---------|---------|
| OffsetReset | earliest |
| EnableAutoCommit | false |
| MaxPollRecords | 100 |
| SessionTimeout | 30s |
| HeartbeatInterval | 10s |
| ProcessingTimeout | 30s |
| MaxRetries | 3 |

## Documentation

See [usage.md](./usage.md) for detailed examples and best practices.

## Dependencies

**Internal:**
- `github.com/Dorico-Dynamics/txova-go-types` - Type definitions
- `github.com/Dorico-Dynamics/txova-go-core` - Core utilities

**External:**
- `github.com/IBM/sarama` - Kafka client

## Development

### Requirements

- Go 1.25+
- Kafka 3.x+

### Testing

```bash
go test -race ./...
```

### Linting

```bash
golangci-lint run ./...
```

### Test Coverage

Current coverage: **91.5%** (target: 80%)

| Package | Coverage |
|---------|----------|
| envelope | 95.8% |
| events | 90.0% |
| topics | 100.0% |
| producer | 87.0% |
| consumer | 91.2% |
| dlq | 86.2% |

## License

Proprietary - Dorico Dynamics
