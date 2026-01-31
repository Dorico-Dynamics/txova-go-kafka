// Package dlq provides dead letter queue functionality for failed Kafka messages.
package dlq

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Dorico-Dynamics/txova-go-kafka/envelope"
)

// Message wraps a failed event with DLQ metadata.
type Message struct {
	// OriginalTopic is the topic where the message was originally published.
	OriginalTopic string `json:"original_topic"`
	// OriginalPartition is the partition where the message was originally published.
	OriginalPartition int32 `json:"original_partition"`
	// OriginalOffset is the offset of the original message.
	OriginalOffset int64 `json:"original_offset"`
	// ErrorMessage describes why the message failed processing.
	ErrorMessage string `json:"error_message"`
	// AttemptCount is the number of processing attempts made.
	AttemptCount int `json:"attempt_count"`
	// FirstFailure is when the message first failed processing.
	FirstFailure time.Time `json:"first_failure"`
	// LastFailure is when the message last failed processing.
	LastFailure time.Time `json:"last_failure"`
	// Envelope is the original event envelope.
	Envelope *envelope.Envelope `json:"envelope"`
}

// NewMessage creates a new DLQ message from a failed event.
// If err is nil, ErrorMessage will be set to an empty string.
func NewMessage(
	originalTopic string,
	originalPartition int32,
	originalOffset int64,
	env *envelope.Envelope,
	err error,
	attemptCount int,
) *Message {
	now := time.Now().UTC()

	errorMessage := ""
	if err != nil {
		errorMessage = err.Error()
	}

	return &Message{
		OriginalTopic:     originalTopic,
		OriginalPartition: originalPartition,
		OriginalOffset:    originalOffset,
		ErrorMessage:      errorMessage,
		AttemptCount:      attemptCount,
		FirstFailure:      now,
		LastFailure:       now,
		Envelope:          env,
	}
}

// MarshalJSON implements json.Marshaler.
func (m *Message) MarshalJSON() ([]byte, error) {
	type alias Message
	data, err := json.Marshal((*alias)(m))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal DLQ message: %w", err)
	}
	return data, nil
}

// UnmarshalJSON implements json.Unmarshaler.
func (m *Message) UnmarshalJSON(data []byte) error {
	type alias Message
	if err := json.Unmarshal(data, (*alias)(m)); err != nil {
		return fmt.Errorf("failed to unmarshal DLQ message: %w", err)
	}
	return nil
}
