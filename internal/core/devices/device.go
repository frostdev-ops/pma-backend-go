package devices

import (
	"context"
	"time"
)

// DeviceStatus represents the current status of a device
type DeviceStatus string

const (
	DeviceStatusOnline     DeviceStatus = "online"
	DeviceStatusOffline    DeviceStatus = "offline"
	DeviceStatusError      DeviceStatus = "error"
	DeviceStatusUnknown    DeviceStatus = "unknown"
	DeviceStatusConnecting DeviceStatus = "connecting"
)

// Device represents a generic device interface
type Device interface {
	GetID() string
	GetType() string
	GetName() string
	GetAdapterType() string
	GetStatus() DeviceStatus
	GetCapabilities() []string
	GetState() map[string]interface{}
	SetState(key string, value interface{}) error
	Execute(command string, params map[string]interface{}) (interface{}, error)
	GetLastSeen() time.Time
	GetMetadata() map[string]interface{}
	Validate() error
}

// DeviceAdapter represents an interface for device adapters
type DeviceAdapter interface {
	GetType() string
	Connect(ctx context.Context) error
	Disconnect() error
	Discover(ctx context.Context) ([]Device, error)
	Subscribe(callback func(DeviceEvent)) error
	Unsubscribe() error
	GetDevice(deviceID string) (Device, error)
	GetDevices() []Device
	IsConnected() bool
	HealthCheck() error
}

// DeviceEvent represents an event from a device
type DeviceEvent struct {
	DeviceID    string                 `json:"device_id"`
	AdapterType string                 `json:"adapter_type"`
	EventType   string                 `json:"event_type"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      string                 `json:"source"`
}

// DeviceCommand represents a command to be executed on a device
type DeviceCommand struct {
	DeviceID string                 `json:"device_id"`
	Command  string                 `json:"command"`
	Params   map[string]interface{} `json:"params"`
}

// DeviceDiscoveryResult represents the result of device discovery
type DeviceDiscoveryResult struct {
	Devices []Device `json:"devices"`
	Errors  []error  `json:"errors"`
}

// Common device capabilities
const (
	CapabilityOnOff        = "on_off"
	CapabilityDimming      = "dimming"
	CapabilityColorControl = "color_control"
	CapabilityTemperature  = "temperature"
	CapabilityHumidity     = "humidity"
	CapabilityMotion       = "motion"
	CapabilityVideo        = "video"
	CapabilityAudio        = "audio"
	CapabilityPowerMonitor = "power_monitor"
	CapabilityBattery      = "battery"
	CapabilityDoorbell     = "doorbell"
	CapabilityRecording    = "recording"
	CapabilitySnapshot     = "snapshot"
	CapabilityLiveStream   = "live_stream"
	CapabilityUPS          = "ups"
	CapabilityNetworking   = "networking"
)

// Common device types
const (
	DeviceTypeSwitch       = "switch"
	DeviceTypeDimmer       = "dimmer"
	DeviceTypeRGBW         = "rgbw"
	DeviceTypeSensor       = "sensor"
	DeviceTypeCamera       = "camera"
	DeviceTypeDoorbell     = "doorbell"
	DeviceTypeUPS          = "ups"
	DeviceTypeMotionSensor = "motion_sensor"
	DeviceTypePowerMeter   = "power_meter"
)

// Common event types
const (
	EventTypeStateChanged    = "state_changed"
	EventTypeMotionDetected  = "motion_detected"
	EventTypeDoorbellPressed = "doorbell_pressed"
	EventTypePowerLoss       = "power_loss"
	EventTypeBatteryLow      = "battery_low"
	EventTypeDeviceOnline    = "device_online"
	EventTypeDeviceOffline   = "device_offline"
	EventTypeError           = "error"
)

// BaseDevice provides a basic implementation of common device functionality
type BaseDevice struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	AdapterType  string                 `json:"adapter_type"`
	Status       DeviceStatus           `json:"status"`
	Capabilities []string               `json:"capabilities"`
	State        map[string]interface{} `json:"state"`
	Metadata     map[string]interface{} `json:"metadata"`
	LastSeen     time.Time              `json:"last_seen"`
}

func (d *BaseDevice) GetID() string {
	return d.ID
}

func (d *BaseDevice) GetType() string {
	return d.Type
}

func (d *BaseDevice) GetName() string {
	return d.Name
}

func (d *BaseDevice) GetAdapterType() string {
	return d.AdapterType
}

func (d *BaseDevice) GetStatus() DeviceStatus {
	return d.Status
}

func (d *BaseDevice) GetCapabilities() []string {
	return d.Capabilities
}

func (d *BaseDevice) GetState() map[string]interface{} {
	if d.State == nil {
		d.State = make(map[string]interface{})
	}
	return d.State
}

func (d *BaseDevice) GetLastSeen() time.Time {
	return d.LastSeen
}

func (d *BaseDevice) GetMetadata() map[string]interface{} {
	if d.Metadata == nil {
		d.Metadata = make(map[string]interface{})
	}
	return d.Metadata
}

func (d *BaseDevice) SetState(key string, value interface{}) error {
	if d.State == nil {
		d.State = make(map[string]interface{})
	}
	d.State[key] = value
	d.LastSeen = time.Now()
	return nil
}

func (d *BaseDevice) Validate() error {
	if d.ID == "" {
		return ErrInvalidDeviceID
	}
	if d.Name == "" {
		return ErrInvalidDeviceName
	}
	if d.Type == "" {
		return ErrInvalidDeviceType
	}
	if d.AdapterType == "" {
		return ErrInvalidAdapterType
	}
	return nil
}

// Execute is a default implementation that returns not supported
func (d *BaseDevice) Execute(command string, params map[string]interface{}) (interface{}, error) {
	return nil, ErrCommandNotSupported
}
