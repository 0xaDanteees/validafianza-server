package models

import (
	"net"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Email       string    `json:"email" db:"email"`
	Phone       *string   `json:"phone,omitempty" db:"phone"`
	FullName    *string   `json:"full_name,omitempty" db:"full_name"`
	CompanyName *string   `json:"company_name,omitempty" db:"company_name"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	IsVerified  bool      `json:"is_verified" db:"is_verified"`
	MaxDevices  int       `json:"max_devices" db:"max_devices"`
	Metadata    *string   `json:"metadata,omitempty" db:"metadata"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Passcode struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	Code           string     `json:"-" db:"code"`
	DeliveryMethod string     `json:"delivery_method" db:"delivery_method"`
	DeliveryTarget string     `json:"delivery_target" db:"delivery_target"`
	IsUsed         bool       `json:"is_used" db:"is_used"`
	Attempts       int        `json:"attempts" db:"attempts"`
	MaxAttempts    int        `json:"max_attempts" db:"max_attempts"`
	IPAddress      *net.IP    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent      *string    `json:"user_agent,omitempty" db:"user_agent"`
	ExpiresAt      time.Time  `json:"expires_at" db:"expires_at"`
	UsedAt         *time.Time `json:"used_at,omitempty" db:"used_at"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
}

type DeviceFingerprint struct {
	ID              uuid.UUID `json:"id" db:"id"`
	UserID          uuid.UUID `json:"user_id" db:"user_id"`
	FingerprintHash string    `json:"fingerprint_hash" db:"fingerprint_hash"`
	DeviceName      *string   `json:"device_name,omitempty" db:"device_name"`
	UserAgent       *string   `json:"user_agent,omitempty" db:"user_agent"`
	IPAddress       *net.IP   `json:"ip_address,omitempty" db:"ip_address"`
	Metadata        *string   `json:"metadata,omitempty" db:"metadata"`
	IsTrusted       bool      `json:"is_trusted" db:"is_trusted"`
	LastSeenAt      time.Time `json:"last_seen_at" db:"last_seen_at"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

type Session struct {
	ID                  uuid.UUID  `json:"id" db:"id"`
	UserID              uuid.UUID  `json:"user_id" db:"user_id"`
	DeviceFingerprintID *uuid.UUID `json:"device_fingerprint_id,omitempty" db:"device_fingerprint_id"`
	TokenHash           string     `json:"-" db:"token_hash"`
	IPAddress           *net.IP    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent           *string    `json:"user_agent,omitempty" db:"user_agent"`
	Data                *string    `json:"data,omitempty" db:"data"`
	LastActivityAt      time.Time  `json:"last_activity_at" db:"last_activity_at"`
	ExpiresAt           time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
}

type SessionValidation struct {
	SessionID           uuid.UUID  `db:"session_id"`
	UserID              uuid.UUID  `db:"user_id"`
	UserEmail           string     `db:"user_email"`
	DeviceFingerprintID *uuid.UUID `db:"device_fingerprint_id"`
	IsValid             bool       `db:"is_valid"`
}
