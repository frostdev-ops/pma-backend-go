package ring

import (
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
)

// convertDoorbellToPMACamera converts a Ring doorbell to a PMA camera entity
func (a *RingAdapter) convertDoorbellToPMACamera(doorbell *RingDoorbell) (types.PMAEntity, error) {
	entity := &types.PMABaseEntity{
		ID:           doorbell.GetID(),
		Type:         types.EntityTypeCamera,
		FriendlyName: doorbell.GetName(),
		Icon:         "mdi:doorbell-video",
		State:        a.mapDeviceStateToPMAState(doorbell),
		Attributes:   a.convertDoorbellAttributes(doorbell),
		LastUpdated:  time.Now(),
		Capabilities: a.convertDoorbellCapabilities(doorbell),
		Metadata: &types.PMAMetadata{
			Source:         types.SourceRing,
			SourceEntityID: fmt.Sprintf("%d", doorbell.deviceData.ID),
			SourceData:     a.convertDoorbellToSourceData(doorbell),
			LastSynced:     time.Now(),
			QualityScore:   0.95, // High quality for Ring devices
		},
		Available: doorbell.GetStatus() == "online",
	}

	return entity, nil
}

// convertCameraToPMACamera converts a Ring camera to a PMA camera entity
func (a *RingAdapter) convertCameraToPMACamera(camera *RingCamera) (types.PMAEntity, error) {
	entity := &types.PMABaseEntity{
		ID:           camera.GetID(),
		Type:         types.EntityTypeCamera,
		FriendlyName: camera.GetName(),
		Icon:         "mdi:cctv",
		State:        a.mapDeviceStateToPMAState(camera),
		Attributes:   a.convertCameraAttributes(camera),
		LastUpdated:  time.Now(),
		Capabilities: a.convertCameraCapabilities(camera),
		Metadata: &types.PMAMetadata{
			Source:         types.SourceRing,
			SourceEntityID: fmt.Sprintf("%d", camera.deviceData.ID),
			SourceData:     a.convertCameraToSourceData(camera),
			LastSynced:     time.Now(),
			QualityScore:   0.95,
		},
		Available: camera.GetStatus() == "online",
	}

	return entity, nil
}

// convertChimeToPMADevice converts a Ring chime to a PMA device entity
func (a *RingAdapter) convertChimeToPMADevice(chime *RingChime) (types.PMAEntity, error) {
	entity := &types.PMABaseEntity{
		ID:           chime.GetID(),
		Type:         types.EntityTypeDevice,
		FriendlyName: chime.GetName(),
		Icon:         "mdi:bell-ring",
		State:        a.mapDeviceStateToPMAState(chime),
		Attributes:   a.convertChimeAttributes(chime),
		LastUpdated:  time.Now(),
		Capabilities: a.convertChimeCapabilities(chime),
		Metadata: &types.PMAMetadata{
			Source:         types.SourceRing,
			SourceEntityID: fmt.Sprintf("%d", chime.deviceData.ID),
			SourceData:     a.convertChimeToSourceData(chime),
			LastSynced:     time.Now(),
			QualityScore:   0.90,
		},
		Available: chime.GetStatus() == "online",
	}

	return entity, nil
}

// Helper methods for attribute conversion
func (a *RingAdapter) convertDoorbellAttributes(doorbell *RingDoorbell) map[string]interface{} {
	attrs := make(map[string]interface{})

	// Basic doorbell attributes
	attrs["device_type"] = "doorbell"
	attrs["motion_detection"] = doorbell.deviceData.MotionDetection
	attrs["streaming_enabled"] = doorbell.deviceData.StreamingEnabled
	attrs["has_subscription"] = doorbell.deviceData.HasSubscription

	// Battery info if available
	if doorbell.deviceData.BatteryLife != nil {
		attrs["battery_level"] = *doorbell.deviceData.BatteryLife
		attrs["battery_status"] = a.getBatteryStatus(*doorbell.deviceData.BatteryLife)
	}

	// Location info
	attrs["location"] = doorbell.deviceData.Location
	attrs["address"] = doorbell.deviceData.Address
	attrs["timezone"] = doorbell.deviceData.Timezone

	// Video capabilities
	if doorbell.deviceData.HasSubscription {
		attrs["recording_enabled"] = true
	}

	return attrs
}

func (a *RingAdapter) convertCameraAttributes(camera *RingCamera) map[string]interface{} {
	attrs := make(map[string]interface{})

	attrs["device_type"] = "camera"
	attrs["motion_detection"] = camera.deviceData.MotionDetection
	attrs["streaming_enabled"] = camera.deviceData.StreamingEnabled
	attrs["has_subscription"] = camera.deviceData.HasSubscription

	if camera.deviceData.BatteryLife != nil {
		attrs["battery_level"] = *camera.deviceData.BatteryLife
		attrs["battery_status"] = a.getBatteryStatus(*camera.deviceData.BatteryLife)
	}

	attrs["location"] = camera.deviceData.Location
	attrs["address"] = camera.deviceData.Address
	attrs["timezone"] = camera.deviceData.Timezone

	return attrs
}

func (a *RingAdapter) convertChimeAttributes(chime *RingChime) map[string]interface{} {
	attrs := make(map[string]interface{})

	attrs["device_type"] = "chime"
	attrs["location"] = chime.deviceData.Location
	attrs["address"] = chime.deviceData.Address
	attrs["timezone"] = chime.deviceData.Timezone

	return attrs
}

// Helper methods for capability conversion
func (a *RingAdapter) convertDoorbellCapabilities(doorbell *RingDoorbell) []types.PMACapability {
	caps := []types.PMACapability{
		types.CapabilityStreaming,
		types.CapabilityRecording,
		types.CapabilityMotion,
		types.CapabilityNotification,
	}

	if doorbell.deviceData.BatteryLife != nil {
		caps = append(caps, types.CapabilityBattery)
	}

	return caps
}

func (a *RingAdapter) convertCameraCapabilities(camera *RingCamera) []types.PMACapability {
	caps := []types.PMACapability{
		types.CapabilityStreaming,
		types.CapabilityRecording,
		types.CapabilityMotion,
	}

	if camera.deviceData.BatteryLife != nil {
		caps = append(caps, types.CapabilityBattery)
	}

	return caps
}

func (a *RingAdapter) convertChimeCapabilities(chime *RingChime) []types.PMACapability {
	return []types.PMACapability{
		types.CapabilityNotification,
	}
}

// Helper methods for state mapping
func (a *RingAdapter) mapDeviceStateToPMAState(device interface{}) types.PMAEntityState {
	// Ring devices don't have traditional on/off states
	// Use availability as the primary state indicator
	switch d := device.(type) {
	case *RingDoorbell:
		if d.GetStatus() == "online" {
			return types.StateActive
		}
		return types.StateUnavailable
	case *RingCamera:
		if d.GetStatus() == "online" {
			return types.StateActive
		}
		return types.StateUnavailable
	case *RingChime:
		if d.GetStatus() == "online" {
			return types.StateActive
		}
		return types.StateUnavailable
	default:
		return types.StateUnknown
	}
}

// Helper methods for source data conversion
func (a *RingAdapter) convertDoorbellToSourceData(doorbell *RingDoorbell) map[string]interface{} {
	return map[string]interface{}{
		"ring_device_id": doorbell.deviceData.ID,
		"kind":           doorbell.deviceData.Kind,
		"features":       doorbell.deviceData.Features,
		"settings":       doorbell.deviceData.Settings,
		"latitude":       doorbell.deviceData.Latitude,
		"longitude":      doorbell.deviceData.Longitude,
	}
}

func (a *RingAdapter) convertCameraToSourceData(camera *RingCamera) map[string]interface{} {
	return map[string]interface{}{
		"ring_device_id": camera.deviceData.ID,
		"kind":           camera.deviceData.Kind,
		"features":       camera.deviceData.Features,
		"settings":       camera.deviceData.Settings,
		"latitude":       camera.deviceData.Latitude,
		"longitude":      camera.deviceData.Longitude,
	}
}

func (a *RingAdapter) convertChimeToSourceData(chime *RingChime) map[string]interface{} {
	return map[string]interface{}{
		"ring_device_id": chime.deviceData.ID,
		"kind":           chime.deviceData.Kind,
		"features":       chime.deviceData.Features,
		"settings":       chime.deviceData.Settings,
	}
}

// Helper method for battery status
func (a *RingAdapter) getBatteryStatus(batteryLevel int) string {
	switch {
	case batteryLevel > 75:
		return "full"
	case batteryLevel > 50:
		return "good"
	case batteryLevel > 25:
		return "low"
	default:
		return "critical"
	}
}
