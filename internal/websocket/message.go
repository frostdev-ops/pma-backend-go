package websocket

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
)

// Message types for WebSocket communication using unified PMA types
const (
	// Core PMA message types
	MessageTypePMAEntityStateChanged  = "pma_entity_state_changed"
	MessageTypePMAEntityAdded         = "pma_entity_added"
	MessageTypePMAEntityRemoved       = "pma_entity_removed"
	MessageTypePMAEntityUpdated       = "pma_entity_updated"
	MessageTypePMARoomUpdated         = "pma_room_updated"
	MessageTypePMARoomAdded           = "pma_room_added"
	MessageTypePMARoomRemoved         = "pma_room_removed"
	MessageTypePMAAreaUpdated         = "pma_area_updated"
	MessageTypePMASceneActivated      = "pma_scene_activated"
	MessageTypePMAAutomationTriggered = "pma_automation_triggered"

	// System and synchronization messages
	MessageTypeSystemStatus       = "system_status"
	MessageTypeSyncStatus         = "sync_status"
	MessageTypeAdapterStatus      = "adapter_status"
	MessageTypeConnectionStatus   = "connection_status"
	MessageTypeSubscriptionUpdate = "subscription_update"

	// Legacy message types (deprecated but maintained for compatibility)
	MessageTypeEntityStateChanged = "entity_state_changed" // Deprecated: use pma_entity_state_changed
	MessageTypeRoomUpdated        = "room_updated"         // Deprecated: use pma_room_updated

	// Source-specific events for debugging/monitoring (optional)
	MessageTypeSourceEvent = "source_event"
)

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// UnmarshalJSON provides custom JSON unmarshaling for Message to handle different timestamp formats
func (m *Message) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with Timestamp as interface{} to handle multiple formats
	type TempMessage struct {
		Type      string                 `json:"type"`
		Data      map[string]interface{} `json:"data"`
		Timestamp interface{}            `json:"timestamp"`
	}

	var temp TempMessage
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Copy basic fields
	m.Type = temp.Type
	m.Data = temp.Data

	// Handle timestamp parsing with multiple format support
	m.Timestamp = parseTimestamp(temp.Timestamp)

	return nil
}

// parseTimestamp handles various timestamp formats and converts them to time.Time
func parseTimestamp(ts interface{}) time.Time {
	if ts == nil {
		return time.Now().UTC()
	}

	switch v := ts.(type) {
	case string:
		// Try parsing as Unix timestamp string first
		if unixTime, err := strconv.ParseInt(v, 10, 64); err == nil {
			// Handle both seconds and milliseconds
			if unixTime > 1e12 { // Milliseconds (13+ digits)
				return time.Unix(0, unixTime*int64(time.Millisecond)).UTC()
			} else { // Seconds (10 digits or less)
				return time.Unix(unixTime, 0).UTC()
			}
		}

		// Try parsing as RFC3339 format
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UTC()
		}

		// Try parsing as RFC3339Nano format
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return t.UTC()
		}

		// Fallback to current time if parsing fails
		return time.Now().UTC()

	case float64:
		// Handle numeric timestamp (JavaScript often sends as float64)
		unixTime := int64(v)
		if unixTime > 1e12 { // Milliseconds
			return time.Unix(0, unixTime*int64(time.Millisecond)).UTC()
		} else { // Seconds
			return time.Unix(unixTime, 0).UTC()
		}

	case int64:
		// Handle int64 timestamp
		if v > 1e12 { // Milliseconds
			return time.Unix(0, v*int64(time.Millisecond)).UTC()
		} else { // Seconds
			return time.Unix(v, 0).UTC()
		}

	case int:
		// Handle int timestamp
		unixTime := int64(v)
		if unixTime > 1e12 { // Milliseconds
			return time.Unix(0, unixTime*int64(time.Millisecond)).UTC()
		} else { // Seconds
			return time.Unix(unixTime, 0).UTC()
		}

	default:
		// Fallback to current time for unknown formats
		return time.Now().UTC()
	}
}

// ToJSON converts the message to JSON bytes
func (m Message) ToJSON() []byte {
	m.Timestamp = time.Now().UTC()
	data, _ := json.Marshal(m)
	return data
}

// PMAEntityStateChangedMessage represents a PMA entity state change event
type PMAEntityStateChangedMessage struct {
	Type       string                 `json:"type"`
	EntityID   string                 `json:"entity_id"`
	EntityType types.PMAEntityType    `json:"entity_type"`
	Source     types.PMASourceType    `json:"source"`
	OldState   types.PMAEntityState   `json:"old_state,omitempty"`
	NewState   types.PMAEntityState   `json:"new_state"`
	Attributes map[string]interface{} `json:"attributes"`
	Timestamp  time.Time              `json:"timestamp"`
	RoomID     *string                `json:"room_id,omitempty"`
	AreaID     *string                `json:"area_id,omitempty"`
}

// ToMessage converts PMAEntityStateChangedMessage to generic Message
func (p PMAEntityStateChangedMessage) ToMessage() Message {
	return Message{
		Type: p.Type,
		Data: map[string]interface{}{
			"entity_id":   p.EntityID,
			"entity_type": p.EntityType,
			"source":      p.Source,
			"old_state":   p.OldState,
			"new_state":   p.NewState,
			"attributes":  p.Attributes,
			"room_id":     p.RoomID,
			"area_id":     p.AreaID,
		},
		Timestamp: p.Timestamp,
	}
}

// PMAEntityAddedMessage represents a new PMA entity
type PMAEntityAddedMessage struct {
	Type      string              `json:"type"`
	Entity    types.PMAEntity     `json:"entity"`
	Source    types.PMASourceType `json:"source"`
	Timestamp time.Time           `json:"timestamp"`
	RoomID    *string             `json:"room_id,omitempty"`
	AreaID    *string             `json:"area_id,omitempty"`
}

// ToMessage converts PMAEntityAddedMessage to generic Message
func (p PMAEntityAddedMessage) ToMessage() Message {
	return Message{
		Type: p.Type,
		Data: map[string]interface{}{
			"entity":  p.Entity,
			"source":  p.Source,
			"room_id": p.RoomID,
			"area_id": p.AreaID,
		},
		Timestamp: p.Timestamp,
	}
}

// PMAEntityRemovedMessage represents a removed PMA entity
type PMAEntityRemovedMessage struct {
	Type      string              `json:"type"`
	EntityID  string              `json:"entity_id"`
	Source    types.PMASourceType `json:"source"`
	Timestamp time.Time           `json:"timestamp"`
	RoomID    *string             `json:"room_id,omitempty"`
}

// ToMessage converts PMAEntityRemovedMessage to generic Message
func (p PMAEntityRemovedMessage) ToMessage() Message {
	return Message{
		Type: p.Type,
		Data: map[string]interface{}{
			"entity_id": p.EntityID,
			"source":    p.Source,
			"room_id":   p.RoomID,
		},
		Timestamp: p.Timestamp,
	}
}

// PMARoomUpdatedMessage represents a PMA room update
type PMARoomUpdatedMessage struct {
	Type      string              `json:"type"`
	Room      *types.PMARoom      `json:"room"`
	Action    string              `json:"action"` // "created", "updated", "deleted"
	Source    types.PMASourceType `json:"source"`
	Timestamp time.Time           `json:"timestamp"`
}

// ToMessage converts PMARoomUpdatedMessage to generic Message
func (p PMARoomUpdatedMessage) ToMessage() Message {
	return Message{
		Type: p.Type,
		Data: map[string]interface{}{
			"room":   p.Room,
			"action": p.Action,
			"source": p.Source,
		},
		Timestamp: p.Timestamp,
	}
}

// SyncStatusMessage represents synchronization status updates
type SyncStatusMessage struct {
	Type         string              `json:"type"`
	Source       types.PMASourceType `json:"source"`
	Status       string              `json:"status"` // "connected", "disconnected", "syncing", "error", "completed"
	Message      string              `json:"message,omitempty"`
	Timestamp    time.Time           `json:"timestamp"`
	EntityCount  int                 `json:"entity_count,omitempty"`
	RoomCount    int                 `json:"room_count,omitempty"`
	ErrorCount   int                 `json:"error_count,omitempty"`
	SyncDuration *time.Duration      `json:"sync_duration,omitempty"`
}

// ToMessage converts SyncStatusMessage to generic Message
func (s SyncStatusMessage) ToMessage() Message {
	return Message{
		Type: s.Type,
		Data: map[string]interface{}{
			"source":        s.Source,
			"status":        s.Status,
			"message":       s.Message,
			"entity_count":  s.EntityCount,
			"room_count":    s.RoomCount,
			"error_count":   s.ErrorCount,
			"sync_duration": s.SyncDuration,
		},
		Timestamp: s.Timestamp,
	}
}

// AdapterStatusMessage represents adapter health and status updates
type AdapterStatusMessage struct {
	Type        string                `json:"type"`
	AdapterID   string                `json:"adapter_id"`
	AdapterName string                `json:"adapter_name"`
	Source      types.PMASourceType   `json:"source"`
	Status      string                `json:"status"` // "connected", "disconnected", "error", "healthy", "unhealthy"
	Health      *types.AdapterHealth  `json:"health,omitempty"`
	Metrics     *types.AdapterMetrics `json:"metrics,omitempty"`
	Timestamp   time.Time             `json:"timestamp"`
}

// ToMessage converts AdapterStatusMessage to generic Message
func (a AdapterStatusMessage) ToMessage() Message {
	return Message{
		Type: a.Type,
		Data: map[string]interface{}{
			"adapter_id":   a.AdapterID,
			"adapter_name": a.AdapterName,
			"source":       a.Source,
			"status":       a.Status,
			"health":       a.Health,
			"metrics":      a.Metrics,
		},
		Timestamp: a.Timestamp,
	}
}

// SourceEventMessage represents low-level source events for debugging
type SourceEventMessage struct {
	Type        string              `json:"type"`
	Source      types.PMASourceType `json:"source"`
	EventType   string              `json:"event_type"`
	EventData   interface{}         `json:"event_data"`
	Timestamp   time.Time           `json:"timestamp"`
	Processed   bool                `json:"processed"`
	ProcessedBy string              `json:"processed_by,omitempty"`
}

// ToMessage converts SourceEventMessage to generic Message
func (s SourceEventMessage) ToMessage() Message {
	return Message{
		Type: s.Type,
		Data: map[string]interface{}{
			"source":       s.Source,
			"event_type":   s.EventType,
			"event_data":   s.EventData,
			"processed":    s.Processed,
			"processed_by": s.ProcessedBy,
		},
		Timestamp: s.Timestamp,
	}
}

// Helper functions for creating PMA WebSocket messages

// NewPMAEntityStateChangedMessage creates a new PMA entity state changed message
func NewPMAEntityStateChangedMessage(entity types.PMAEntity, oldState, newState types.PMAEntityState) PMAEntityStateChangedMessage {
	return PMAEntityStateChangedMessage{
		Type:       MessageTypePMAEntityStateChanged,
		EntityID:   entity.GetID(),
		EntityType: entity.GetType(),
		Source:     entity.GetSource(),
		OldState:   oldState,
		NewState:   newState,
		Attributes: entity.GetAttributes(),
		Timestamp:  time.Now().UTC(),
		RoomID:     entity.GetRoomID(),
		AreaID:     entity.GetAreaID(),
	}
}

// NewPMAEntityAddedMessage creates a new PMA entity added message
func NewPMAEntityAddedMessage(entity types.PMAEntity) PMAEntityAddedMessage {
	return PMAEntityAddedMessage{
		Type:      MessageTypePMAEntityAdded,
		Entity:    entity,
		Source:    entity.GetSource(),
		Timestamp: time.Now().UTC(),
		RoomID:    entity.GetRoomID(),
		AreaID:    entity.GetAreaID(),
	}
}

// NewPMAEntityRemovedMessage creates a new PMA entity removed message
func NewPMAEntityRemovedMessage(entityID string, source types.PMASourceType, roomID *string) PMAEntityRemovedMessage {
	return PMAEntityRemovedMessage{
		Type:      MessageTypePMAEntityRemoved,
		EntityID:  entityID,
		Source:    source,
		Timestamp: time.Now().UTC(),
		RoomID:    roomID,
	}
}

// NewPMARoomUpdatedMessage creates a new PMA room updated message
func NewPMARoomUpdatedMessage(room *types.PMARoom, action string) PMARoomUpdatedMessage {
	return PMARoomUpdatedMessage{
		Type:      MessageTypePMARoomUpdated,
		Room:      room,
		Action:    action,
		Source:    room.GetSource(),
		Timestamp: time.Now().UTC(),
	}
}

// NewSyncStatusMessage creates a new sync status message
func NewSyncStatusMessage(source types.PMASourceType, status, message string, entityCount, roomCount, errorCount int, duration *time.Duration) SyncStatusMessage {
	return SyncStatusMessage{
		Type:         MessageTypeSyncStatus,
		Source:       source,
		Status:       status,
		Message:      message,
		Timestamp:    time.Now().UTC(),
		EntityCount:  entityCount,
		RoomCount:    roomCount,
		ErrorCount:   errorCount,
		SyncDuration: duration,
	}
}

// NewAdapterStatusMessage creates a new adapter status message
func NewAdapterStatusMessage(adapter types.PMAAdapter, status string) AdapterStatusMessage {
	return AdapterStatusMessage{
		Type:        MessageTypeAdapterStatus,
		AdapterID:   adapter.GetID(),
		AdapterName: adapter.GetName(),
		Source:      adapter.GetSourceType(),
		Status:      status,
		Health:      adapter.GetHealth(),
		Metrics:     adapter.GetMetrics(),
		Timestamp:   time.Now().UTC(),
	}
}

// NewSourceEventMessage creates a new source event message for debugging
func NewSourceEventMessage(source types.PMASourceType, eventType string, eventData interface{}, processed bool, processedBy string) SourceEventMessage {
	return SourceEventMessage{
		Type:        MessageTypeSourceEvent,
		Source:      source,
		EventType:   eventType,
		EventData:   eventData,
		Timestamp:   time.Now().UTC(),
		Processed:   processed,
		ProcessedBy: processedBy,
	}
}

// Legacy helper functions (maintained for compatibility)

// EntityStateChangedMessage creates a legacy message for entity state changes
// Deprecated: Use NewPMAEntityStateChangedMessage instead
func EntityStateChangedMessage(entityID, oldState, newState string, attributes map[string]interface{}) Message {
	return Message{
		Type: MessageTypeEntityStateChanged,
		Data: map[string]interface{}{
			"entity_id":  entityID,
			"old_state":  oldState,
			"new_state":  newState,
			"attributes": attributes,
		},
	}
}

// RoomUpdatedMessage creates a legacy message for room updates
// Deprecated: Use NewPMARoomUpdatedMessage instead
func RoomUpdatedMessage(roomID int, roomName string, action string) Message {
	return Message{
		Type: MessageTypeRoomUpdated,
		Data: map[string]interface{}{
			"room_id":   roomID,
			"room_name": roomName,
			"action":    action, // "created", "updated", "deleted"
		},
	}
}

// SystemStatusMessage creates a message for system status updates
func SystemStatusMessage(status string, details map[string]interface{}) Message {
	return Message{
		Type: MessageTypeSystemStatus,
		Data: map[string]interface{}{
			"status":  status,
			"details": details,
		},
	}
}
