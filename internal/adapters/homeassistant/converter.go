package homeassistant

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// EntityConverter handles conversion between HomeAssistant entities and PMA types
type EntityConverter struct {
	logger *logrus.Logger
}

// NewEntityConverter creates a new entity converter
func NewEntityConverter(logger *logrus.Logger) *EntityConverter {
	return &EntityConverter{
		logger: logger,
	}
}

// ConvertToPMAEntity converts a HomeAssistant entity to appropriate PMA entity type
func (c *EntityConverter) ConvertToPMAEntity(haEntity *HAEntity) (types.PMAEntity, error) {
	if haEntity == nil {
		return nil, fmt.Errorf("HAEntity is nil")
	}

	baseEntity := &types.PMABaseEntity{
		ID:           fmt.Sprintf("ha_%s", haEntity.EntityID),
		Type:         c.mapEntityType(haEntity.Domain),
		FriendlyName: haEntity.FriendlyName,
		State:        c.mapState(haEntity.State, haEntity.Domain),
		Attributes:   c.convertAttributes(haEntity.Attributes),
		LastUpdated:  haEntity.LastUpdated,
		Capabilities: c.detectCapabilities(haEntity),
		Available:    haEntity.State != "unavailable",
		Metadata: &types.PMAMetadata{
			Source:         types.SourceHomeAssistant,
			SourceEntityID: haEntity.EntityID,
			SourceData: map[string]interface{}{
				"domain":       haEntity.Domain,
				"state":        haEntity.State,
				"last_changed": haEntity.LastChanged,
				"context":      haEntity.Context,
			},
			LastSynced:   time.Now(),
			QualityScore: c.calculateQualityScore(haEntity),
		},
	}

	// Add area/room information if available
	if areaID, ok := haEntity.Attributes["area_id"].(string); ok && areaID != "" {
		roomID := fmt.Sprintf("ha_room_%s", areaID)
		baseEntity.RoomID = &roomID
		areaIDFormatted := fmt.Sprintf("ha_area_%s", areaID)
		baseEntity.AreaID = &areaIDFormatted
	}

	// Add device information if available
	if deviceID, ok := haEntity.Attributes["device_id"].(string); ok && deviceID != "" {
		deviceIDFormatted := fmt.Sprintf("ha_device_%s", deviceID)
		baseEntity.DeviceID = &deviceIDFormatted
	}

	// Convert to specific entity type based on domain
	switch haEntity.Domain {
	case "light":
		return c.convertToLight(haEntity, baseEntity)
	case "switch":
		return c.convertToSwitch(haEntity, baseEntity)
	case "sensor":
		return c.convertToSensor(haEntity, baseEntity)
	case "binary_sensor":
		return c.convertToBinarySensor(haEntity, baseEntity)
	case "climate":
		return c.convertToClimate(haEntity, baseEntity)
	case "cover":
		return c.convertToCover(haEntity, baseEntity)
	case "camera":
		return c.convertToCamera(haEntity, baseEntity)
	case "lock":
		return c.convertToLock(haEntity, baseEntity)
	case "fan":
		return c.convertToFan(haEntity, baseEntity)
	case "media_player":
		return c.convertToMediaPlayer(haEntity, baseEntity)
	default:
		// Return as generic entity for unsupported domains
		baseEntity.Type = types.EntityTypeGeneric
		return baseEntity, nil
	}
}

// mapEntityType maps HomeAssistant domain to PMA entity type
func (c *EntityConverter) mapEntityType(domain string) types.PMAEntityType {
	switch domain {
	case "light":
		return types.EntityTypeLight
	case "switch":
		return types.EntityTypeSwitch
	case "sensor":
		return types.EntityTypeSensor
	case "binary_sensor":
		return types.EntityTypeBinarySensor
	case "climate":
		return types.EntityTypeClimate
	case "cover":
		return types.EntityTypeCover
	case "camera":
		return types.EntityTypeCamera
	case "lock":
		return types.EntityTypeLock
	case "fan":
		return types.EntityTypeFan
	case "media_player":
		return types.EntityTypeMediaPlayer
	default:
		return types.EntityTypeGeneric
	}
}

// mapState maps HomeAssistant state to PMA state
func (c *EntityConverter) mapState(haState, domain string) types.PMAEntityState {
	switch domain {
	case "light", "switch", "fan":
		switch haState {
		case "on":
			return types.StateOn
		case "off":
			return types.StateOff
		case "unavailable":
			return types.StateUnavailable
		default:
			return types.StateUnknown
		}
	case "cover":
		switch haState {
		case "open":
			return types.StateOpen
		case "closed":
			return types.StateClosed
		case "opening", "closing":
			return types.StateActive
		case "unavailable":
			return types.StateUnavailable
		default:
			return types.StateUnknown
		}
	case "lock":
		switch haState {
		case "locked":
			return types.StateLocked
		case "unlocked":
			return types.StateUnlocked
		case "unavailable":
			return types.StateUnavailable
		default:
			return types.StateUnknown
		}
	case "binary_sensor":
		switch haState {
		case "on":
			return types.StateActive
		case "off":
			return types.StateIdle
		case "unavailable":
			return types.StateUnavailable
		default:
			return types.StateUnknown
		}
	case "media_player":
		switch haState {
		case "playing":
			return types.StateActive
		case "paused", "idle":
			return types.StateIdle
		case "off":
			return types.StateOff
		case "unavailable":
			return types.StateUnavailable
		default:
			return types.StateUnknown
		}
	default:
		if haState == "unavailable" {
			return types.StateUnavailable
		}
		return types.StateUnknown
	}
}

// convertAttributes converts and filters HomeAssistant attributes for PMA
func (c *EntityConverter) convertAttributes(haAttributes map[string]interface{}) map[string]interface{} {
	pmaAttributes := make(map[string]interface{})

	// Copy all attributes, filtering out internal ones
	for key, value := range haAttributes {
		// Skip internal attributes
		if strings.HasPrefix(key, "__") || key == "context" {
			continue
		}
		pmaAttributes[key] = value
	}

	return pmaAttributes
}

// detectCapabilities analyzes the entity and its attributes to determine capabilities
func (c *EntityConverter) detectCapabilities(haEntity *HAEntity) []types.PMACapability {
	var capabilities []types.PMACapability

	switch haEntity.Domain {
	case "light":
		capabilities = append(capabilities, types.CapabilityBrightness)
		
		// Check for color capabilities
		if _, hasColorMode := haEntity.Attributes["color_mode"]; hasColorMode {
			capabilities = append(capabilities, types.CapabilityColorable)
		}
		if _, hasRGB := haEntity.Attributes["rgb_color"]; hasRGB {
			capabilities = append(capabilities, types.CapabilityColorable)
		}
		
		// Check for dimming capability
		if _, hasBrightness := haEntity.Attributes["brightness"]; hasBrightness {
			capabilities = append(capabilities, types.CapabilityDimmable)
		}

	case "sensor":
		// Check for specific sensor types
		if deviceClass, ok := haEntity.Attributes["device_class"].(string); ok {
			switch deviceClass {
			case "temperature":
				capabilities = append(capabilities, types.CapabilityTemperature)
			case "humidity":
				capabilities = append(capabilities, types.CapabilityHumidity)
			case "battery":
				capabilities = append(capabilities, types.CapabilityBattery)
			case "motion":
				capabilities = append(capabilities, types.CapabilityMotion)
			}
		}

	case "binary_sensor":
		if deviceClass, ok := haEntity.Attributes["device_class"].(string); ok {
			switch deviceClass {
			case "motion":
				capabilities = append(capabilities, types.CapabilityMotion)
			case "connectivity":
				capabilities = append(capabilities, types.CapabilityConnectivity)
			case "battery":
				capabilities = append(capabilities, types.CapabilityBattery)
			}
		}

	case "climate":
		capabilities = append(capabilities, types.CapabilityTemperature)
		if _, hasHumidity := haEntity.Attributes["current_humidity"]; hasHumidity {
			capabilities = append(capabilities, types.CapabilityHumidity)
		}

	case "cover":
		capabilities = append(capabilities, types.CapabilityPosition)

	case "media_player":
		capabilities = append(capabilities, types.CapabilityVolume)

	case "camera":
		capabilities = append(capabilities, types.CapabilityStreaming)
		if _, hasRecording := haEntity.Attributes["recording"]; hasRecording {
			capabilities = append(capabilities, types.CapabilityRecording)
		}
	}

	// Check for battery level in any entity
	if _, hasBattery := haEntity.Attributes["battery_level"]; hasBattery {
		if !c.hasCapability(capabilities, types.CapabilityBattery) {
			capabilities = append(capabilities, types.CapabilityBattery)
		}
	}

	return capabilities
}

// calculateQualityScore calculates a quality score based on entity availability and attributes
func (c *EntityConverter) calculateQualityScore(haEntity *HAEntity) float64 {
	score := 1.0

	// Reduce score if unavailable
	if haEntity.State == "unavailable" {
		score *= 0.1
	}

	// Reduce score if state is unknown
	if haEntity.State == "unknown" {
		score *= 0.7
	}

	// Boost score if entity has a friendly name
	if haEntity.FriendlyName != "" && haEntity.FriendlyName != haEntity.EntityID {
		score *= 1.1
	}

	// Boost score if entity has area/room assignment
	if _, hasArea := haEntity.Attributes["area_id"]; hasArea {
		score *= 1.05
	}

	// Boost score if entity has device assignment
	if _, hasDevice := haEntity.Attributes["device_id"]; hasDevice {
		score *= 1.05
	}

	// Ensure score is between 0 and 1
	if score > 1.0 {
		score = 1.0
	}
	if score < 0.0 {
		score = 0.0
	}

	return score
}

// Specific entity type converters

func (c *EntityConverter) convertToLight(haEntity *HAEntity, base *types.PMABaseEntity) (types.PMAEntity, error) {
	// For now, return the base entity as PMALightEntity is not defined in the provided types
	// In a full implementation, you would create specific light entity types
	base.Type = types.EntityTypeLight
	return base, nil
}

func (c *EntityConverter) convertToSwitch(haEntity *HAEntity, base *types.PMABaseEntity) (types.PMAEntity, error) {
	base.Type = types.EntityTypeSwitch
	return base, nil
}

func (c *EntityConverter) convertToSensor(haEntity *HAEntity, base *types.PMABaseEntity) (types.PMAEntity, error) {
	base.Type = types.EntityTypeSensor
	
	// Add sensor-specific attributes
	if unitOfMeasurement, ok := haEntity.Attributes["unit_of_measurement"].(string); ok {
		base.Attributes["unit_of_measurement"] = unitOfMeasurement
	}
	
	if deviceClass, ok := haEntity.Attributes["device_class"].(string); ok {
		base.Attributes["device_class"] = deviceClass
	}
	
	// Try to parse numeric value
	if value, err := strconv.ParseFloat(haEntity.State, 64); err == nil {
		base.Attributes["numeric_value"] = value
	}
	
	return base, nil
}

func (c *EntityConverter) convertToBinarySensor(haEntity *HAEntity, base *types.PMABaseEntity) (types.PMAEntity, error) {
	base.Type = types.EntityTypeBinarySensor
	
	if deviceClass, ok := haEntity.Attributes["device_class"].(string); ok {
		base.Attributes["device_class"] = deviceClass
	}
	
	return base, nil
}

func (c *EntityConverter) convertToClimate(haEntity *HAEntity, base *types.PMABaseEntity) (types.PMAEntity, error) {
	base.Type = types.EntityTypeClimate
	return base, nil
}

func (c *EntityConverter) convertToCover(haEntity *HAEntity, base *types.PMABaseEntity) (types.PMAEntity, error) {
	base.Type = types.EntityTypeCover
	return base, nil
}

func (c *EntityConverter) convertToCamera(haEntity *HAEntity, base *types.PMABaseEntity) (types.PMAEntity, error) {
	base.Type = types.EntityTypeCamera
	return base, nil
}

func (c *EntityConverter) convertToLock(haEntity *HAEntity, base *types.PMABaseEntity) (types.PMAEntity, error) {
	base.Type = types.EntityTypeLock
	return base, nil
}

func (c *EntityConverter) convertToFan(haEntity *HAEntity, base *types.PMABaseEntity) (types.PMAEntity, error) {
	base.Type = types.EntityTypeFan
	return base, nil
}

func (c *EntityConverter) convertToMediaPlayer(haEntity *HAEntity, base *types.PMABaseEntity) (types.PMAEntity, error) {
	base.Type = types.EntityTypeMediaPlayer
	return base, nil
}

// Helper methods

func (c *EntityConverter) hasCapability(capabilities []types.PMACapability, capability types.PMACapability) bool {
	for _, cap := range capabilities {
		if cap == capability {
			return true
		}
	}
	return false
} 