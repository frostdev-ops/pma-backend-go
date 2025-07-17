package ring

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/devices"
)

// timePtr returns a pointer to the current time
func timePtr() *time.Time {
	t := time.Now()
	return &t
}

// RingDoorbell represents a Ring doorbell device
type RingDoorbell struct {
	devices.BaseDevice
	client     *RingClient
	deviceData *RingDeviceData
}

// RingCamera represents a Ring camera device
type RingCamera struct {
	devices.BaseDevice
	client     *RingClient
	deviceData *RingDeviceData
}

// RingChime represents a Ring chime device
type RingChime struct {
	devices.BaseDevice
	client     *RingClient
	deviceData *RingDeviceData
}

// NewRingDoorbell creates a new Ring doorbell device
func NewRingDoorbell(client *RingClient, data *RingDeviceData) *RingDoorbell {
	capabilities := []devices.DeviceCapability{
		devices.CapabilityDoorbell,
		devices.CapabilityVideo,
		devices.CapabilityAudio,
		devices.CapabilityMotion,
		devices.CapabilitySnapshot,
		devices.CapabilityLiveStream,
	}

	// Add battery capability if device has battery
	if data.BatteryLife != nil {
		capabilities = append(capabilities, devices.CapabilityBattery)
	}

	// Add recording capability if device has subscription
	if data.HasSubscription {
		capabilities = append(capabilities, devices.CapabilityRecording)
	}

	// Create base device
	baseDevice := devices.BaseDevice{
		ID:           fmt.Sprintf("ring_%s", data.ID),
		Type:         devices.DeviceTypeDoorbell,
		AdapterType:  "ring",
		Name:         data.Description,
		Status:       devices.DeviceStatusOnline,
		Adapter:      "ring",
		Capabilities: capabilities,
		State:        make(map[string]interface{}),
		Metadata:     make(map[string]interface{}),
		LastSeen:     timePtr(),
	}

	// Set initial state
	baseDevice.State["motion_detection"] = data.MotionDetection
	baseDevice.State["streaming_enabled"] = data.StreamingEnabled
	baseDevice.State["subscribed_motions"] = data.SubscribedMotions
	baseDevice.State["has_subscription"] = data.HasSubscription

	if data.BatteryLife != nil {
		baseDevice.State["battery_level"] = *data.BatteryLife
	}

	// Set metadata
	baseDevice.Metadata["location"] = data.Location
	baseDevice.Metadata["latitude"] = data.Latitude
	baseDevice.Metadata["longitude"] = data.Longitude
	baseDevice.Metadata["address"] = data.Address
	baseDevice.Metadata["timezone"] = data.Timezone
	baseDevice.Metadata["ring_device_id"] = data.ID
	baseDevice.Metadata["kind"] = data.Kind
	baseDevice.Metadata["features"] = data.Features
	baseDevice.Metadata["settings"] = data.Settings

	return &RingDoorbell{
		BaseDevice: baseDevice,
		client:     client,
		deviceData: data,
	}
}

// NewRingCamera creates a new Ring camera device
func NewRingCamera(client *RingClient, data *RingDeviceData) *RingCamera {
	capabilities := []devices.DeviceCapability{
		devices.CapabilityVideo,
		devices.CapabilityAudio,
		devices.CapabilityMotion,
		devices.CapabilitySnapshot,
		devices.CapabilityLiveStream,
	}

	// Add battery capability if device has battery
	if data.BatteryLife != nil {
		capabilities = append(capabilities, devices.CapabilityBattery)
	}

	// Add recording capability if device has subscription
	if data.HasSubscription {
		capabilities = append(capabilities, devices.CapabilityRecording)
	}

	// Create base device
	baseDevice := devices.BaseDevice{
		ID:           fmt.Sprintf("ring_%s", data.ID),
		Type:         devices.DeviceTypeCamera,
		AdapterType:  "ring",
		Name:         data.Description,
		Status:       devices.DeviceStatusOnline,
		Adapter:      "ring",
		Capabilities: capabilities,
		State:        make(map[string]interface{}),
		Metadata:     make(map[string]interface{}),
		LastSeen:     timePtr(),
	}

	// Set initial state
	baseDevice.State["motion_detection"] = data.MotionDetection
	baseDevice.State["streaming_enabled"] = data.StreamingEnabled
	baseDevice.State["subscribed_motions"] = data.SubscribedMotions
	baseDevice.State["has_subscription"] = data.HasSubscription

	if data.BatteryLife != nil {
		baseDevice.State["battery_level"] = *data.BatteryLife
	}

	// Set metadata
	baseDevice.Metadata["location"] = data.Location
	baseDevice.Metadata["latitude"] = data.Latitude
	baseDevice.Metadata["longitude"] = data.Longitude
	baseDevice.Metadata["address"] = data.Address
	baseDevice.Metadata["timezone"] = data.Timezone
	baseDevice.Metadata["ring_device_id"] = data.ID
	baseDevice.Metadata["kind"] = data.Kind
	baseDevice.Metadata["features"] = data.Features
	baseDevice.Metadata["settings"] = data.Settings

	return &RingCamera{
		BaseDevice: baseDevice,
		client:     client,
		deviceData: data,
	}
}

// NewRingChime creates a new Ring chime device
func NewRingChime(client *RingClient, data *RingDeviceData) *RingChime {
	capabilities := []devices.DeviceCapability{
		devices.CapabilityAudio,
	}

	// Create base device
	baseDevice := devices.BaseDevice{
		ID:           fmt.Sprintf("ring_%s", data.ID),
		Type:         "chime",
		AdapterType:  "ring",
		Name:         data.Description,
		Status:       devices.DeviceStatusOnline,
		Adapter:      "ring",
		Capabilities: capabilities,
		State:        make(map[string]interface{}),
		Metadata:     make(map[string]interface{}),
		LastSeen:     timePtr(),
	}

	// Set metadata
	baseDevice.Metadata["location"] = data.Location
	baseDevice.Metadata["ring_device_id"] = data.ID
	baseDevice.Metadata["kind"] = data.Kind

	return &RingChime{
		BaseDevice: baseDevice,
		client:     client,
		deviceData: data,
	}
}

// Execute executes a command on the Ring doorbell
func (d *RingDoorbell) Execute(command string, params map[string]interface{}) (interface{}, error) {
	ctx := context.Background()
	deviceID := strconv.Itoa(d.deviceData.ID)

	switch command {
	case "get_snapshot":
		url, err := d.client.GetSnapshot(ctx, deviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get snapshot: %w", err)
		}
		return map[string]interface{}{"snapshot_url": url}, nil

	case "get_live_stream":
		url, err := d.client.GetLiveStreamURL(ctx, deviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get live stream: %w", err)
		}
		return map[string]interface{}{"stream_url": url}, nil

	case "set_lights":
		enabled, ok := params["enabled"].(bool)
		if !ok {
			return nil, devices.ErrInvalidParams
		}
		if err := d.client.SetLights(ctx, deviceID, enabled); err != nil {
			return nil, fmt.Errorf("failed to set lights: %w", err)
		}
		d.SetState("lights_enabled", enabled)
		return map[string]interface{}{"lights_enabled": enabled}, nil

	case "set_siren":
		enabled, ok := params["enabled"].(bool)
		if !ok {
			return nil, devices.ErrInvalidParams
		}
		if err := d.client.SetSiren(ctx, deviceID, enabled); err != nil {
			return nil, fmt.Errorf("failed to set siren: %w", err)
		}
		d.SetState("siren_enabled", enabled)
		return map[string]interface{}{"siren_enabled": enabled}, nil

	case "get_events":
		limit := 10
		if l, ok := params["limit"].(int); ok {
			limit = l
		}
		events, err := d.client.GetEvents(ctx, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to get events: %w", err)
		}

		// Filter events for this device
		deviceEvents := make([]RingEvent, 0)
		for _, event := range events {
			if event.DoorbotID == d.deviceData.ID {
				deviceEvents = append(deviceEvents, event)
			}
		}

		return map[string]interface{}{"events": deviceEvents}, nil

	default:
		return nil, devices.ErrCommandNotSupported
	}
}

// Execute executes a command on the Ring camera
func (c *RingCamera) Execute(command string, params map[string]interface{}) (interface{}, error) {
	ctx := context.Background()
	deviceID := strconv.Itoa(c.deviceData.ID)

	switch command {
	case "get_snapshot":
		url, err := c.client.GetSnapshot(ctx, deviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get snapshot: %w", err)
		}
		return map[string]interface{}{"snapshot_url": url}, nil

	case "get_live_stream":
		url, err := c.client.GetLiveStreamURL(ctx, deviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get live stream: %w", err)
		}
		return map[string]interface{}{"stream_url": url}, nil

	case "set_lights":
		enabled, ok := params["enabled"].(bool)
		if !ok {
			return nil, devices.ErrInvalidParams
		}
		if err := c.client.SetLights(ctx, deviceID, enabled); err != nil {
			return nil, fmt.Errorf("failed to set lights: %w", err)
		}
		c.SetState("lights_enabled", enabled)
		return map[string]interface{}{"lights_enabled": enabled}, nil

	case "set_siren":
		enabled, ok := params["enabled"].(bool)
		if !ok {
			return nil, devices.ErrInvalidParams
		}
		if err := c.client.SetSiren(ctx, deviceID, enabled); err != nil {
			return nil, fmt.Errorf("failed to set siren: %w", err)
		}
		c.SetState("siren_enabled", enabled)
		return map[string]interface{}{"siren_enabled": enabled}, nil

	case "get_events":
		limit := 10
		if l, ok := params["limit"].(int); ok {
			limit = l
		}
		events, err := c.client.GetEvents(ctx, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to get events: %w", err)
		}

		// Filter events for this device
		deviceEvents := make([]RingEvent, 0)
		for _, event := range events {
			if event.DoorbotID == c.deviceData.ID {
				deviceEvents = append(deviceEvents, event)
			}
		}

		return map[string]interface{}{"events": deviceEvents}, nil

	default:
		return nil, devices.ErrCommandNotSupported
	}
}

// Execute executes a command on the Ring chime
func (c *RingChime) Execute(command string, params map[string]interface{}) (interface{}, error) {
	// Chimes have limited functionality
	switch command {
	case "get_status":
		return map[string]interface{}{
			"status": c.GetStatus(),
			"state":  c.GetState(),
		}, nil

	default:
		return nil, devices.ErrCommandNotSupported
	}
}

// UpdateFromRingData updates the device from fresh Ring API data
func (d *RingDoorbell) UpdateFromRingData(data *RingDeviceData) {
	d.deviceData = data
	d.LastSeen = timePtr()

	// Update state
	d.State["motion_detection"] = data.MotionDetection
	d.State["streaming_enabled"] = data.StreamingEnabled
	d.State["subscribed_motions"] = data.SubscribedMotions
	d.State["has_subscription"] = data.HasSubscription

	if data.BatteryLife != nil {
		d.State["battery_level"] = *data.BatteryLife
	}

	// Update metadata
	d.Metadata["features"] = data.Features
	d.Metadata["settings"] = data.Settings
}

// UpdateFromRingData updates the camera device from fresh Ring API data
func (c *RingCamera) UpdateFromRingData(data *RingDeviceData) {
	c.deviceData = data
	c.LastSeen = timePtr()

	// Update state
	c.State["motion_detection"] = data.MotionDetection
	c.State["streaming_enabled"] = data.StreamingEnabled
	c.State["subscribed_motions"] = data.SubscribedMotions
	c.State["has_subscription"] = data.HasSubscription

	if data.BatteryLife != nil {
		c.State["battery_level"] = *data.BatteryLife
	}

	// Update metadata
	c.Metadata["features"] = data.Features
	c.Metadata["settings"] = data.Settings
}

// UpdateFromRingData updates the chime device from fresh Ring API data
func (c *RingChime) UpdateFromRingData(data *RingDeviceData) {
	c.deviceData = data
	c.LastSeen = timePtr()
}

// GetRingDeviceID returns the internal Ring device ID
func (d *RingDoorbell) GetRingDeviceID() int {
	return d.deviceData.ID
}

// GetRingDeviceID returns the internal Ring device ID
func (c *RingCamera) GetRingDeviceID() int {
	return c.deviceData.ID
}

// GetRingDeviceID returns the internal Ring device ID
func (c *RingChime) GetRingDeviceID() int {
	return c.deviceData.ID
}

// IsOnline checks if the device is currently online
func (d *RingDoorbell) IsOnline() bool {
	return d.Status == devices.DeviceStatusOnline
}

// IsOnline checks if the device is currently online
func (c *RingCamera) IsOnline() bool {
	return c.Status == devices.DeviceStatusOnline
}

// IsOnline checks if the device is currently online
func (c *RingChime) IsOnline() bool {
	return c.Status == devices.DeviceStatusOnline
}
