package homeassistant

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/sirupsen/logrus"
)

// EntityMapper handles mapping between Home Assistant and PMA entities
type EntityMapper struct {
	logger           *logrus.Logger
	supportedDomains map[string]bool
}

// NewEntityMapper creates a new entity mapper
func NewEntityMapper(logger *logrus.Logger) *EntityMapper {
	supportedDomains := map[string]bool{
		"light":         true,
		"switch":        true,
		"sensor":        true,
		"binary_sensor": true,
		"climate":       true,
		"cover":         true,
		"fan":           true,
		"lock":          true,
		"vacuum":        true,
		"media_player":  true,
	}

	return &EntityMapper{
		logger:           logger,
		supportedDomains: supportedDomains,
	}
}

// EntityMapperInterface defines the interface for entity mapping
type EntityMapperInterface interface {
	// Mapping operations
	HAEntityToPMAEntity(haEntity homeassistant.EntityState) (*models.Entity, error)
	PMAEntityToHAUpdate(entity *models.Entity) (map[string]interface{}, error)

	// Domain handling
	GetSupportedDomains() []string
	IsDomainSupported(domain string) bool

	// Attribute mapping
	MapAttributes(haAttributes map[string]interface{}) (map[string]interface{}, error)
	NormalizeEntityID(haEntityID string) string
}

// HAEntityToPMAEntity converts a Home Assistant entity to a PMA entity
func (m *EntityMapper) HAEntityToPMAEntity(haEntity homeassistant.EntityState) (*models.Entity, error) {
	// Extract domain from entity ID
	domain := m.extractDomain(haEntity.EntityID)

	if !m.IsDomainSupported(domain) {
		return nil, fmt.Errorf("unsupported domain: %s", domain)
	}

	// Get friendly name from attributes
	friendlyName := m.extractFriendlyName(haEntity.Attributes)
	if friendlyName == "" {
		friendlyName = m.generateFriendlyName(haEntity.EntityID)
	}

	// Map and normalize attributes
	normalizedAttributes, err := m.MapAttributes(haEntity.Attributes)
	if err != nil {
		m.logger.WithError(err).Warnf("Failed to map attributes for entity %s", haEntity.EntityID)
		normalizedAttributes = make(map[string]interface{})
	}

	// Convert attributes to JSON
	attributesJSON, err := json.Marshal(normalizedAttributes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal attributes: %w", err)
	}

	// Create PMA entity
	entity := &models.Entity{
		EntityID:     m.NormalizeEntityID(haEntity.EntityID),
		FriendlyName: sql.NullString{String: friendlyName, Valid: true},
		Domain:       domain,
		State:        sql.NullString{String: haEntity.State, Valid: true},
		Attributes:   json.RawMessage(attributesJSON),
		LastUpdated:  haEntity.LastUpdated,
	}

	return entity, nil
}

// PMAEntityToHAUpdate converts a PMA entity to Home Assistant update data
func (m *EntityMapper) PMAEntityToHAUpdate(entity *models.Entity) (map[string]interface{}, error) {
	updateData := make(map[string]interface{})

	// Add entity ID
	updateData["entity_id"] = entity.EntityID

	// Add state if valid
	if entity.State.Valid {
		updateData["state"] = entity.State.String
	}

	// Parse and add attributes
	if len(entity.Attributes) > 0 {
		var attributes map[string]interface{}
		if err := json.Unmarshal(entity.Attributes, &attributes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal entity attributes: %w", err)
		}

		// Add relevant attributes for service calls
		m.addServiceAttributes(updateData, entity.Domain, attributes)
	}

	return updateData, nil
}

// GetSupportedDomains returns a list of supported domains
func (m *EntityMapper) GetSupportedDomains() []string {
	domains := make([]string, 0, len(m.supportedDomains))
	for domain := range m.supportedDomains {
		domains = append(domains, domain)
	}
	return domains
}

// IsDomainSupported checks if a domain is supported
func (m *EntityMapper) IsDomainSupported(domain string) bool {
	return m.supportedDomains[domain]
}

// MapAttributes maps and normalizes Home Assistant attributes for PMA
func (m *EntityMapper) MapAttributes(haAttributes map[string]interface{}) (map[string]interface{}, error) {
	normalized := make(map[string]interface{})

	for key, value := range haAttributes {
		normalizedKey := m.normalizeAttributeKey(key)
		normalizedValue := m.normalizeAttributeValue(value)

		// Skip certain internal HA attributes
		if m.shouldIncludeAttribute(normalizedKey) {
			normalized[normalizedKey] = normalizedValue
		}
	}

	return normalized, nil
}

// NormalizeEntityID normalizes a Home Assistant entity ID for PMA
func (m *EntityMapper) NormalizeEntityID(haEntityID string) string {
	// Home Assistant entity IDs are already in the correct format (domain.object_id)
	// Just ensure lowercase and trim whitespace
	return strings.ToLower(strings.TrimSpace(haEntityID))
}

// Helper methods

// extractDomain extracts the domain from an entity ID
func (m *EntityMapper) extractDomain(entityID string) string {
	if idx := strings.Index(entityID, "."); idx > 0 {
		return entityID[:idx]
	}
	return entityID
}

// extractFriendlyName extracts the friendly name from attributes
func (m *EntityMapper) extractFriendlyName(attributes map[string]interface{}) string {
	if friendlyName, ok := attributes["friendly_name"].(string); ok {
		return friendlyName
	}
	return ""
}

// generateFriendlyName generates a friendly name from entity ID
func (m *EntityMapper) generateFriendlyName(entityID string) string {
	if idx := strings.Index(entityID, "."); idx > 0 && idx < len(entityID)-1 {
		objectID := entityID[idx+1:]
		// Replace underscores with spaces and title case
		return strings.Title(strings.ReplaceAll(objectID, "_", " "))
	}
	return entityID
}

// normalizeAttributeKey normalizes attribute keys
func (m *EntityMapper) normalizeAttributeKey(key string) string {
	// Convert to snake_case if needed and lowercase
	return strings.ToLower(key)
}

// normalizeAttributeValue normalizes attribute values
func (m *EntityMapper) normalizeAttributeValue(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]interface{}:
		// Recursively normalize nested objects
		normalized := make(map[string]interface{})
		for k, val := range v {
			normalized[m.normalizeAttributeKey(k)] = m.normalizeAttributeValue(val)
		}
		return normalized
	case []interface{}:
		// Normalize array elements
		normalized := make([]interface{}, len(v))
		for i, val := range v {
			normalized[i] = m.normalizeAttributeValue(val)
		}
		return normalized
	default:
		return value
	}
}

// shouldIncludeAttribute determines if an attribute should be included
func (m *EntityMapper) shouldIncludeAttribute(key string) bool {
	// Skip internal Home Assistant attributes
	skipAttributes := map[string]bool{
		"context":            true,
		"last_changed":       true,
		"last_updated":       true,
		"supported_features": false, // Keep this one as it's useful
	}

	if skip, exists := skipAttributes[key]; exists {
		return !skip
	}

	// Include all other attributes
	return true
}

// addServiceAttributes adds relevant attributes for Home Assistant service calls
func (m *EntityMapper) addServiceAttributes(updateData map[string]interface{}, domain string, attributes map[string]interface{}) {
	switch domain {
	case "light":
		m.addLightAttributes(updateData, attributes)
	case "climate":
		m.addClimateAttributes(updateData, attributes)
	case "cover":
		m.addCoverAttributes(updateData, attributes)
	case "fan":
		m.addFanAttributes(updateData, attributes)
	case "media_player":
		m.addMediaPlayerAttributes(updateData, attributes)
	}
}

// addLightAttributes adds light-specific attributes
func (m *EntityMapper) addLightAttributes(updateData map[string]interface{}, attributes map[string]interface{}) {
	if brightness, ok := attributes["brightness"]; ok {
		updateData["brightness"] = brightness
	}
	if colorTemp, ok := attributes["color_temp"]; ok {
		updateData["color_temp"] = colorTemp
	}
	if hsColor, ok := attributes["hs_color"]; ok {
		updateData["hs_color"] = hsColor
	}
	if rgbColor, ok := attributes["rgb_color"]; ok {
		updateData["rgb_color"] = rgbColor
	}
}

// addClimateAttributes adds climate-specific attributes
func (m *EntityMapper) addClimateAttributes(updateData map[string]interface{}, attributes map[string]interface{}) {
	if temperature, ok := attributes["temperature"]; ok {
		updateData["temperature"] = temperature
	}
	if hvacMode, ok := attributes["hvac_mode"]; ok {
		updateData["hvac_mode"] = hvacMode
	}
	if fanMode, ok := attributes["fan_mode"]; ok {
		updateData["fan_mode"] = fanMode
	}
}

// addCoverAttributes adds cover-specific attributes
func (m *EntityMapper) addCoverAttributes(updateData map[string]interface{}, attributes map[string]interface{}) {
	if position, ok := attributes["current_position"]; ok {
		updateData["position"] = position
	}
}

// addFanAttributes adds fan-specific attributes
func (m *EntityMapper) addFanAttributes(updateData map[string]interface{}, attributes map[string]interface{}) {
	if speed, ok := attributes["speed"]; ok {
		updateData["speed"] = speed
	}
	if percentage, ok := attributes["percentage"]; ok {
		updateData["percentage"] = percentage
	}
}

// addMediaPlayerAttributes adds media player-specific attributes
func (m *EntityMapper) addMediaPlayerAttributes(updateData map[string]interface{}, attributes map[string]interface{}) {
	if volume, ok := attributes["volume_level"]; ok {
		updateData["volume_level"] = volume
	}
	if source, ok := attributes["source"]; ok {
		updateData["source"] = source
	}
}
