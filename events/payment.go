package events

import (
	"fmt"

	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
)

// PaymentInitiated is the payload for payment.initiated events.
type PaymentInitiated struct {
	PaymentID ids.PaymentID       `json:"payment_id"`
	RideID    ids.RideID          `json:"ride_id"`
	Amount    int64               `json:"amount"`
	Method    enums.PaymentMethod `json:"method"`
}

// PaymentCompleted is the payload for payment.completed events.
type PaymentCompleted struct {
	PaymentID      ids.PaymentID `json:"payment_id"`
	TransactionRef string        `json:"transaction_ref"`
}

// PaymentFailed is the payload for payment.failed events.
type PaymentFailed struct {
	PaymentID    ids.PaymentID `json:"payment_id"`
	ErrorCode    string        `json:"error_code"`
	ErrorMessage string        `json:"error_message"`
}

// PaymentRefunded is the payload for payment.refunded events.
type PaymentRefunded struct {
	PaymentID    ids.PaymentID `json:"payment_id"`
	RefundAmount int64         `json:"refund_amount"`
	Reason       string        `json:"reason"`
}

// PayoutID is a typed identifier for payouts.
type PayoutID struct {
	ids.UUID
}

// NewPayoutID creates a new random PayoutID.
func NewPayoutID() (PayoutID, error) {
	uuid, err := ids.NewUUID()
	if err != nil {
		return PayoutID{}, fmt.Errorf("failed to generate payout ID: %w", err)
	}
	return PayoutID{UUID: uuid}, nil
}

// MustNewPayoutID creates a new PayoutID or panics.
func MustNewPayoutID() PayoutID {
	id, err := NewPayoutID()
	if err != nil {
		panic(err)
	}
	return id
}

// PayoutRequested is the payload for payout.requested events.
type PayoutRequested struct {
	PayoutID PayoutID     `json:"payout_id"`
	DriverID ids.DriverID `json:"driver_id"`
	Amount   int64        `json:"amount"`
}

// PayoutCompleted is the payload for payout.completed events.
type PayoutCompleted struct {
	PayoutID       PayoutID `json:"payout_id"`
	TransactionRef string   `json:"transaction_ref"`
}
