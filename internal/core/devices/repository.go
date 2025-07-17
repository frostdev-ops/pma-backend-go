package devices

import (
	"time"
)

// DeviceRepository defines the interface for device persistence
type DeviceRepository interface {
	// Device management
	SaveDevice(device Device) error
	GetDevice(deviceID string) (Device, error)
	GetDevices() ([]Device, error)
	GetDevicesByType(deviceType string) ([]Device, error)
	GetDevicesByAdapter(adapterType string) ([]Device, error)
	DeleteDevice(deviceID string) error
	UpdateDevice(device Device) error

	// Device state management
	SaveDeviceState(deviceID string, state map[string]interface{}) error
	GetDeviceState(deviceID string) (map[string]interface{}, error)
	GetDeviceStateHistory(deviceID string, since time.Time, limit int) ([]DeviceStateRecord, error)

	// Device credentials (encrypted storage)
	SaveDeviceCredentials(deviceID string, credentials map[string]interface{}) error
	GetDeviceCredentials(deviceID string) (map[string]interface{}, error)
	DeleteDeviceCredentials(deviceID string) error

	// Device events
	SaveDeviceEvent(event DeviceEvent) error
	GetDeviceEvents(deviceID string, since time.Time, limit int) ([]DeviceEvent, error)
	GetEventsByType(eventType string, since time.Time, limit int) ([]DeviceEvent, error)

	// Maintenance
	CleanupOldStates(olderThan time.Time) error
	CleanupOldEvents(olderThan time.Time) error
}

// DeviceStateRecord represents a historical device state record
type DeviceStateRecord struct {
	DeviceID  string                 `json:"device_id"`
	State     map[string]interface{} `json:"state"`
	Timestamp time.Time              `json:"timestamp"`
}

// DeviceConfig represents stored device configuration
type DeviceConfig struct {
	DeviceID    string                 `json:"device_id"`
	AdapterType string                 `json:"adapter_type"`
	DeviceType  string                 `json:"device_type"`
	Name        string                 `json:"name"`
	Metadata    map[string]interface{} `json:"metadata"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}
