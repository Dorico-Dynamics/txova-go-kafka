package events

import (
	"github.com/Dorico-Dynamics/txova-go-types/contact"
	"github.com/Dorico-Dynamics/txova-go-types/enums"
	"github.com/Dorico-Dynamics/txova-go-types/ids"
)

// UserRegistered is the payload for user.registered events.
type UserRegistered struct {
	UserID   ids.UserID          `json:"user_id"`
	Phone    contact.PhoneNumber `json:"phone"`
	UserType enums.UserType      `json:"user_type"`
	Email    *contact.Email      `json:"email,omitempty"`
	Name     string              `json:"name,omitempty"`
}

// VerificationType represents the type of verification completed.
type VerificationType string

const (
	VerificationTypePhone VerificationType = "phone"
	VerificationTypeEmail VerificationType = "email"
	VerificationTypeID    VerificationType = "id"
)

// UserVerified is the payload for user.verified events.
type UserVerified struct {
	UserID           ids.UserID       `json:"user_id"`
	VerificationType VerificationType `json:"verification_type"`
}

// UserProfileUpdated is the payload for user.profile_updated events.
type UserProfileUpdated struct {
	UserID        ids.UserID `json:"user_id"`
	ChangedFields []string   `json:"changed_fields"`
}

// UserSuspended is the payload for user.suspended events.
type UserSuspended struct {
	UserID      ids.UserID  `json:"user_id"`
	Reason      string      `json:"reason"`
	SuspendedBy *ids.UserID `json:"suspended_by,omitempty"`
}

// DeletionType represents the type of account deletion.
type DeletionType string

const (
	DeletionTypeUserRequested DeletionType = "user_requested"
	DeletionTypeAdminAction   DeletionType = "admin_action"
	DeletionTypeInactivity    DeletionType = "inactivity"
)

// UserDeleted is the payload for user.deleted events.
type UserDeleted struct {
	UserID       ids.UserID   `json:"user_id"`
	DeletionType DeletionType `json:"deletion_type"`
}
