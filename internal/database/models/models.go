package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// User represents a user in the system
type User struct {
	ID           int       `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// SystemConfig represents a configuration entry
type SystemConfig struct {
	Key         string    `json:"key" db:"key"`
	Value       string    `json:"value" db:"value"`
	Encrypted   bool      `json:"encrypted" db:"encrypted"`
	Description string    `json:"description" db:"description"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Entity represents a Home Assistant entity
type Entity struct {
	EntityID     string          `json:"entity_id" db:"entity_id"`
	FriendlyName sql.NullString  `json:"friendly_name" db:"friendly_name"`
	Domain       string          `json:"domain" db:"domain"`
	State        sql.NullString  `json:"state" db:"state"`
	Attributes   json.RawMessage `json:"attributes" db:"attributes"`
	LastUpdated  time.Time       `json:"last_updated" db:"last_updated"`
	RoomID       sql.NullInt64   `json:"room_id" db:"room_id"`
}

// Room represents a room in the system
type Room struct {
	ID                  int            `json:"id" db:"id"`
	Name                string         `json:"name" db:"name"`
	HomeAssistantAreaID sql.NullString `json:"home_assistant_area_id" db:"home_assistant_area_id"`
	Icon                sql.NullString `json:"icon" db:"icon"`
	Description         sql.NullString `json:"description" db:"description"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
}

// DisplaySetting represents a display configuration
type DisplaySetting struct {
	ID        int             `json:"id" db:"id"`
	Key       string          `json:"key" db:"key"`
	Value     json.RawMessage `json:"value" db:"value"`
	Category  string          `json:"category" db:"category"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// AutomationRule represents an automation rule
type AutomationRule struct {
	ID            string          `json:"id" db:"id"`
	Name          string          `json:"name" db:"name"`
	Description   sql.NullString  `json:"description" db:"description"`
	Enabled       bool            `json:"enabled" db:"enabled"`
	TriggerType   string          `json:"trigger_type" db:"trigger_type"`
	TriggerConfig json.RawMessage `json:"trigger_config" db:"trigger_config"`
	Conditions    json.RawMessage `json:"conditions" db:"conditions"`
	Actions       json.RawMessage `json:"actions" db:"actions"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}
