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

// AuthSetting represents authentication configuration
type AuthSetting struct {
	ID                int            `json:"id" db:"id"`
	PinCode           sql.NullString `json:"-" db:"pin_code"` // Hidden from JSON
	SessionTimeout    int            `json:"session_timeout" db:"session_timeout"`
	MaxFailedAttempts int            `json:"max_failed_attempts" db:"max_failed_attempts"`
	LockoutDuration   int            `json:"lockout_duration" db:"lockout_duration"`
	LastUpdated       time.Time      `json:"last_updated" db:"last_updated"`
}

// Session represents an authentication session
type Session struct {
	ID        string    `json:"id" db:"id"`
	Token     string    `json:"token" db:"token"`
	ExpiresAt time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// FailedAuthAttempt represents a failed authentication attempt
type FailedAuthAttempt struct {
	ID          int       `json:"id" db:"id"`
	ClientID    string    `json:"client_id" db:"client_id"`
	IPAddress   string    `json:"ip_address" db:"ip_address"`
	AttemptAt   time.Time `json:"attempt_at" db:"attempt_at"`
	AttemptType string    `json:"attempt_type" db:"attempt_type"` // pin, token, etc.
}

// KioskToken represents a kiosk device authentication token
type KioskToken struct {
	ID             string          `json:"id" db:"id"`
	Token          string          `json:"token" db:"token"`
	Name           string          `json:"name" db:"name"`
	RoomID         string          `json:"room_id" db:"room_id"`
	AllowedDevices json.RawMessage `json:"allowed_devices" db:"allowed_devices"`
	Active         bool            `json:"active" db:"active"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
	LastUsed       sql.NullTime    `json:"last_used" db:"last_used"`
	ExpiresAt      sql.NullTime    `json:"expires_at" db:"expires_at"`
}

// KioskPairingSession represents an active kiosk pairing session
type KioskPairingSession struct {
	ID         string          `json:"id" db:"id"`
	Pin        string          `json:"pin" db:"pin"`
	RoomID     string          `json:"room_id" db:"room_id"`
	DeviceInfo json.RawMessage `json:"device_info" db:"device_info"`
	ExpiresAt  time.Time       `json:"expires_at" db:"expires_at"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	Status     string          `json:"status" db:"status"` // pending, confirmed, expired
}

// NetworkDevice represents a discovered network device
type NetworkDevice struct {
	ID           int             `json:"id" db:"id"`
	IPAddress    string          `json:"ip_address" db:"ip_address"`
	MACAddress   sql.NullString  `json:"mac_address" db:"mac_address"`
	Hostname     sql.NullString  `json:"hostname" db:"hostname"`
	Manufacturer sql.NullString  `json:"manufacturer" db:"manufacturer"`
	DeviceType   sql.NullString  `json:"device_type" db:"device_type"`
	LastSeen     time.Time       `json:"last_seen" db:"last_seen"`
	FirstSeen    time.Time       `json:"first_seen" db:"first_seen"`
	IsOnline     bool            `json:"is_online" db:"is_online"`
	Services     json.RawMessage `json:"services" db:"services"`
	Metadata     json.RawMessage `json:"metadata" db:"metadata"`
}

// UPSStatus represents UPS monitoring data
type UPSStatus struct {
	ID             int       `json:"id" db:"id"`
	BatteryCharge  float64   `json:"battery_charge" db:"battery_charge"`
	BatteryRuntime int       `json:"battery_runtime" db:"battery_runtime"`
	InputVoltage   float64   `json:"input_voltage" db:"input_voltage"`
	OutputVoltage  float64   `json:"output_voltage" db:"output_voltage"`
	Load           float64   `json:"load" db:"load"`
	Status         string    `json:"status" db:"status"`
	Temperature    float64   `json:"temperature" db:"temperature"`
	LastUpdated    time.Time `json:"last_updated" db:"last_updated"`
}

// Camera represents a camera device
type Camera struct {
	ID           int             `json:"id" db:"id"`
	EntityID     string          `json:"entity_id" db:"entity_id"`
	Name         string          `json:"name" db:"name"`
	Type         string          `json:"type" db:"type"` // ring, generic, etc.
	StreamURL    sql.NullString  `json:"stream_url" db:"stream_url"`
	SnapshotURL  sql.NullString  `json:"snapshot_url" db:"snapshot_url"`
	Capabilities json.RawMessage `json:"capabilities" db:"capabilities"`
	Settings     json.RawMessage `json:"settings" db:"settings"`
	IsEnabled    bool            `json:"is_enabled" db:"is_enabled"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// DisplaySettings represents display/screensaver configuration
type DisplaySettings struct {
	ID                           int       `json:"id" db:"id"`
	Brightness                   int       `json:"brightness" db:"brightness"`
	Timeout                      int       `json:"timeout" db:"timeout"` // seconds, 0 = never
	Orientation                  string    `json:"orientation" db:"orientation"`
	DarkMode                     string    `json:"darkMode" db:"darkMode"`
	Screensaver                  bool      `json:"screensaver" db:"screensaver"`
	ScreensaverType              string    `json:"screensaverType" db:"screensaverType"`
	ScreensaverShowClock         bool      `json:"screensaverShowClock" db:"screensaverShowClock"`
	ScreensaverRotationSpeed     int       `json:"screensaverRotationSpeed" db:"screensaverRotationSpeed"`
	ScreensaverPictureFrameImage string    `json:"screensaverPictureFrameImage" db:"screensaverPictureFrameImage"`
	ScreensaverUploadEnabled     bool      `json:"screensaverUploadEnabled" db:"screensaverUploadEnabled"`
	DimBeforeSleep               bool      `json:"dimBeforeSleep" db:"dimBeforeSleep"`
	DimLevel                     int       `json:"dimLevel" db:"dimLevel"`
	DimTimeout                   int       `json:"dimTimeout" db:"dimTimeout"`
	CreatedAt                    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt                    time.Time `json:"updatedAt" db:"updated_at"`
}

// BluetoothDevice represents a Bluetooth device
type BluetoothDevice struct {
	ID          int             `json:"id" db:"id"`
	Address     string          `json:"address" db:"address"`
	Name        sql.NullString  `json:"name" db:"name"`
	DeviceClass sql.NullString  `json:"device_class" db:"device_class"`
	IsPaired    bool            `json:"is_paired" db:"is_paired"`
	IsConnected bool            `json:"is_connected" db:"is_connected"`
	Services    json.RawMessage `json:"services" db:"services"`
	LastSeen    time.Time       `json:"last_seen" db:"last_seen"`
	PairedAt    sql.NullTime    `json:"paired_at" db:"paired_at"`
}
