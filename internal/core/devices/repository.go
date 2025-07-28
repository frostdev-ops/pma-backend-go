package devices

import (
	"time"
)

// Repository defines the interface for device persistence
type Repository interface {
	// Device operations
	CreateDevice(device Device) error
	UpdateDevice(device Device) error
	CreateOrUpdateDevice(device Device) error
	GetDevice(id string) (Device, error)
	GetAllDevices() ([]Device, error)
	GetDevicesByType(deviceType DeviceType) ([]Device, error)
	DeleteDevice(id string) error

	// State operations
	SaveDeviceState(deviceID string, state map[string]interface{}) error
	GetDeviceState(deviceID string) (map[string]interface{}, error)
	GetDeviceStateHistory(deviceID string, limit int) ([]DeviceStateEntry, error)

	// Event operations
	SaveDeviceEvent(event DeviceEvent) error
	GetDeviceEvents(deviceID string, eventType EventType, since time.Time, limit int) ([]DeviceEvent, error)

	// Credentials operations
	SaveDeviceCredentials(deviceID string, credentials map[string]interface{}) error
	GetDeviceCredentials(deviceID string) (map[string]interface{}, error)
	DeleteDeviceCredentials(deviceID string) error
}

// DeviceStateEntry represents a historical state entry
type DeviceStateEntry struct {
	DeviceID  string                 `json:"device_id"`
	State     map[string]interface{} `json:"state"`
	Timestamp time.Time              `json:"timestamp"`
}

// DeviceModel represents the database model for a device
type DeviceModel struct {
	ID           string                 `json:"id"`
	AdapterType  string                 `json:"adapter_type"`
	DeviceType   string                 `json:"device_type"`
	Name         string                 `json:"name"`
	Status       string                 `json:"status"`
	Capabilities []string               `json:"capabilities"`
	State        map[string]interface{} `json:"state"`
	Metadata     map[string]interface{} `json:"metadata"`
	Config       map[string]interface{} `json:"config"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// ToDevice converts a DeviceModel to a Device
func (m *DeviceModel) ToDevice() Device {
	capabilities := make([]DeviceCapability, len(m.Capabilities))
	for i, cap := range m.Capabilities {
		capabilities[i] = DeviceCapability(cap)
	}

	return &BaseDevice{
		ID:           m.ID,
		Type:         DeviceType(m.DeviceType),
		Name:         m.Name,
		Status:       DeviceStatus(m.Status),
		Capabilities: capabilities,
		State:        m.State,
		Adapter:      m.AdapterType,
		Metadata:     m.Metadata,
	}
}

// FromDevice creates a DeviceModel from a Device
func FromDevice(device Device) *DeviceModel {
	capabilities := device.GetCapabilities()
	capStrings := make([]string, len(capabilities))
	for i, cap := range capabilities {
		capStrings[i] = string(cap)
	}

	return &DeviceModel{
		ID:           device.GetID(),
		AdapterType:  device.GetAdapter(),
		DeviceType:   string(device.GetType()),
		Name:         device.GetName(),
		Status:       string(device.GetStatus()),
		Capabilities: capStrings,
		State:        device.GetState(),
		Metadata:     device.GetMetadata(),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}
