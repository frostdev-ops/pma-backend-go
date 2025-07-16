package homeassistant

import (
	"encoding/json"
	"time"
)

// Function type aliases for cleaner interfaces
type EventHandler func(event Event)
type StateChangeHandler func(entityID string, oldState, newState *EntityState)
type ConnectionStateHandler func(connected bool)

// HAConfig represents Home Assistant configuration
type HAConfig struct {
	Version               string     `json:"version"`
	ConfigDir             string     `json:"config_dir"`
	Elevation             int        `json:"elevation"`
	Latitude              float64    `json:"latitude"`
	Longitude             float64    `json:"longitude"`
	LocationName          string     `json:"location_name"`
	TimeZone              string     `json:"time_zone"`
	Components            []string   `json:"components"`
	UnitSystem            UnitSystem `json:"unit_system"`
	WhitelistExternalDirs []string   `json:"whitelist_external_dirs"`
	AllowlistExternalDirs []string   `json:"allowlist_external_dirs"`
	ExternalURL           string     `json:"external_url"`
	InternalURL           string     `json:"internal_url"`
	Currency              string     `json:"currency"`
	Country               string     `json:"country"`
	Language              string     `json:"language"`
	SafeMode              bool       `json:"safe_mode"`
	State                 string     `json:"state"`
	ExternalURLConfigured bool       `json:"external_url_configured"`
	InternalURLConfigured bool       `json:"internal_url_configured"`
}

// UnitSystem represents Home Assistant unit system
type UnitSystem struct {
	Length      string `json:"length"`
	Mass        string `json:"mass"`
	Temperature string `json:"temperature"`
	Volume      string `json:"volume"`
}

// EntityState represents a Home Assistant entity state
type EntityState struct {
	EntityID    string                 `json:"entity_id"`
	State       string                 `json:"state"`
	Attributes  map[string]interface{} `json:"attributes"`
	LastChanged time.Time              `json:"last_changed"`
	LastUpdated time.Time              `json:"last_updated"`
	Context     Context                `json:"context"`
}

// Context represents the context of an entity state change
type Context struct {
	ID       string  `json:"id"`
	ParentID *string `json:"parent_id"`
	UserID   *string `json:"user_id"`
}

// Area represents a Home Assistant area/room
type Area struct {
	AreaID  string   `json:"area_id"`
	Name    string   `json:"name"`
	Picture *string  `json:"picture"`
	Aliases []string `json:"aliases"`
}

// Device represents a Home Assistant device
type Device struct {
	ID               string     `json:"id"`
	AreaID           *string    `json:"area_id"`
	ConfigurationURL *string    `json:"configuration_url"`
	Connections      [][]string `json:"connections"`
	Identifiers      [][]string `json:"identifiers"`
	Manufacturer     *string    `json:"manufacturer"`
	Model            *string    `json:"model"`
	Name             *string    `json:"name"`
	NameByUser       *string    `json:"name_by_user"`
	SWVersion        *string    `json:"sw_version"`
	HWVersion        *string    `json:"hw_version"`
	ViaDeviceID      *string    `json:"via_device_id"`
	DisabledBy       *string    `json:"disabled_by"`
	EntryType        *string    `json:"entry_type"`
}

// Service represents a Home Assistant service
type Service struct {
	Domain      string                 `json:"domain"`
	Service     string                 `json:"service"`
	ServiceData map[string]interface{} `json:"service_data,omitempty"`
	Target      *ServiceTarget         `json:"target,omitempty"`
}

// ServiceTarget represents service call targets
type ServiceTarget struct {
	EntityID []string `json:"entity_id,omitempty"`
	DeviceID []string `json:"device_id,omitempty"`
	AreaID   []string `json:"area_id,omitempty"`
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	ID      int             `json:"id,omitempty"`
	Type    string          `json:"type"`
	Success *bool           `json:"success,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Event   *Event          `json:"event,omitempty"`
	Error   *WSError        `json:"error,omitempty"`
}

// WSError represents a WebSocket error
type WSError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Event represents a Home Assistant event
type Event struct {
	EventType string                 `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
	Origin    string                 `json:"origin"`
	TimeFired time.Time              `json:"time_fired"`
	Context   Context                `json:"context"`
}

// StateChangedEventData represents data for state_changed events
type StateChangedEventData struct {
	EntityID string       `json:"entity_id"`
	OldState *EntityState `json:"old_state"`
	NewState *EntityState `json:"new_state"`
}

// AuthMessage represents the WebSocket authentication message
type AuthMessage struct {
	Type        string `json:"type"`
	AccessToken string `json:"access_token,omitempty"`
}

// AuthOKMessage represents successful WebSocket authentication
type AuthOKMessage struct {
	Type      string `json:"type"`
	HAVersion string `json:"ha_version"`
}

// SubscribeEventsMessage represents event subscription message
type SubscribeEventsMessage struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	EventType string `json:"event_type,omitempty"`
}

// CallServiceMessage represents service call message
type CallServiceMessage struct {
	ID          int                    `json:"id"`
	Type        string                 `json:"type"`
	Domain      string                 `json:"domain"`
	Service     string                 `json:"service"`
	ServiceData map[string]interface{} `json:"service_data,omitempty"`
	Target      *ServiceTarget         `json:"target,omitempty"`
}

// GetStatesMessage represents get states message
type GetStatesMessage struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

// PingMessage represents ping message
type PingMessage struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
}

// PongMessage represents pong response
type PongMessage struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Success bool   `json:"success"`
}


