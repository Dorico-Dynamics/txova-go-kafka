# txova-go-kafka Usage Guide

This guide provides detailed examples for using the txova-go-kafka library.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Envelope Package](#envelope-package)
- [Producer Package](#producer-package)
- [Consumer Package](#consumer-package)
- [DLQ Package](#dlq-package)
- [Events Package](#events-package)
- [Topics Package](#topics-package)
- [Best Practices](#best-practices)

## Installation

```bash
go get github.com/Dorico-Dynamics/txova-go-kafka
```

## Quick Start

### Publishing an Event

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "time"

    "github.com/Dorico-Dynamics/txova-go-kafka/envelope"
    "github.com/Dorico-Dynamics/txova-go-kafka/events"
    "github.com/Dorico-Dynamics/txova-go-kafka/producer"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    ctx := context.Background()

    // Create producer configuration
    cfg := producer.DefaultConfig()
    cfg.Brokers = []string{"localhost:9092"}
    cfg.ClientID = "my-service"

    // Create producer
    prod, err := producer.New(cfg, logger)
    if err != nil {
        logger.Error("failed to create producer", slog.String("error", err.Error()))
        return
    }
    defer prod.Close()

    // Create event envelope
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
        logger.Error("failed to create envelope", slog.String("error", err.Error()))
        return
    }

    // Publish event (auto-routes to appropriate topic)
    if err := prod.Publish(ctx, env, rideID.String()); err != nil {
        logger.Error("failed to publish", slog.String("error", err.Error()))
        return
    }

    logger.Info("event published successfully")
}
```

### Consuming Events

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"

    "github.com/Dorico-Dynamics/txova-go-kafka/consumer"
    "github.com/Dorico-Dynamics/txova-go-kafka/envelope"
    "github.com/Dorico-Dynamics/txova-go-kafka/events"
    "github.com/Dorico-Dynamics/txova-go-kafka/topics"
)

func main() {
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

    // Create handler registry
    registry := consumer.NewHandlerRegistry()

    // Register event handlers
    registry.Register(events.EventTypeRideCompleted, func(ctx context.Context, env *envelope.Envelope) error {
        var payload events.RideCompleted
        if err := env.UnmarshalPayload(&payload); err != nil {
            return err
        }

        logger.InfoContext(ctx, "ride completed",
            slog.String("ride_id", payload.RideID.String()),
            slog.Float64("distance_km", payload.DistanceKM),
            slog.Int64("fare", payload.Fare),
        )

        return nil
    })

    // Create consumer configuration
    cfg := consumer.DefaultConfig()
    cfg.Brokers = []string{"localhost:9092"}
    cfg.GroupID = topics.ConsumerGroupRideService.String()
    cfg.Topics = []string{topics.Rides.String()}

    // Create consumer
    cons, err := consumer.New(cfg, registry, logger)
    if err != nil {
        logger.Error("failed to create consumer", slog.String("error", err.Error()))
        return
    }

    // Set up graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigCh
        logger.Info("shutting down...")
        cancel()
    }()

    // Start consuming (blocks until context is cancelled)
    if err := cons.Start(ctx); err != nil && err != context.Canceled {
        logger.Error("consumer error", slog.String("error", err.Error()))
    }

    if err := cons.Close(); err != nil {
        logger.Error("failed to close consumer", slog.String("error", err.Error()))
    }
}
```

## Envelope Package

The envelope package provides the standard event format for all Kafka messages.

### Creating Envelopes

```go
import "github.com/Dorico-Dynamics/txova-go-kafka/envelope"

// Basic envelope creation
env, err := envelope.New(&envelope.Config{
    Type:          "user.registered",
    Version:       "1.0",
    Source:        "user-service",
    CorrelationID: "req-123",
    Payload: map[string]any{
        "user_id": "user-456",
        "email":   "user@example.com",
    },
})

// Create with context (extracts correlation ID automatically)
env, err := envelope.NewWithContext(ctx, &envelope.Config{
    Type:    "user.registered",
    Version: "1.0",
    Source:  "user-service",
    Payload: payload,
})

// With optional actor
env, err := envelope.New(&envelope.Config{
    Type:    "user.suspended",
    Version: "1.0",
    Source:  "admin-service",
    Actor: &envelope.Actor{
        ID:   adminUserID,
        Type: envelope.ActorTypeAdmin,
    },
    Payload: payload,
})
```

### Envelope Methods

```go
// Validate envelope
if err := env.Validate(); err != nil {
    return err
}

// Unmarshal payload
var payload events.RideCompleted
if err := env.UnmarshalPayload(&payload); err != nil {
    return err
}

// Extract event domain and action
domain := env.EventDomain()  // "ride" from "ride.completed"
action := env.EventAction()  // "completed" from "ride.completed"

// Builder methods (return new copies)
envWithCorrelation := env.WithCorrelationID("corr-789")
envWithCausation := env.WithCausationID("cause-123")
envWithActor := env.WithActor(&envelope.Actor{
    ID:   userID,
    Type: envelope.ActorTypeUser,
})
```

### Actor Types

```go
envelope.ActorTypeUser   // Regular user action
envelope.ActorTypeSystem // Automated system action
envelope.ActorTypeAdmin  // Administrator action
```

## Producer Package

### Configuration

```go
import "github.com/Dorico-Dynamics/txova-go-kafka/producer"

// Default configuration (recommended for production)
cfg := producer.DefaultConfig()
cfg.Brokers = []string{"kafka1:9092", "kafka2:9092", "kafka3:9092"}
cfg.ClientID = "my-service"

// Configuration options
cfg.RequiredAcks = sarama.WaitForAll  // Default: WaitForAll
cfg.MaxRetries = 3                     // Default: 3
cfg.RetryBackoff = 100 * time.Millisecond  // Default: 100ms
cfg.BatchSize = 16 * 1024              // Default: 16KB
cfg.Linger = 5 * time.Millisecond      // Default: 5ms
cfg.Compression = sarama.CompressionSnappy  // Default: Snappy
cfg.Idempotent = true                  // Default: true
cfg.Timeout = 10 * time.Second         // Default: 10s
```

### Publishing Methods

```go
// Create producer
prod, err := producer.New(cfg, logger)
if err != nil {
    return err
}
defer prod.Close()

// Auto-route to topic based on event type
err = prod.Publish(ctx, env, partitionKey)

// Publish to specific topic
err = prod.PublishToTopic(ctx, topics.Rides, env, partitionKey)

// Batch publish
envs := []*envelope.Envelope{env1, env2, env3}
keys := []string{"key1", "key2", "key3"}
err = prod.PublishBatch(ctx, envs, keys)
```

### Error Handling

```go
switch {
case errors.Is(err, producer.ErrProducerClosed):
    // Producer was closed
case errors.Is(err, producer.ErrNilEnvelope):
    // Envelope is nil
case errors.Is(err, producer.ErrInvalidEnvelope):
    // Envelope validation failed
case errors.Is(err, producer.ErrNoTopicForEventType):
    // Unknown event type
case errors.Is(err, producer.ErrBatchLengthMismatch):
    // Envelope and key arrays have different lengths
}
```

## Consumer Package

### Configuration

```go
import "github.com/Dorico-Dynamics/txova-go-kafka/consumer"

cfg := consumer.DefaultConfig()
cfg.Brokers = []string{"kafka1:9092", "kafka2:9092"}
cfg.GroupID = "payment-service-group"
cfg.Topics = []string{"txova.rides.v1", "txova.payments.v1"}

// Configuration options
cfg.OffsetReset = consumer.OffsetResetEarliest  // Default: earliest
cfg.EnableAutoCommit = false                     // Default: false (manual commit recommended)
cfg.MaxPollRecords = 100                         // Default: 100
cfg.SessionTimeout = 30 * time.Second            // Default: 30s
cfg.HeartbeatInterval = 10 * time.Second         // Default: 10s
cfg.ProcessingTimeout = 30 * time.Second         // Default: 30s
cfg.MaxRetries = 3                               // Default: 3
```

### Handler Registry

```go
registry := consumer.NewHandlerRegistry()

// Register handlers for specific event types
registry.Register(events.EventTypeRideCompleted, handleRideCompleted)
registry.Register(events.EventTypeRideCancelled, handleRideCancelled)

// Register wildcard handler (catches unregistered event types)
registry.RegisterWildcard(func(ctx context.Context, env *envelope.Envelope) error {
    logger.WarnContext(ctx, "unhandled event type", slog.String("type", env.Type))
    return nil  // Return nil to acknowledge, error to retry
})

// Check if handler exists
if registry.Has(events.EventTypeRideCompleted) {
    // Handler registered
}

// Get all registered event types
types := registry.RegisteredTypes()
```

### DLQ Integration

```go
import "github.com/Dorico-Dynamics/txova-go-kafka/dlq"

// Create DLQ handler
dlqHandler := dlq.NewHandler(syncProducer, logger)

// Set up consumer with DLQ
cons, err := consumer.New(cfg, registry, logger)
if err != nil {
    return err
}

// Assign DLQ handler (called after MaxRetries failures)
cons.DLQHandler = func(ctx context.Context, env *envelope.Envelope, err error) {
    if sendErr := dlqHandler.Handle(ctx, "txova.rides.v1", 0, 0, env, err, cfg.MaxRetries); sendErr != nil {
        logger.ErrorContext(ctx, "failed to send to DLQ", slog.String("error", sendErr.Error()))
    }
}
```

## DLQ Package

### DLQ Message Structure

```go
import "github.com/Dorico-Dynamics/txova-go-kafka/dlq"

// DLQ messages contain metadata about the failure
type Message struct {
    OriginalTopic     string             // Where the message was published
    OriginalPartition int32              // Original partition
    OriginalOffset    int64              // Original offset
    ErrorMessage      string             // Why processing failed
    AttemptCount      int                // Number of attempts made
    FirstFailure      time.Time          // First failure timestamp
    LastFailure       time.Time          // Latest failure timestamp
    Envelope          *envelope.Envelope // Original event
}
```

### Processing DLQ Messages

```go
// Create consumer for DLQ topic
dlqCfg := consumer.DefaultConfig()
dlqCfg.Brokers = brokers
dlqCfg.GroupID = "ride-service-dlq-processor"
dlqCfg.Topics = []string{"txova.rides.v1.dlq"}

dlqRegistry := consumer.NewHandlerRegistry()
dlqRegistry.RegisterWildcard(func(ctx context.Context, env *envelope.Envelope) error {
    var msg dlq.Message
    if err := env.UnmarshalPayload(&msg); err != nil {
        return err
    }

    logger.InfoContext(ctx, "processing DLQ message",
        slog.String("original_topic", msg.OriginalTopic),
        slog.String("error", msg.ErrorMessage),
        slog.Int("attempts", msg.AttemptCount),
        slog.Time("first_failure", msg.FirstFailure),
    )

    // Decide action: retry, alert, or discard
    if msg.AttemptCount > 10 {
        // Alert operations team
        alertOperations(ctx, &msg)
        return nil  // Acknowledge to remove from DLQ
    }

    // Attempt reprocessing
    return reprocessEvent(ctx, msg.Envelope)
})

dlqCons, _ := consumer.New(dlqCfg, dlqRegistry, logger)
dlqCons.Start(ctx)
```

## Events Package

### Event Types

```go
import "github.com/Dorico-Dynamics/txova-go-kafka/events"

// User events
events.EventTypeUserRegistered      // "user.registered"
events.EventTypeUserVerified        // "user.verified"
events.EventTypeUserProfileUpdated  // "user.profile_updated"
events.EventTypeUserSuspended       // "user.suspended"
events.EventTypeUserDeleted         // "user.deleted"

// Driver events
events.EventTypeDriverApplicationSubmitted  // "driver.application_submitted"
events.EventTypeDriverDocumentsSubmitted    // "driver.documents_submitted"
events.EventTypeDriverApproved              // "driver.approved"
events.EventTypeDriverRejected              // "driver.rejected"
events.EventTypeDriverWentOnline            // "driver.went_online"
events.EventTypeDriverWentOffline           // "driver.went_offline"
events.EventTypeDriverLocationUpdated       // "driver.location_updated"

// Ride events
events.EventTypeRideRequested       // "ride.requested"
events.EventTypeRideDriverAssigned  // "ride.driver_assigned"
events.EventTypeRideDriverArrived   // "ride.driver_arrived"
events.EventTypeRideStarted         // "ride.started"
events.EventTypeRideCompleted       // "ride.completed"
events.EventTypeRideCancelled       // "ride.cancelled"
events.EventTypeRideRated           // "ride.rated"

// Payment events
events.EventTypePaymentInitiated  // "payment.initiated"
events.EventTypePaymentCompleted  // "payment.completed"
events.EventTypePaymentFailed     // "payment.failed"
events.EventTypePaymentRefunded   // "payment.refunded"
events.EventTypePayoutRequested   // "payout.requested"
events.EventTypePayoutCompleted   // "payout.completed"

// Safety events
events.EventTypeSafetyEmergencyTriggered  // "safety.emergency_triggered"
events.EventTypeSafetyIncidentReported    // "safety.incident_reported"
events.EventTypeSafetyTripShared          // "safety.trip_shared"
```

### Event Payloads

```go
// User registered event
payload := &events.UserRegistered{
    UserID:   userID,
    Phone:    phone,
    UserType: enums.UserTypeRider,
    Email:    &email,  // Optional
    Name:     "John Doe",
}

// Ride completed event
payload := &events.RideCompleted{
    RideID:       rideID,
    EndTime:      time.Now(),
    DistanceKM:   12.5,
    DurationMins: 25,
    Fare:         5000,  // In smallest currency unit (cents)
}

// Driver location updated event
payload := &events.DriverLocationUpdated{
    DriverID: driverID,
    Location: geo.Location{Lat: 40.7128, Lng: -74.0060},
    Heading:  90.0,   // Degrees
    Speed:    45.5,   // km/h
}
```

### EventType Methods

```go
eventType := events.EventTypeRideCompleted

// String representation
str := eventType.String()  // "ride.completed"

// Extract domain and action
domain := eventType.Domain()  // "ride"
action := eventType.Action()  // "completed"

// Validate event type
if eventType.Valid() {
    // Known event type
}
```

## Topics Package

### Topic Constants

```go
import "github.com/Dorico-Dynamics/txova-go-kafka/topics"

topics.Users      // "txova.users.v1"
topics.Drivers    // "txova.drivers.v1"
topics.Rides      // "txova.rides.v1"
topics.Payments   // "txova.payments.v1"
topics.Safety     // "txova.safety.v1"
topics.Locations  // "txova.locations.v1"
```

### Topic Methods

```go
topic := topics.Rides

// Get DLQ topic
dlqTopic := topic.DLQ()  // "txova.rides.v1.dlq"

// Get partition key field
keyField := topic.PartitionKey()  // "ride_id"

// Check if valid topic
if topic.Valid() {
    // Known topic
}

// Get all topics
allTopics := topics.All()
```

### Event to Topic Routing

```go
// Auto-route event type to topic
topic := topics.ForEventType(events.EventTypeRideCompleted)  // topics.Rides

// Routing rules:
// - user.* -> topics.Users
// - driver.* -> topics.Drivers
// - ride.* -> topics.Rides
// - payment.*, payout.* -> topics.Payments
// - safety.* -> topics.Safety
```

### Consumer Groups

```go
topics.ConsumerGroupUserService          // "user-service-group"
topics.ConsumerGroupDriverService        // "driver-service-group"
topics.ConsumerGroupRideService          // "ride-service-group"
topics.ConsumerGroupPaymentService       // "payment-service-group"
topics.ConsumerGroupNotificationService  // "notification-service-group"
topics.ConsumerGroupSafetyService        // "safety-service-group"
topics.ConsumerGroupOperationsService    // "operations-service-group"
topics.ConsumerGroupAnalyticsService     // "analytics-service-group"

// Generate custom group ID
customGroup := topics.ForService("my-service")  // "my-service-group"
```

### Topic Configuration

```go
// Production configuration
prodConfig := topics.DefaultConfig()
// NumPartitions: 12
// ReplicationFactor: 3
// RetentionMs: 7 days

// Development configuration
devConfig := topics.DevConfig()
// NumPartitions: 3
// ReplicationFactor: 1
// RetentionMs: 1 day
```

## Best Practices

### 1. Use Correlation IDs

Always propagate correlation IDs for distributed tracing:

```go
// Use NewWithContext to auto-extract from context
env, err := envelope.NewWithContext(ctx, &envelope.Config{
    Type:    eventType,
    Version: events.VersionLatest,
    Source:  "my-service",
    Payload: payload,
})
```

### 2. Use Appropriate Partition Keys

Use entity IDs as partition keys to ensure ordering:

```go
// Ride events partitioned by ride_id
prod.Publish(ctx, rideEnv, rideID.String())

// User events partitioned by user_id
prod.Publish(ctx, userEnv, userID.String())
```

### 3. Handle Errors Gracefully

Return errors from handlers to trigger retry logic:

```go
registry.Register(events.EventTypeRideCompleted, func(ctx context.Context, env *envelope.Envelope) error {
    var payload events.RideCompleted
    if err := env.UnmarshalPayload(&payload); err != nil {
        // Permanent error - don't retry
        logger.ErrorContext(ctx, "invalid payload", slog.String("error", err.Error()))
        return nil
    }

    if err := processRide(ctx, &payload); err != nil {
        // Transient error - retry
        return err
    }

    return nil
})
```

### 4. Implement Idempotent Handlers

Handlers may be called multiple times for the same message:

```go
registry.Register(events.EventTypePaymentCompleted, func(ctx context.Context, env *envelope.Envelope) error {
    var payload events.PaymentCompleted
    if err := env.UnmarshalPayload(&payload); err != nil {
        return nil
    }

    // Check if already processed
    if alreadyProcessed(ctx, env.ID) {
        return nil
    }

    // Process and mark as processed atomically
    return processPaymentIdempotent(ctx, env.ID, &payload)
})
```

### 5. Use Structured Logging

Use slog for consistent structured logging:

```go
logger.InfoContext(ctx, "processing event",
    slog.String("event_id", env.ID),
    slog.String("event_type", env.Type),
    slog.String("correlation_id", env.CorrelationID),
)
```

### 6. Graceful Shutdown

Always handle graceful shutdown:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

go func() {
    <-sigCh
    cancel()
}()

// Consumer will stop when context is cancelled
cons.Start(ctx)
cons.Close()
```
