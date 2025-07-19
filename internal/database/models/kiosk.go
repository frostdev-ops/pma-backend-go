package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// KioskConfig represents kiosk device configuration
type KioskConfig struct {
	ID                    int             `json:"id" db:"id"`
	RoomID                string          `json:"room_id" db:"room_id"`
	Theme                 string          `json:"theme" db:"theme"`                     // light, dark, auto
	Layout                string          `json:"layout" db:"layout"`                   // grid, list
	QuickActions          json.RawMessage `json:"quick_actions" db:"quick_actions"`     // JSON array of device IDs
	UpdateInterval        int             `json:"update_interval" db:"update_interval"` // milliseconds
	DisplayTimeout        int             `json:"display_timeout" db:"display_timeout"` // seconds
	Brightness            int             `json:"brightness" db:"brightness"`           // 0-100
	ScreensaverEnabled    bool            `json:"screensaver_enabled" db:"screensaver_enabled"`
	ScreensaverType       string          `json:"screensaver_type" db:"screensaver_type"`       // clock, slideshow, blank
	ScreensaverTimeout    int             `json:"screensaver_timeout" db:"screensaver_timeout"` // seconds
	AutoHideNavigation    bool            `json:"auto_hide_navigation" db:"auto_hide_navigation"`
	FullscreenMode        bool            `json:"fullscreen_mode" db:"fullscreen_mode"`
	VoiceControlEnabled   bool            `json:"voice_control_enabled" db:"voice_control_enabled"`
	GestureControlEnabled bool            `json:"gesture_control_enabled" db:"gesture_control_enabled"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at" db:"updated_at"`
}

// KioskDeviceGroup represents a group of kiosk devices for bulk management
type KioskDeviceGroup struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Color       string    `json:"color" db:"color"` // hex color for UI
	Icon        string    `json:"icon" db:"icon"`   // icon name
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// KioskGroupMembership represents the many-to-many relationship between kiosks and groups
type KioskGroupMembership struct {
	ID           int       `json:"id" db:"id"`
	KioskTokenID string    `json:"kiosk_token_id" db:"kiosk_token_id"`
	GroupID      string    `json:"group_id" db:"group_id"`
	AddedAt      time.Time `json:"added_at" db:"added_at"`
}

// KioskLog represents a kiosk activity or error log entry
type KioskLog struct {
	ID           int             `json:"id" db:"id"`
	KioskTokenID string          `json:"kiosk_token_id" db:"kiosk_token_id"`
	Level        string          `json:"level" db:"level"`       // debug, info, warn, error, critical
	Category     string          `json:"category" db:"category"` // system, user_action, device_interaction, error, security
	Message      string          `json:"message" db:"message"`
	Details      json.RawMessage `json:"details" db:"details"` // JSON object with additional context
	DeviceID     sql.NullString  `json:"device_id" db:"device_id"`
	UserAction   sql.NullString  `json:"user_action" db:"user_action"`
	ErrorCode    sql.NullString  `json:"error_code" db:"error_code"`
	StackTrace   sql.NullString  `json:"stack_trace" db:"stack_trace"`
	IPAddress    sql.NullString  `json:"ip_address" db:"ip_address"`
	UserAgent    sql.NullString  `json:"user_agent" db:"user_agent"`
	Timestamp    time.Time       `json:"timestamp" db:"timestamp"`
}

// KioskDeviceStatus represents the health and status of a kiosk device
type KioskDeviceStatus struct {
	ID                  int             `json:"id" db:"id"`
	KioskTokenID        string          `json:"kiosk_token_id" db:"kiosk_token_id"`
	Status              string          `json:"status" db:"status"` // online, offline, pairing, error, maintenance
	LastHeartbeat       time.Time       `json:"last_heartbeat" db:"last_heartbeat"`
	DeviceInfo          json.RawMessage `json:"device_info" db:"device_info"`                 // JSON object with device details
	PerformanceMetrics  json.RawMessage `json:"performance_metrics" db:"performance_metrics"` // JSON object with performance data
	ErrorCount24h       int             `json:"error_count_24h" db:"error_count_24h"`
	UptimeSeconds       int             `json:"uptime_seconds" db:"uptime_seconds"`
	NetworkQuality      sql.NullString  `json:"network_quality" db:"network_quality"` // excellent, good, fair, poor
	BatteryLevel        sql.NullInt64   `json:"battery_level" db:"battery_level"`
	Temperature         sql.NullInt64   `json:"temperature" db:"temperature"`
	MemoryUsagePercent  sql.NullInt64   `json:"memory_usage_percent" db:"memory_usage_percent"`
	CPUUsagePercent     sql.NullInt64   `json:"cpu_usage_percent" db:"cpu_usage_percent"`
	StorageUsagePercent sql.NullInt64   `json:"storage_usage_percent" db:"storage_usage_percent"`
	CreatedAt           time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at" db:"updated_at"`
}

// KioskCommand represents a remote management command for kiosk devices
type KioskCommand struct {
	ID             string          `json:"id" db:"id"`
	KioskTokenID   string          `json:"kiosk_token_id" db:"kiosk_token_id"`
	CommandType    string          `json:"command_type" db:"command_type"` // restart, update_config, refresh, screenshot, etc.
	CommandData    json.RawMessage `json:"command_data" db:"command_data"` // JSON object with command parameters
	Status         string          `json:"status" db:"status"`             // pending, sent, acknowledged, completed, failed, expired
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	SentAt         sql.NullTime    `json:"sent_at" db:"sent_at"`
	AcknowledgedAt sql.NullTime    `json:"acknowledged_at" db:"acknowledged_at"`
	CompletedAt    sql.NullTime    `json:"completed_at" db:"completed_at"`
	ExpiresAt      time.Time       `json:"expires_at" db:"expires_at"`
	ResultData     json.RawMessage `json:"result_data" db:"result_data"` // JSON object with command result
	ErrorMessage   sql.NullString  `json:"error_message" db:"error_message"`
}

// Request/Response DTOs for API endpoints

// KioskPairingRequest represents a kiosk pairing request
type KioskPairingRequest struct {
	Pin            string   `json:"pin" validate:"required,len=6,numeric"`
	RoomID         string   `json:"room_id,omitempty"`
	Name           string   `json:"name" validate:"required,min=2,max=50"`
	AllowedDevices []string `json:"allowed_devices,omitempty"`
}

// KioskPairingResponse represents the response to a pairing request
type KioskPairingResponse struct {
	Success   bool         `json:"success"`
	Token     string       `json:"token,omitempty"`
	Config    *KioskConfig `json:"config,omitempty"`
	ExpiresAt string       `json:"expires_at,omitempty"`
	Error     string       `json:"error,omitempty"`
}

// KioskConfigUpdateRequest represents a request to update kiosk configuration
type KioskConfigUpdateRequest struct {
	Theme                 *string  `json:"theme,omitempty" validate:"omitempty,oneof=light dark auto"`
	Layout                *string  `json:"layout,omitempty" validate:"omitempty,oneof=grid list"`
	QuickActions          []string `json:"quick_actions,omitempty"`
	UpdateInterval        *int     `json:"update_interval,omitempty" validate:"omitempty,min=100,max=60000"`
	DisplayTimeout        *int     `json:"display_timeout,omitempty" validate:"omitempty,min=0,max=3600"`
	Brightness            *int     `json:"brightness,omitempty" validate:"omitempty,min=0,max=100"`
	ScreensaverEnabled    *bool    `json:"screensaver_enabled,omitempty"`
	ScreensaverType       *string  `json:"screensaver_type,omitempty" validate:"omitempty,oneof=clock slideshow blank"`
	ScreensaverTimeout    *int     `json:"screensaver_timeout,omitempty" validate:"omitempty,min=30,max=7200"`
	AutoHideNavigation    *bool    `json:"auto_hide_navigation,omitempty"`
	FullscreenMode        *bool    `json:"fullscreen_mode,omitempty"`
	VoiceControlEnabled   *bool    `json:"voice_control_enabled,omitempty"`
	GestureControlEnabled *bool    `json:"gesture_control_enabled,omitempty"`
}

// KioskDeviceInfo represents minimal device information for kiosk display
type KioskDeviceInfo struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	State      string                 `json:"state"`
	Icon       string                 `json:"icon,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// KioskCommandRequest represents a command to be executed on a device
type KioskCommandRequest struct {
	DeviceID string                 `json:"device_id" validate:"required"`
	Action   string                 `json:"action" validate:"required,oneof=toggle turn_on turn_off set_brightness set_color set_temperature"`
	Payload  map[string]interface{} `json:"payload,omitempty"`
}

// KioskCommandResponse represents the response to a command execution
type KioskCommandResponse struct {
	Success   bool   `json:"success"`
	DeviceID  string `json:"device_id"`
	NewState  string `json:"new_state,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
}

// KioskStatusUpdate represents a real-time status update message
type KioskStatusUpdate struct {
	Type string      `json:"type"` // device_update, config_update, heartbeat, error
	Data interface{} `json:"data"`
}

// KioskDeviceGroupCreateRequest represents a request to create a device group
type KioskDeviceGroupCreateRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=50"`
	Description string `json:"description,omitempty" validate:"max=200"`
	Color       string `json:"color,omitempty" validate:"omitempty,hexcolor"`
	Icon        string `json:"icon,omitempty" validate:"max=50"`
}

// KioskLogQuery represents query parameters for log retrieval
type KioskLogQuery struct {
	Level     string    `json:"level,omitempty"`
	Category  string    `json:"category,omitempty"`
	StartTime time.Time `json:"start_time,omitempty"`
	EndTime   time.Time `json:"end_time,omitempty"`
	DeviceID  string    `json:"device_id,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
}

// KioskStatsResponse represents kiosk system statistics
type KioskStatsResponse struct {
	TotalDevices    int `json:"total_devices"`
	ActiveDevices   int `json:"active_devices"`
	OfflineDevices  int `json:"offline_devices"`
	ErrorDevices    int `json:"error_devices"`
	TotalGroups     int `json:"total_groups"`
	PendingSessions int `json:"pending_pairing_sessions"`
	LogsToday       int `json:"logs_today"`
	ErrorsToday     int `json:"errors_today"`
}
