package devices

import (
	"sync"
	"time"
)

// DeviceType represents the type of device
type DeviceType string

const (
	DeviceTypeRingDoorbell DeviceType = "ring_doorbell"
	DeviceTypeRingCamera   DeviceType = "ring_camera"
	DeviceTypeRingChime    DeviceType = "ring_chime"
	DeviceTypeShellySwitch DeviceType = "shelly_switch"
	DeviceTypeShellyDimmer DeviceType = "shelly_dimmer"
	DeviceTypeShellyRGBW   DeviceType = "shelly_rgbw"
	DeviceTypeUPS          DeviceType = "ups"
)

// DeviceStatus represents the current status of a device
type DeviceStatus string

const (
	DeviceStatusOnline       DeviceStatus = "online"
	DeviceStatusOffline      DeviceStatus = "offline"
	DeviceStatusConnecting   DeviceStatus = "connecting"
	DeviceStatusError        DeviceStatus = "error"
	DeviceStatusUnknown      DeviceStatus = "unknown"
	DeviceStatusInitializing DeviceStatus = "initializing"
)

// DeviceCapability represents a capability that a device supports
type DeviceCapability string

const (
	CapabilitySwitch       DeviceCapability = "switch"
	CapabilityDimmer       DeviceCapability = "dimmer"
	CapabilityColorControl DeviceCapability = "color_control"
	CapabilityMotion       DeviceCapability = "motion"
	CapabilityCamera       DeviceCapability = "camera"
	CapabilityDoorbell     DeviceCapability = "doorbell"
	CapabilityBattery      DeviceCapability = "battery"
	CapabilityPower        DeviceCapability = "power"
	CapabilityTemperature  DeviceCapability = "temperature"
)

// EventType represents the type of device event
type EventType string

const (
	EventTypeStateChange    EventType = "state_change"
	EventTypeMotionDetected EventType = "motion_detected"
	EventTypeDoorbellPress  EventType = "doorbell_press"
	EventTypeBatteryLow     EventType = "battery_low"
	EventTypePowerLoss      EventType = "power_loss"
	EventTypePowerRestored  EventType = "power_restored"
	EventTypeConnected      EventType = "connected"
	EventTypeDisconnected   EventType = "disconnected"
)

// Device represents a generic device interface
type Device interface {
	GetID() string
	GetType() DeviceType
	GetName() string
	GetStatus() DeviceStatus
	GetCapabilities() []DeviceCapability
	GetState() map[string]interface{}
	SetState(key string, value interface{}) error
	Execute(command string, params map[string]interface{}) (interface{}, error)
	GetAdapter() string
	GetMetadata() map[string]interface{}
}

// DeviceAdapter represents an adapter for specific device types
type DeviceAdapter interface {
	GetName() string
	Connect() error
	Disconnect() error
	Discover() ([]Device, error)
	Subscribe(callback func(DeviceEvent)) error
	GetDevice(id string) (Device, error)
	GetStatus() AdapterStatus
}

// DeviceEvent represents an event from a device
type DeviceEvent struct {
	DeviceID  string                 `json:"device_id"`
	EventType EventType              `json:"event_type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// AdapterStatus represents the status of a device adapter
type AdapterStatus struct {
	Connected      bool                   `json:"connected"`
	LastError      error                  `json:"last_error,omitempty"`
	LastConnected  *time.Time             `json:"last_connected,omitempty"`
	DeviceCount    int                    `json:"device_count"`
	AdditionalInfo map[string]interface{} `json:"additional_info,omitempty"`
}

// BaseDevice provides a basic implementation of the Device interface
type BaseDevice struct {
	ID           string                 `json:"id"`
	Type         DeviceType             `json:"type"`
	Name         string                 `json:"name"`
	Status       DeviceStatus           `json:"status"`
	Capabilities []DeviceCapability     `json:"capabilities"`
	State        map[string]interface{} `json:"state"`
	Adapter      string                 `json:"adapter"`
	Metadata     map[string]interface{} `json:"metadata"`
	mu           sync.RWMutex
}

// GetID returns the device ID
func (d *BaseDevice) GetID() string {
	return d.ID
}

// GetType returns the device type
func (d *BaseDevice) GetType() DeviceType {
	return d.Type
}

// GetName returns the device name
func (d *BaseDevice) GetName() string {
	return d.Name
}

// GetStatus returns the device status
func (d *BaseDevice) GetStatus() DeviceStatus {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Status
}

// GetCapabilities returns the device capabilities
func (d *BaseDevice) GetCapabilities() []DeviceCapability {
	return d.Capabilities
}

// GetState returns the current state of the device
func (d *BaseDevice) GetState() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Return a copy to prevent external modification
	stateCopy := make(map[string]interface{})
	for k, v := range d.State {
		stateCopy[k] = v
	}
	return stateCopy
}

// SetState updates a state value
func (d *BaseDevice) SetState(key string, value interface{}) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if d.State == nil {
		d.State = make(map[string]interface{})
	}
	d.State[key] = value
	return nil
}

// Execute runs a command on the device
func (d *BaseDevice) Execute(command string, params map[string]interface{}) (interface{}, error) {
	return nil, ErrCommandNotSupported
}

// GetAdapter returns the adapter name
func (d *BaseDevice) GetAdapter() string {
	return d.Adapter
}

// GetMetadata returns the device metadata
func (d *BaseDevice) GetMetadata() map[string]interface{} {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	// Return a copy to prevent external modification
	metadataCopy := make(map[string]interface{})
	for k, v := range d.Metadata {
		metadataCopy[k] = v
	}
	return metadataCopy
}

// SetStatus updates the device status
func (d *BaseDevice) SetStatus(status DeviceStatus) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Status = status
}

// UpdateState updates multiple state values at once
func (d *BaseDevice) UpdateState(updates map[string]interface{}) {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if d.State == nil {
		d.State = make(map[string]interface{})
	}
	
	for k, v := range updates {
		d.State[k] = v
	}
}