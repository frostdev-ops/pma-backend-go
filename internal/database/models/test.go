package models

import (
	"time"
)

// TestConfiguration represents test configuration stored in database
type TestConfiguration struct {
	ID        int       `json:"id" db:"id"`
	Key       string    `json:"key" db:"key"`
	Value     string    `json:"value" db:"value"`
	Category  string    `json:"category" db:"category"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// MockEntityRecord represents a mock entity stored in database (optional persistence)
type MockEntityRecord struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Domain      string    `json:"domain" db:"domain"`
	State       string    `json:"state" db:"state"`
	Attributes  string    `json:"attributes" db:"attributes"` // JSON string
	RoomID      *int      `json:"room_id" db:"room_id"`
	LastChanged time.Time `json:"last_changed" db:"last_changed"`
	LastUpdated time.Time `json:"last_updated" db:"last_updated"`
	EntityType  string    `json:"entity_type" db:"entity_type"`
	DeviceClass *string   `json:"device_class" db:"device_class"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// TestSession represents a test session for tracking test operations
type TestSession struct {
	ID          string     `json:"id" db:"id"`
	SessionType string     `json:"session_type" db:"session_type"` // mock_entities, performance_test, etc.
	Status      string     `json:"status" db:"status"`             // active, completed, failed
	StartedAt   time.Time  `json:"started_at" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at" db:"completed_at"`
	Results     string     `json:"results" db:"results"` // JSON string
	CreatedBy   string     `json:"created_by" db:"created_by"`
}
