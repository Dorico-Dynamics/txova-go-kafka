# txova-go-kafka Execution Plan

**Version:** 1.0  
**Module:** `github.com/Dorico-Dynamics/txova-go-kafka`  
**Target Test Coverage:** >85%  
**External Dependencies:** `github.com/IBM/sarama`

---

## Phase 1: Project Setup & Foundation

### 1.1 Project Initialization
- [ ] Initialize Go module with `go mod init github.com/Dorico-Dynamics/txova-go-kafka`
- [ ] Add dependencies: `txova-go-types`, `txova-go-core`, `github.com/IBM/sarama`
- [ ] Create directory structure for all packages
- [ ] Set up `.gitignore` for Go projects
- [ ] Configure linting (golangci-lint) matching other txova libraries
- [ ] Create go.mod with proper dependency versions

### 1.2 Package: `envelope` - Event Envelope Standard
- [ ] Define `Envelope` struct with all required fields:
  - `ID` (string, UUID)
  - `Type` (string, event type e.g., "ride.completed")
  - `Version` (string, semver e.g., "1.0")
  - `Source` (string, service name)
  - `Timestamp` (time.Time, UTC)
  - `CorrelationID` (string, request correlation)
  - `CausationID` (string, optional, causing event ID)
  - `Actor` (Actor struct, optional)
  - `Payload` (json.RawMessage)
- [ ] Define `Actor` struct with `ID` (UserID) and `Type` (string)
- [ ] Implement `New()` constructor with validation
- [ ] Implement `NewWithContext()` to extract correlation ID from context
- [ ] Implement `Validate()` method for envelope validation
- [ ] Implement `UnmarshalPayload()` generic method for typed payload extraction
- [ ] Implement JSON marshaling/unmarshaling
- [ ] Implement `Key()` method to extract partition key from payload
- [ ] Write comprehensive tests (>90% coverage)

**Deliverables:**
- [ ] `envelope/` package with full envelope implementation
- [ ] Test suite covering validation, serialization, and edge cases

---

## Phase 2: Event Definitions & Topics

### 2.1 Package: `events` - Event Type Constants
- [ ] Define event type constants for all domains:
  - User events: `UserRegistered`, `UserVerified`, `UserProfileUpdated`, `UserSuspended`, `UserDeleted`
  - Driver events: `DriverApplicationSubmitted`, `DriverDocumentsSubmitted`, `DriverApproved`, `DriverRejected`, `DriverWentOnline`, `DriverWentOffline`, `DriverLocationUpdated`
  - Ride events: `RideRequested`, `RideDriverAssigned`, `RideDriverArrived`, `RideStarted`, `RideCompleted`, `RideCancelled`, `RideRated`
  - Payment events: `PaymentInitiated`, `PaymentCompleted`, `PaymentFailed`, `PaymentRefunded`, `PayoutRequested`, `PayoutCompleted`
  - Safety events: `SafetyEmergencyTriggered`, `SafetyIncidentReported`, `SafetyTripShared`
- [ ] Define event type string constants (e.g., "user.registered")
- [ ] Write tests for event type constants

### 2.2 Package: `events` - Event Payload Structs
- [ ] Implement User event payloads:
  - `UserRegisteredPayload` (UserID, Phone, UserType)
  - `UserVerifiedPayload` (UserID, VerificationType)
  - `UserProfileUpdatedPayload` (UserID, ChangedFields)
  - `UserSuspendedPayload` (UserID, Reason, SuspendedBy)
  - `UserDeletedPayload` (UserID, DeletionType)
- [ ] Implement Driver event payloads:
  - `DriverApplicationSubmittedPayload` (DriverID, UserID)
  - `DriverDocumentsSubmittedPayload` (DriverID, DocumentTypes)
  - `DriverApprovedPayload` (DriverID, ApprovedBy)
  - `DriverRejectedPayload` (DriverID, Reason, RejectedBy)
  - `DriverWentOnlinePayload` (DriverID, Location, VehicleID)
  - `DriverWentOfflinePayload` (DriverID, Reason)
  - `DriverLocationUpdatedPayload` (DriverID, Location, Heading, Speed)
- [ ] Implement Ride event payloads:
  - `RideRequestedPayload` (RideID, RiderID, Pickup, Dropoff, ServiceType)
  - `RideDriverAssignedPayload` (RideID, DriverID, VehicleID, ETAMinutes)
  - `RideDriverArrivedPayload` (RideID, ArrivalTime)
  - `RideStartedPayload` (RideID, StartTime, StartLocation)
  - `RideCompletedPayload` (RideID, EndTime, DistanceKM, DurationMins, Fare)
  - `RideCancelledPayload` (RideID, CancelledBy, Reason, CancellationFee)
  - `RideRatedPayload` (RideID, Rating, RatedBy, Review)
- [ ] Implement Payment event payloads:
  - `PaymentInitiatedPayload` (PaymentID, RideID, Amount, Method)
  - `PaymentCompletedPayload` (PaymentID, TransactionRef)
  - `PaymentFailedPayload` (PaymentID, ErrorCode, ErrorMessage)
  - `PaymentRefundedPayload` (PaymentID, RefundAmount, Reason)
  - `PayoutRequestedPayload` (PayoutID, DriverID, Amount)
  - `PayoutCompletedPayload` (PayoutID, TransactionRef)
- [ ] Implement Safety event payloads:
  - `SafetyEmergencyTriggeredPayload` (RideID, UserID, Location, EmergencyType)
  - `SafetyIncidentReportedPayload` (IncidentID, ReporterID, RideID, Severity)
  - `SafetyTripSharedPayload` (RideID, SharedWithPhone)
- [ ] Write tests for all payload structs

### 2.3 Package: `topics` - Topic Definitions
- [ ] Define topic name constants:
  - `TopicUsers` = "txova.users.v1"
  - `TopicDrivers` = "txova.drivers.v1"
  - `TopicRides` = "txova.rides.v1"
  - `TopicPayments` = "txova.payments.v1"
  - `TopicSafety` = "txova.safety.v1"
  - `TopicLocations` = "txova.locations.v1"
- [ ] Define DLQ topic suffix constant
- [ ] Implement `DLQTopic()` function to generate DLQ topic name
- [ ] Implement `TopicForEventType()` function to map event type to topic
- [ ] Implement `PartitionKey()` helper for each topic
- [ ] Write tests for topic helpers

**Deliverables:**
- [ ] `events/` package with all event types and payloads
- [ ] `topics/` package with topic constants and helpers
- [ ] Test suite for all event payloads and topic mappings

---

## Phase 3: Producer Implementation

### 3.1 Package: `producer` - Configuration
- [ ] Define `Config` struct:
  - `Brokers` ([]string)
  - `Acks` (AckLevel enum: None, Leader, All)
  - `Retries` (int, default 3)
  - `RetryBackoff` (time.Duration, default 100ms)
  - `BatchSize` (int, default 16KB)
  - `LingerMs` (int, default 5)
  - `Compression` (CompressionCodec enum: None, Snappy, Gzip, LZ4, Zstd)
  - `Idempotent` (bool, default true)
  - `MaxMessageBytes` (int, default 1MB)
  - `RequiredAcks` (sarama.RequiredAcks)
- [ ] Implement `DefaultConfig()` with production defaults
- [ ] Implement `Validate()` method for config validation
- [ ] Define `AckLevel` enum (None, Leader, All)
- [ ] Define `CompressionCodec` enum

### 3.2 Package: `producer` - Producer Implementation
- [ ] Implement `Producer` struct wrapping sarama.SyncProducer
- [ ] Implement `New()` constructor with config validation
- [ ] Implement `Publish()` for synchronous publishing:
  - Validate envelope before publishing
  - Inject correlation ID from context if not set
  - Serialize envelope to JSON
  - Send to Kafka with partition key
  - Return error on failure
- [ ] Implement `PublishAsync()` for asynchronous publishing:
  - Same validation as sync
  - Fire-and-forget with callback
  - Handle errors via callback
- [ ] Implement `PublishBatch()` for batch publishing:
  - Accept slice of envelopes
  - Validate all before sending
  - Use sarama batch producer
- [ ] Implement `Close()` for graceful shutdown
- [ ] Implement retry logic with exponential backoff
- [ ] Implement dead letter routing on max retries
- [ ] Add logging for all publish operations
- [ ] Write comprehensive tests with mock Kafka

### 3.3 Package: `producer` - Metrics & Health
- [ ] Implement `HealthCheck()` method (app.HealthChecker interface)
- [ ] Add metrics hooks for observability integration
- [ ] Write tests for health check

**Deliverables:**
- [ ] `producer/` package with full producer implementation
- [ ] Test suite with mocked Kafka interactions (>85% coverage)

---

## Phase 4: Consumer Implementation

### 4.1 Package: `consumer` - Configuration
- [ ] Define `Config` struct:
  - `Brokers` ([]string)
  - `GroupID` (string)
  - `Topics` ([]string)
  - `AutoOffsetReset` (OffsetReset enum: Earliest, Latest)
  - `EnableAutoCommit` (bool, default false)
  - `MaxPollRecords` (int, default 100)
  - `SessionTimeout` (time.Duration, default 30s)
  - `HeartbeatInterval` (time.Duration, default 10s)
  - `MaxRetries` (int, default 3)
  - `RetryBackoff` (time.Duration, default 1s)
  - `ProcessingTimeout` (time.Duration, default 30s)
- [ ] Implement `DefaultConfig()` with production defaults
- [ ] Implement `Validate()` method for config validation
- [ ] Define `OffsetReset` enum (Earliest, Latest)

### 4.2 Package: `consumer` - Handler Pattern
- [ ] Define `Handler` function type: `func(context.Context, envelope.Envelope) error`
- [ ] Define `HandlerRegistry` for routing events to handlers
- [ ] Implement `Handle()` method to register handler for event type
- [ ] Implement `HandlePattern()` for wildcard handlers (e.g., "user.*")
- [ ] Implement handler matching with exact match priority over wildcards
- [ ] Write tests for handler registry

### 4.3 Package: `consumer` - Consumer Implementation
- [ ] Implement `Consumer` struct wrapping sarama.ConsumerGroup
- [ ] Implement `New()` constructor with config validation
- [ ] Implement `Start()` to begin consuming:
  - Join consumer group
  - Poll for messages
  - Deserialize envelope from JSON
  - Route to appropriate handler by event type
  - Handle errors with retry logic
  - Commit offset only after successful handling
  - Send to DLQ after max retries
- [ ] Implement `Stop()` for graceful shutdown:
  - Stop accepting new messages
  - Wait for current batch to complete
  - Commit final offsets
  - Leave consumer group
- [ ] Implement `Close()` for cleanup
- [ ] Implement sarama.ConsumerGroupHandler interface
- [ ] Add context propagation (extract correlation ID, inject into context)
- [ ] Add logging for all consume operations
- [ ] Write comprehensive tests with mock Kafka

### 4.4 Package: `consumer` - Metrics & Health
- [ ] Implement `HealthCheck()` method (app.HealthChecker interface)
- [ ] Add metrics hooks for observability integration
- [ ] Track consumer lag
- [ ] Write tests for health check

**Deliverables:**
- [ ] `consumer/` package with full consumer implementation
- [ ] Test suite with mocked Kafka interactions (>85% coverage)

---

## Phase 5: Dead Letter Queue

### 5.1 Package: `dlq` - DLQ Message Format
- [ ] Define `DLQMessage` struct:
  - `OriginalTopic` (string)
  - `OriginalPartition` (int32)
  - `OriginalOffset` (int64)
  - `OriginalKey` (string)
  - `Envelope` (envelope.Envelope)
  - `ErrorMessage` (string)
  - `AttemptCount` (int)
  - `FirstFailure` (time.Time)
  - `LastFailure` (time.Time)
- [ ] Implement JSON marshaling/unmarshaling
- [ ] Write tests for DLQ message format

### 5.2 Package: `dlq` - DLQ Producer
- [ ] Implement `DLQProducer` struct
- [ ] Implement `Send()` to route failed message to DLQ:
  - Wrap original envelope in DLQ message
  - Add failure metadata
  - Publish to DLQ topic
- [ ] Implement `DLQTopicName()` helper
- [ ] Write tests for DLQ producer

### 5.3 Package: `dlq` - DLQ Replayer
- [ ] Implement `Replayer` struct
- [ ] Implement `Replay()` to replay DLQ messages:
  - Consume from DLQ topic
  - Extract original envelope
  - Call provided handler
  - Commit offset on success
  - Track replay statistics
- [ ] Implement `ReplayWithFilter()` to replay subset of messages
- [ ] Implement graceful shutdown
- [ ] Write tests for replayer

**Deliverables:**
- [ ] `dlq/` package with DLQ handling
- [ ] Test suite for DLQ operations (>85% coverage)

---

## Phase 6: Integration & Quality Assurance

### 6.1 Cross-Package Integration
- [ ] Verify all packages work together without circular dependencies
- [ ] Test full flow: publish → consume → DLQ
- [ ] Ensure consistent error handling patterns across packages
- [ ] Validate context propagation end-to-end

### 6.2 Application Lifecycle Integration
- [ ] Implement `Initializer` interface for producer/consumer
- [ ] Implement `Closer` interface for producer/consumer
- [ ] Write integration tests with txova-go-core app lifecycle

### 6.3 Quality Assurance
- [ ] Run full test suite and verify >85% coverage
- [ ] Run linter and fix all issues
- [ ] Run `go vet` and address all warnings
- [ ] Test with `go build` for all target platforms
- [ ] Verify dependency versions are compatible

### 6.4 Documentation
- [ ] Add package-level documentation (doc.go) for each package
- [ ] Ensure all exported types and functions have godoc comments
- [ ] Update README.md with final usage examples
- [ ] Create CHANGELOG.md with v1.0.0 release notes

### 6.5 Release
- [ ] Tag release as v1.0.0
- [ ] Push tag to GitHub
- [ ] Verify module is accessible via `go get`

**Deliverables:**
- [ ] Complete, tested library
- [ ] v1.0.0 release tagged and published
- [ ] >85% test coverage verified

---

## Success Criteria

| Criteria | Target | Current |
| -------- | ------ | ------- |
| Test Coverage | >85% | - |
| External Dependencies | 1 (sarama) | - |
| Linting Errors | 0 | - |
| `go vet` Warnings | 0 | - |
| Publish Latency P99 | <50ms | - |
| Consumer Lag | <1000 messages | - |
| DLQ Rate | <0.1% | - |
| Message Ordering | 100% per partition | - |

---

## Package Dependency Order

```text
envelope (depends on: txova-go-types/ids, txova-go-core/context)
    ↓
events (depends on: txova-go-types/*, envelope)
    ↓
topics (depends on: events)
    ↓
producer (depends on: envelope, topics, txova-go-core/*, sarama)
    ↓
consumer (depends on: envelope, topics, txova-go-core/*, sarama)
    ↓
dlq (depends on: envelope, producer, consumer)
```

---

## Risk Mitigation

| Risk | Mitigation |
| ---- | ---------- |
| Kafka connection failures | Implement retry with exponential backoff, circuit breaker pattern |
| Message ordering issues | Partition by entity ID, document ordering guarantees |
| Consumer lag buildup | Monitor lag metrics, auto-scale consumers, alert on threshold |
| DLQ overflow | Implement DLQ retention policy, replay tooling, alerting |
| Schema evolution breaks | Document versioning strategy, support multiple versions during migration |
| Memory pressure from batching | Configure max batch size, implement backpressure |
| Duplicate message processing | Require idempotent handlers, document idempotency requirements |
| Context propagation loss | Test correlation ID flow end-to-end, add logging |

---

## Testing Strategy

### Unit Tests
- Mock sarama interfaces for producer/consumer tests
- Test all envelope validation scenarios
- Test all event payload serialization
- Test handler routing with wildcards

### Integration Tests (Optional - requires Kafka)
- Full publish/consume flow
- Consumer group rebalancing
- DLQ routing and replay
- Graceful shutdown behavior

### Test Utilities
- Provide mock producer/consumer for downstream service tests
- Provide test envelope builders
- Provide in-memory event store for testing
