package types

import (
	"time"
)

// PMASourceType represents the source of an entity
type PMASourceType string

const (
	SourceHomeAssistant PMASourceType = "homeassistant"
	SourceRing          PMASourceType = "ring"
	SourceShelly        PMASourceType = "shelly"
	SourceUPS           PMASourceType = "ups"
	SourceNetwork       PMASourceType = "network"
	SourcePMA           PMASourceType = "pma"
)

// PMAEntityType represents the type/domain of an entity
type PMAEntityType string

const (
	EntityTypeLight        PMAEntityType = "light"
	EntityTypeSwitch       PMAEntityType = "switch"
	EntityTypeSensor       PMAEntityType = "sensor"
	EntityTypeClimate      PMAEntityType = "climate"
	EntityTypeCover        PMAEntityType = "cover"
	EntityTypeCamera       PMAEntityType = "camera"
	EntityTypeLock         PMAEntityType = "lock"
	EntityTypeFan          PMAEntityType = "fan"
	EntityTypeMediaPlayer  PMAEntityType = "media_player"
	EntityTypeBinarySensor PMAEntityType = "binary_sensor"
	EntityTypeDevice       PMAEntityType = "device"
	EntityTypeGeneric      PMAEntityType = "generic"
)

// PMAEntityState represents the possible states of an entity
type PMAEntityState string

const (
	StateOn          PMAEntityState = "on"
	StateOff         PMAEntityState = "off"
	StateOpen        PMAEntityState = "open"
	StateClosed      PMAEntityState = "closed"
	StateLocked      PMAEntityState = "locked"
	StateUnlocked    PMAEntityState = "unlocked"
	StateIdle        PMAEntityState = "idle"
	StateActive      PMAEntityState = "active"
	StateUnavailable PMAEntityState = "unavailable"
	StateUnknown     PMAEntityState = "unknown"
)

// PMACapability represents capabilities that entities can support
type PMACapability string

const (
	CapabilityDimmable     PMACapability = "dimmable"
	CapabilityColorable    PMACapability = "colorable"
	CapabilityTemperature  PMACapability = "temperature"
	CapabilityHumidity     PMACapability = "humidity"
	CapabilityPosition     PMACapability = "position"
	CapabilityVolume       PMACapability = "volume"
	CapabilityBrightness   PMACapability = "brightness"
	CapabilityMotion       PMACapability = "motion"
	CapabilityRecording    PMACapability = "recording"
	CapabilityStreaming    PMACapability = "streaming"
	CapabilityNotification PMACapability = "notification"
	CapabilityBattery      PMACapability = "battery"
	CapabilityConnectivity PMACapability = "connectivity"
)

// PMAMetadata contains source-specific metadata and tracking information
type PMAMetadata struct {
	Source         PMASourceType          `json:"source"`
	SourceEntityID string                 `json:"source_entity_id"`
	SourceDeviceID *string                `json:"source_device_id,omitempty"`
	SourceData     map[string]interface{} `json:"source_data,omitempty"`
	LastSynced     time.Time              `json:"last_synced"`
	SyncErrors     []string               `json:"sync_errors,omitempty"`
	QualityScore   float64                `json:"quality_score"` // 0.0-1.0 reliability score
	IsVirtual      bool                   `json:"is_virtual"`    // Combines multiple sources
	VirtualSources []PMASourceType        `json:"virtual_sources,omitempty"`
}

// PMAContext represents the context of a state change or action
type PMAContext struct {
	ID          string    `json:"id"`
	UserID      *string   `json:"user_id,omitempty"`
	TriggerID   *string   `json:"trigger_id,omitempty"`
	Source      string    `json:"source"`
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description,omitempty"`
}

// PMAError represents errors in the PMA type system
type PMAError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	EntityID  string                 `json:"entity_id"`
	Timestamp time.Time              `json:"timestamp"`
	Retryable bool                   `json:"retryable"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// PMAControlAction represents control actions that can be performed on entities
type PMAControlAction struct {
	Action     string                 `json:"action"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
	EntityID   string                 `json:"entity_id"`
	Context    *PMAContext            `json:"context,omitempty"`
	Validation map[string]interface{} `json:"validation,omitempty"`
}

// PMAControlResult represents the result of a control action
type PMAControlResult struct {
	Success     bool                   `json:"success"`
	EntityID    string                 `json:"entity_id"`
	Action      string                 `json:"action"`
	NewState    PMAEntityState         `json:"new_state,omitempty"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	Error       *PMAError              `json:"error,omitempty"`
	ProcessedAt time.Time              `json:"processed_at"`
	Duration    time.Duration          `json:"duration"`
}
