package websocket

import (
	"encoding/json"
	"time"
)

// Message types for WebSocket communication
const (
	// Existing types
	MessageTypeEntityStateChanged = "entity_state_changed"
	MessageTypeRoomUpdated        = "room_updated"
	MessageTypeSystemStatus       = "system_status"

	// New Home Assistant event types
	MessageTypeHAStateChanged  = "ha_state_changed"
	MessageTypeHAEntityAdded   = "ha_entity_added"
	MessageTypeHAEntityRemoved = "ha_entity_removed"
	MessageTypeHAAreaUpdated   = "ha_area_updated"
	MessageTypeHAServiceCalled = "ha_service_called"
	MessageTypeHASyncStatus    = "ha_sync_status"

	// Client subscription management
	MessageTypeSubscriptionUpdate = "subscription_update"
	MessageTypeConnectionStatus   = "connection_status"
)

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// ToJSON converts the message to JSON bytes
func (m Message) ToJSON() []byte {
	m.Timestamp = time.Now().UTC()
	data, _ := json.Marshal(m)
	return data
}

// HAStateChangedMessage represents a Home Assistant state change event
type HAStateChangedMessage struct {
	Type       string                 `json:"type"`
	EntityID   string                 `json:"entity_id"`
	OldState   string                 `json:"old_state,omitempty"`
	NewState   string                 `json:"new_state"`
	Attributes map[string]interface{} `json:"attributes"`
	Timestamp  time.Time              `json:"timestamp"`
	RoomID     *string                `json:"room_id,omitempty"`
}

// ToMessage converts HAStateChangedMessage to generic Message
func (h HAStateChangedMessage) ToMessage() Message {
	return Message{
		Type: h.Type,
		Data: map[string]interface{}{
			"entity_id":  h.EntityID,
			"old_state":  h.OldState,
			"new_state":  h.NewState,
			"attributes": h.Attributes,
			"room_id":    h.RoomID,
		},
		Timestamp: h.Timestamp,
	}
}

// HASyncStatusMessage represents Home Assistant sync status updates
type HASyncStatusMessage struct {
	Type        string    `json:"type"`
	Status      string    `json:"status"` // "connected", "disconnected", "syncing", "error"
	Message     string    `json:"message,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	EntityCount int       `json:"entity_count,omitempty"`
}

// ToMessage converts HASyncStatusMessage to generic Message
func (h HASyncStatusMessage) ToMessage() Message {
	return Message{
		Type: h.Type,
		Data: map[string]interface{}{
			"status":       h.Status,
			"message":      h.Message,
			"entity_count": h.EntityCount,
		},
		Timestamp: h.Timestamp,
	}
}

// HAEntityAddedMessage represents a new Home Assistant entity
type HAEntityAddedMessage struct {
	Type       string                 `json:"type"`
	EntityID   string                 `json:"entity_id"`
	EntityData map[string]interface{} `json:"entity_data"`
	Timestamp  time.Time              `json:"timestamp"`
	RoomID     *string                `json:"room_id,omitempty"`
}

// ToMessage converts HAEntityAddedMessage to generic Message
func (h HAEntityAddedMessage) ToMessage() Message {
	return Message{
		Type: h.Type,
		Data: map[string]interface{}{
			"entity_id":   h.EntityID,
			"entity_data": h.EntityData,
			"room_id":     h.RoomID,
		},
		Timestamp: h.Timestamp,
	}
}

// HAEntityRemovedMessage represents a removed Home Assistant entity
type HAEntityRemovedMessage struct {
	Type      string    `json:"type"`
	EntityID  string    `json:"entity_id"`
	Timestamp time.Time `json:"timestamp"`
	RoomID    *string   `json:"room_id,omitempty"`
}

// ToMessage converts HAEntityRemovedMessage to generic Message
func (h HAEntityRemovedMessage) ToMessage() Message {
	return Message{
		Type: h.Type,
		Data: map[string]interface{}{
			"entity_id": h.EntityID,
			"room_id":   h.RoomID,
		},
		Timestamp: h.Timestamp,
	}
}

// HAServiceCalledMessage represents a Home Assistant service call
type HAServiceCalledMessage struct {
	Type        string                 `json:"type"`
	Service     string                 `json:"service"`
	ServiceData map[string]interface{} `json:"service_data"`
	EntityID    *string                `json:"entity_id,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	RoomID      *string                `json:"room_id,omitempty"`
}

// ToMessage converts HAServiceCalledMessage to generic Message
func (h HAServiceCalledMessage) ToMessage() Message {
	return Message{
		Type: h.Type,
		Data: map[string]interface{}{
			"service":      h.Service,
			"service_data": h.ServiceData,
			"entity_id":    h.EntityID,
			"room_id":      h.RoomID,
		},
		Timestamp: h.Timestamp,
	}
}

// Helper functions for existing message types

// EntityStateChangedMessage creates a message for entity state changes
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

// RoomUpdatedMessage creates a message for room updates
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

// NewHAStateChangedMessage creates a new HA state changed message
func NewHAStateChangedMessage(entityID, oldState, newState string, attributes map[string]interface{}, roomID *string) HAStateChangedMessage {
	return HAStateChangedMessage{
		Type:       MessageTypeHAStateChanged,
		EntityID:   entityID,
		OldState:   oldState,
		NewState:   newState,
		Attributes: attributes,
		Timestamp:  time.Now().UTC(),
		RoomID:     roomID,
	}
}

// NewHASyncStatusMessage creates a new HA sync status message
func NewHASyncStatusMessage(status, message string, entityCount int) HASyncStatusMessage {
	return HASyncStatusMessage{
		Type:        MessageTypeHASyncStatus,
		Status:      status,
		Message:     message,
		Timestamp:   time.Now().UTC(),
		EntityCount: entityCount,
	}
}

// NewHAEntityAddedMessage creates a new HA entity added message
func NewHAEntityAddedMessage(entityID string, entityData map[string]interface{}, roomID *string) HAEntityAddedMessage {
	return HAEntityAddedMessage{
		Type:       MessageTypeHAEntityAdded,
		EntityID:   entityID,
		EntityData: entityData,
		Timestamp:  time.Now().UTC(),
		RoomID:     roomID,
	}
}

// NewHAEntityRemovedMessage creates a new HA entity removed message
func NewHAEntityRemovedMessage(entityID string, roomID *string) HAEntityRemovedMessage {
	return HAEntityRemovedMessage{
		Type:      MessageTypeHAEntityRemoved,
		EntityID:  entityID,
		Timestamp: time.Now().UTC(),
		RoomID:    roomID,
	}
}

// NewHAServiceCalledMessage creates a new HA service called message
func NewHAServiceCalledMessage(service string, serviceData map[string]interface{}, entityID *string, roomID *string) HAServiceCalledMessage {
	return HAServiceCalledMessage{
		Type:        MessageTypeHAServiceCalled,
		Service:     service,
		ServiceData: serviceData,
		EntityID:    entityID,
		Timestamp:   time.Now().UTC(),
		RoomID:      roomID,
	}
}
