package websocket

import (
	"encoding/json"
	"time"
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

// EntityStateChangedMessage creates a message for entity state changes
func EntityStateChangedMessage(entityID, oldState, newState string, attributes map[string]interface{}) Message {
	return Message{
		Type: "entity_state_changed",
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
		Type: "room_updated",
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
		Type: "system_status",
		Data: map[string]interface{}{
			"status":  status,
			"details": details,
		},
	}
}
