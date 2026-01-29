# txova-go-kafka

## Overview
Event streaming library providing Kafka producer/consumer implementations, event envelope standards, and typed event definitions for asynchronous communication between services.

**Module:** `github.com/txova/txova-go-kafka`

---

## Packages

### `envelope` - Event Envelope Standard

Every event published to Kafka must use the standard envelope format.

**Envelope Fields:**
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| id | string | Yes | Unique event ID (UUID) |
| type | string | Yes | Event type (e.g., "user.registered") |
| version | string | Yes | Schema version (e.g., "1.0") |
| source | string | Yes | Producing service name |
| timestamp | ISO8601 | Yes | Event creation time |
| correlation_id | string | Yes | Request correlation ID |
| causation_id | string | No | ID of event that caused this |
| actor | object | No | Who triggered the event |
| actor.id | UserID | No | Actor's user ID |
| actor.type | string | No | Actor type (user, system, admin) |
| payload | object | Yes | Event-specific data |

**Requirements:**
- All fields except causation_id and actor are required
- Envelope must be validated before publishing
- Version should follow semver (major.minor)
- Timestamp must be UTC

---

### `events` - Event Type Definitions

#### User Events
| Event Type | Description | Key Fields |
|------------|-------------|------------|
| user.registered | New user created | user_id, phone, user_type |
| user.verified | User completed verification | user_id, verification_type |
| user.profile_updated | Profile information changed | user_id, changed_fields[] |
| user.suspended | User account suspended | user_id, reason, suspended_by |
| user.deleted | User account deleted | user_id, deletion_type |

#### Driver Events
| Event Type | Description | Key Fields |
|------------|-------------|------------|
| driver.application_submitted | Driver applied | driver_id, user_id |
| driver.documents_submitted | Documents uploaded | driver_id, document_types[] |
| driver.approved | Driver application approved | driver_id, approved_by |
| driver.rejected | Driver application rejected | driver_id, reason, rejected_by |
| driver.went_online | Driver became available | driver_id, location, vehicle_id |
| driver.went_offline | Driver became unavailable | driver_id, reason |
| driver.location_updated | Driver position changed | driver_id, location, heading, speed |

#### Ride Events
| Event Type | Description | Key Fields |
|------------|-------------|------------|
| ride.requested | Rider requested ride | ride_id, rider_id, pickup, dropoff, service_type |
| ride.driver_assigned | Driver matched to ride | ride_id, driver_id, vehicle_id, eta_minutes |
| ride.driver_arrived | Driver at pickup | ride_id, arrival_time |
| ride.started | Trip began | ride_id, start_time, start_location |
| ride.completed | Trip finished | ride_id, end_time, distance_km, duration_mins, fare |
| ride.cancelled | Trip cancelled | ride_id, cancelled_by, reason, cancellation_fee |
| ride.rated | Ride was rated | ride_id, rating, rated_by, review |

#### Payment Events
| Event Type | Description | Key Fields |
|------------|-------------|------------|
| payment.initiated | Payment started | payment_id, ride_id, amount, method |
| payment.completed | Payment successful | payment_id, transaction_ref |
| payment.failed | Payment failed | payment_id, error_code, error_message |
| payment.refunded | Payment refunded | payment_id, refund_amount, reason |
| payout.requested | Driver payout requested | payout_id, driver_id, amount |
| payout.completed | Driver payout sent | payout_id, transaction_ref |

#### Safety Events
| Event Type | Description | Key Fields |
|------------|-------------|------------|
| safety.emergency_triggered | SOS activated | ride_id, user_id, location, emergency_type |
| safety.incident_reported | Incident reported | incident_id, reporter_id, ride_id, severity |
| safety.trip_shared | Trip shared with contact | ride_id, shared_with_phone |

---

### `topics` - Topic Definitions

**Topic Naming Convention:** `txova.{domain}.{version}`

| Topic | Partition Key | Consumers |
|-------|---------------|-----------|
| txova.users.v1 | user_id | Driver, Notification, Safety |
| txova.drivers.v1 | driver_id | Ride, Notification, Operations |
| txova.rides.v1 | ride_id | Payment, Safety, Notification, Analytics |
| txova.payments.v1 | payment_id | Ride, Driver, Notification |
| txova.safety.v1 | incident_id | Notification, Operations |
| txova.locations.v1 | driver_id | Ride, Safety |

**Requirements:**
- Partition by entity ID for ordering guarantees
- Retain messages for 7 days
- Replication factor: 3 (production)

---

### `producer` - Event Publishing

| Feature | Priority | Description |
|---------|----------|-------------|
| Sync publish | P0 | Wait for broker acknowledgment |
| Async publish | P1 | Fire-and-forget with callback |
| Batch publish | P1 | Publish multiple events efficiently |
| Retry logic | P0 | Retry on transient failures |
| Dead letter | P0 | Route failed messages to DLQ |

**Producer Configuration:**
| Setting | Default | Description |
|---------|---------|-------------|
| acks | all | Wait for all replicas |
| retries | 3 | Retry attempts |
| retry_backoff | 100ms | Backoff between retries |
| batch_size | 16KB | Batch size for async |
| linger_ms | 5 | Wait time before sending batch |
| compression | snappy | Compression codec |

**Requirements:**
- Validate envelope before publishing
- Inject correlation_id from context if not set
- Log all publish failures
- Support idempotent producer

---

### `consumer` - Event Consumption

| Feature | Priority | Description |
|---------|----------|-------------|
| Consumer group | P0 | Coordinated consumption |
| Manual commit | P0 | Commit after processing |
| Auto offset reset | P0 | Configurable (earliest/latest) |
| Error handling | P0 | Handler for processing failures |
| Graceful shutdown | P0 | Finish current batch before exit |

**Consumer Configuration:**
| Setting | Default | Description |
|---------|---------|-------------|
| group_id | {service}-group | Consumer group name |
| auto_offset_reset | earliest | Start from beginning |
| enable_auto_commit | false | Manual commit |
| max_poll_records | 100 | Records per poll |
| session_timeout | 30s | Consumer timeout |
| heartbeat_interval | 10s | Heartbeat frequency |

**Handler Pattern:**
| Requirement | Description |
|-------------|-------------|
| Typed handlers | Handler per event type with typed payload |
| Error returns | Return error to trigger retry/DLQ |
| Idempotency | Handlers must be idempotent |
| Timeout | Handlers have processing timeout |

**Requirements:**
- Route events to handlers by event type
- Support wildcard handlers (e.g., "user.*")
- Commit offset only after successful handling
- Send to DLQ after max retries (3)

---

### `dlq` - Dead Letter Queue

| Feature | Priority | Description |
|---------|----------|-------------|
| DLQ routing | P0 | Route failed events to DLQ topic |
| Failure metadata | P0 | Include error, attempts, timestamps |
| Replay support | P1 | Tooling to replay DLQ events |

**DLQ Topic Naming:** `txova.{original-topic}.dlq`

**DLQ Message Additions:**
| Field | Description |
|-------|-------------|
| original_topic | Source topic |
| original_partition | Source partition |
| original_offset | Source offset |
| error_message | Processing error |
| attempt_count | Number of attempts |
| first_failure | First failure timestamp |
| last_failure | Last failure timestamp |

---

## Consumer Group Naming

| Service | Group ID |
|---------|----------|
| user-service | user-service-group |
| driver-service | driver-service-group |
| ride-service | ride-service-group |
| payment-service | payment-service-group |
| notification-service | notification-service-group |
| safety-service | safety-service-group |
| operations-service | operations-service-group |

---

## Event Versioning Strategy

| Scenario | Approach |
|----------|----------|
| Add optional field | Same version, backwards compatible |
| Add required field | New version, deprecate old |
| Remove field | New version, deprecate old |
| Change field type | New version, never change |

**Requirements:**
- Consumers must ignore unknown fields
- Producers must support publishing multiple versions during migration
- Version deprecation period: 30 days minimum

---

## Dependencies

**Internal:**
- `txova-go-types`
- `txova-go-core`

**External:**
- `github.com/IBM/sarama` — Kafka client

---

## Success Metrics
| Metric | Target |
|--------|--------|
| Test coverage | > 85% |
| Publish latency P99 | < 50ms |
| Consumer lag | < 1000 messages |
| DLQ rate | < 0.1% |
| Message ordering | 100% per partition |
