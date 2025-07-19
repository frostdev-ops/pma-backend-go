package types

import (
	"encoding/json"
	"time"
)

// PMARoom represents a unified room/area in the PMA system
type PMARoom struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Icon        string                 `json:"icon,omitempty"`
	Description string                 `json:"description,omitempty"`
	EntityIDs   []string               `json:"entity_ids"`
	ParentID    *string                `json:"parent_id,omitempty"`
	Children    []string               `json:"children,omitempty"`
	Metadata    *PMAMetadata           `json:"metadata"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PMAArea represents a larger grouping of rooms
type PMAArea struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Icon        string                 `json:"icon,omitempty"`
	Description string                 `json:"description,omitempty"`
	RoomIDs     []string               `json:"room_ids"`
	EntityIDs   []string               `json:"entity_ids"`
	Metadata    *PMAMetadata           `json:"metadata"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PMAScene represents an automation scene
type PMAScene struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Icon        string                 `json:"icon,omitempty"`
	Description string                 `json:"description,omitempty"`
	Actions     []PMAControlAction     `json:"actions"`
	RoomID      *string                `json:"room_id,omitempty"`
	AreaID      *string                `json:"area_id,omitempty"`
	Metadata    *PMAMetadata           `json:"metadata"`
	Attributes  map[string]interface{} `json:"attributes,omitempty"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// PMAAutomation represents an automation rule
type PMAAutomation struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	Description   string                   `json:"description,omitempty"`
	TriggerType   string                   `json:"trigger_type"`
	TriggerConfig map[string]interface{}   `json:"trigger_config"`
	Conditions    []map[string]interface{} `json:"conditions,omitempty"`
	Actions       []PMAControlAction       `json:"actions"`
	Enabled       bool                     `json:"enabled"`
	Metadata      *PMAMetadata             `json:"metadata"`
	LastTriggered *time.Time               `json:"last_triggered,omitempty"`
	CreatedAt     time.Time                `json:"created_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
}

// PMAIntegration represents a connected service/platform
type PMAIntegration struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          PMASourceType          `json:"type"`
	Version       string                 `json:"version,omitempty"`
	Status        string                 `json:"status"` // connected, disconnected, error
	StatusMessage string                 `json:"status_message,omitempty"`
	EntityCount   int                    `json:"entity_count"`
	Config        map[string]interface{} `json:"config,omitempty"`
	Capabilities  []string               `json:"capabilities"`
	LastSync      *time.Time             `json:"last_sync,omitempty"`
	SyncErrors    []string               `json:"sync_errors,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

// Room methods
func (r *PMARoom) GetSource() PMASourceType {
	if r.Metadata != nil {
		return r.Metadata.Source
	}
	return SourcePMA
}

func (r *PMARoom) AddEntity(entityID string) {
	for _, id := range r.EntityIDs {
		if id == entityID {
			return // Already exists
		}
	}
	r.EntityIDs = append(r.EntityIDs, entityID)
	r.UpdatedAt = time.Now()
}

func (r *PMARoom) RemoveEntity(entityID string) {
	for i, id := range r.EntityIDs {
		if id == entityID {
			r.EntityIDs = append(r.EntityIDs[:i], r.EntityIDs[i+1:]...)
			r.UpdatedAt = time.Now()
			return
		}
	}
}

func (r *PMARoom) HasEntity(entityID string) bool {
	for _, id := range r.EntityIDs {
		if id == entityID {
			return true
		}
	}
	return false
}

func (r *PMARoom) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// Area methods
func (a *PMAArea) GetSource() PMASourceType {
	if a.Metadata != nil {
		return a.Metadata.Source
	}
	return SourcePMA
}

func (a *PMAArea) AddRoom(roomID string) {
	for _, id := range a.RoomIDs {
		if id == roomID {
			return // Already exists
		}
	}
	a.RoomIDs = append(a.RoomIDs, roomID)
	a.UpdatedAt = time.Now()
}

func (a *PMAArea) RemoveRoom(roomID string) {
	for i, id := range a.RoomIDs {
		if id == roomID {
			a.RoomIDs = append(a.RoomIDs[:i], a.RoomIDs[i+1:]...)
			a.UpdatedAt = time.Now()
			return
		}
	}
}

func (a *PMAArea) AddEntity(entityID string) {
	for _, id := range a.EntityIDs {
		if id == entityID {
			return // Already exists
		}
	}
	a.EntityIDs = append(a.EntityIDs, entityID)
	a.UpdatedAt = time.Now()
}

func (a *PMAArea) RemoveEntity(entityID string) {
	for i, id := range a.EntityIDs {
		if id == entityID {
			a.EntityIDs = append(a.EntityIDs[:i], a.EntityIDs[i+1:]...)
			a.UpdatedAt = time.Now()
			return
		}
	}
}

func (a *PMAArea) ToJSON() ([]byte, error) {
	return json.Marshal(a)
}

// Scene methods
func (s *PMAScene) Execute() error {
	// Execute all actions in the scene
	for _, action := range s.Actions {
		// This would need to be routed through the appropriate adapter
		// For now, return a placeholder
		_ = action
	}
	return nil
}

func (s *PMAScene) AddAction(action PMAControlAction) {
	s.Actions = append(s.Actions, action)
	s.UpdatedAt = time.Now()
}

func (s *PMAScene) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

// Automation methods
func (a *PMAAutomation) Trigger() error {
	// Execute all actions in the automation
	for _, action := range a.Actions {
		// This would need to be routed through the appropriate adapter
		_ = action
	}
	now := time.Now()
	a.LastTriggered = &now
	return nil
}

func (a *PMAAutomation) AddAction(action PMAControlAction) {
	a.Actions = append(a.Actions, action)
	a.UpdatedAt = time.Now()
}

func (a *PMAAutomation) ToJSON() ([]byte, error) {
	return json.Marshal(a)
}

// Integration methods
func (i *PMAIntegration) IsConnected() bool {
	return i.Status == "connected"
}

func (i *PMAIntegration) HasCapability(capability string) bool {
	for _, cap := range i.Capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

func (i *PMAIntegration) UpdateStatus(status, message string) {
	i.Status = status
	i.StatusMessage = message
	i.UpdatedAt = time.Now()
}

func (i *PMAIntegration) ToJSON() ([]byte, error) {
	return json.Marshal(i)
}
