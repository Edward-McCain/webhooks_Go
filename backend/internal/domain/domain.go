package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventStatus represents the lifecycle state of a webhook event.
type EventStatus string

const (
	EventStatusPending    EventStatus = "PENDING"
	EventStatusProcessing EventStatus = "PROCESSING"
	EventStatusDelivered  EventStatus = "DELIVERED"
	EventStatusRetrying   EventStatus = "RETRYING"
	EventStatusFailed     EventStatus = "FAILED"
)

// DeliveryStatus represents the state of a delivery attempt sequence.
type DeliveryStatus string

const (
	DeliveryStatusPending    DeliveryStatus = "PENDING"
	DeliveryStatusProcessing DeliveryStatus = "PROCESSING"
	DeliveryStatusDelivered  DeliveryStatus = "DELIVERED"
	DeliveryStatusRetrying   DeliveryStatus = "RETRYING"
	DeliveryStatusFailed     DeliveryStatus = "FAILED"
)

// AttemptStatus represents the outcome of a single delivery attempt.
type AttemptStatus string

const (
	AttemptStatusSuccess AttemptStatus = "SUCCESS"
	AttemptStatusFailed  AttemptStatus = "FAILED"
	AttemptStatusTimeout AttemptStatus = "TIMEOUT"
)

// Endpoint is a registered webhook destination.
type Endpoint struct {
	ID        uuid.UUID `json:"id"`
	PublicID  string    `json:"public_id"`
	Name      string    `json:"name"`
	TargetURL string    `json:"target_url"`
	Secret    string    `json:"-"`
	Active    bool      `json:"active"`
	RateLimit int       `json:"rate_limit"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Event is an ingested webhook payload awaiting delivery.
type Event struct {
	ID              uuid.UUID       `json:"id"`
	EndpointID      uuid.UUID       `json:"endpoint_id"`
	ExternalEventID string          `json:"external_event_id"`
	EventType       string          `json:"event_type"`
	Payload         json.RawMessage `json:"payload"`
	Status          EventStatus     `json:"status"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// Delivery tracks delivery progress for an event.
type Delivery struct {
	ID             uuid.UUID      `json:"id"`
	EventID        uuid.UUID      `json:"event_id"`
	EndpointID     uuid.UUID      `json:"endpoint_id"`
	Status         DeliveryStatus `json:"status"`
	AttemptCount   int            `json:"attempt_count"`
	ResponseStatus *int           `json:"response_status,omitempty"`
	ResponseBody   *string        `json:"response_body,omitempty"`
	NextRetryAt    *time.Time     `json:"next_retry_at,omitempty"`
	DeliveredAt    *time.Time     `json:"delivered_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// DeliveryAttempt is a single HTTP delivery try.
type DeliveryAttempt struct {
	ID             uuid.UUID     `json:"id"`
	DeliveryID     uuid.UUID     `json:"delivery_id"`
	AttemptNumber  int           `json:"attempt_number"`
	Status         AttemptStatus `json:"status"`
	ResponseStatus *int          `json:"response_status,omitempty"`
	ResponseBody   *string       `json:"response_body,omitempty"`
	DurationMS     int64         `json:"duration_ms"`
	Error          *string       `json:"error,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
}

// APIKey is a hashed API credential for management API access.
type APIKey struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	KeyHash    string     `json:"-"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// KafkaEventMessage is the lightweight message published to Kafka.
type KafkaEventMessage struct {
	EventID    string `json:"event_id"`
	EndpointID string `json:"endpoint_id"`
	CreatedAt  string `json:"created_at"`
}

// Stats aggregates high-level delivery metrics.
type Stats struct {
	TotalEvents       int64   `json:"total_events"`
	Delivered         int64   `json:"delivered"`
	Failed            int64   `json:"failed"`
	Retrying          int64   `json:"retrying"`
	Pending           int64   `json:"pending"`
	Processing        int64   `json:"processing"`
	SuccessRate       float64 `json:"success_rate"`
	AvgDeliveryTimeMS float64 `json:"avg_delivery_time_ms"`
}
