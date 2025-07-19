package homeassistant

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// StateMapper handles state and attribute mapping between PMA and HomeAssistant
type StateMapper struct {
	logger *logrus.Logger
}

// NewStateMapper creates a new state mapper
func NewStateMapper(logger *logrus.Logger) *StateMapper {
	return &StateMapper{
		logger: logger,
	}
}

// MapState maps HomeAssistant state to PMA state
func (m *StateMapper) MapState(haState string, domain string) types.PMAEntityState {
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

// MapActionToService maps a PMA control action to HomeAssistant service call
func (m *StateMapper) MapActionToService(action types.PMAControlAction) (domain, service string, data map[string]interface{}, err error) {
	// Extract domain from entity ID
	haEntityID := m.convertPMAEntityIDToHA(action.EntityID)
	parts := strings.Split(haEntityID, ".")
	if len(parts) != 2 {
		return "", "", nil, fmt.Errorf("invalid entity ID format: %s", haEntityID)
	}
	domain = parts[0]

	// Initialize service data
	data = make(map[string]interface{})

	// Map action based on domain and action type
	switch domain {
	case "light":
		service, data, err = m.mapLightAction(action)
	case "switch":
		service, data, err = m.mapSwitchAction(action)
	case "cover":
		service, data, err = m.mapCoverAction(action)
	case "lock":
		service, data, err = m.mapLockAction(action)
	case "fan":
		service, data, err = m.mapFanAction(action)
	case "climate":
		service, data, err = m.mapClimateAction(action)
	case "media_player":
		service, data, err = m.mapMediaPlayerAction(action)
	case "camera":
		service, data, err = m.mapCameraAction(action)
	default:
		return "", "", nil, fmt.Errorf("unsupported domain: %s", domain)
	}

	if err != nil {
		return "", "", nil, err
	}

	return domain, service, data, nil
}

// Domain-specific action mappers

func (m *StateMapper) mapLightAction(action types.PMAControlAction) (string, map[string]interface{}, error) {
	data := make(map[string]interface{})

	switch action.Action {
	case "turn_on":
		// Handle brightness
		if brightness, ok := action.Parameters["brightness"]; ok {
			if b, ok := brightness.(float64); ok {
				// Convert 0-1 range to 0-255
				data["brightness"] = int(b * 255)
			} else if b, ok := brightness.(int); ok {
				data["brightness"] = b
			}
		}

		// Handle color
		if color, ok := action.Parameters["color"]; ok {
			if colorMap, ok := color.(map[string]interface{}); ok {
				if r, okR := colorMap["r"].(float64); okR {
					if g, okG := colorMap["g"].(float64); okG {
						if b, okB := colorMap["b"].(float64); okB {
							data["rgb_color"] = []int{int(r), int(g), int(b)}
						}
					}
				}
			}
		}

		// Handle color temperature
		if colorTemp, ok := action.Parameters["color_temp"]; ok {
			data["color_temp"] = colorTemp
		}

		return "turn_on", data, nil

	case "turn_off":
		return "turn_off", data, nil

	case "toggle":
		return "toggle", data, nil

	default:
		return "", nil, fmt.Errorf("unsupported light action: %s", action.Action)
	}
}

func (m *StateMapper) mapSwitchAction(action types.PMAControlAction) (string, map[string]interface{}, error) {
	data := make(map[string]interface{})

	switch action.Action {
	case "turn_on":
		return "turn_on", data, nil
	case "turn_off":
		return "turn_off", data, nil
	case "toggle":
		return "toggle", data, nil
	default:
		return "", nil, fmt.Errorf("unsupported switch action: %s", action.Action)
	}
}

func (m *StateMapper) mapCoverAction(action types.PMAControlAction) (string, map[string]interface{}, error) {
	data := make(map[string]interface{})

	switch action.Action {
	case "open":
		return "open_cover", data, nil
	case "close":
		return "close_cover", data, nil
	case "stop":
		return "stop_cover", data, nil
	case "set_position":
		if position, ok := action.Parameters["position"]; ok {
			data["position"] = position
		}
		return "set_cover_position", data, nil
	default:
		return "", nil, fmt.Errorf("unsupported cover action: %s", action.Action)
	}
}

func (m *StateMapper) mapLockAction(action types.PMAControlAction) (string, map[string]interface{}, error) {
	data := make(map[string]interface{})

	switch action.Action {
	case "lock":
		return "lock", data, nil
	case "unlock":
		return "unlock", data, nil
	default:
		return "", nil, fmt.Errorf("unsupported lock action: %s", action.Action)
	}
}

func (m *StateMapper) mapFanAction(action types.PMAControlAction) (string, map[string]interface{}, error) {
	data := make(map[string]interface{})

	switch action.Action {
	case "turn_on":
		// Handle speed
		if speed, ok := action.Parameters["speed"]; ok {
			data["percentage"] = speed
		}
		return "turn_on", data, nil
	case "turn_off":
		return "turn_off", data, nil
	case "set_speed":
		if speed, ok := action.Parameters["speed"]; ok {
			data["percentage"] = speed
		}
		return "set_percentage", data, nil
	default:
		return "", nil, fmt.Errorf("unsupported fan action: %s", action.Action)
	}
}

func (m *StateMapper) mapClimateAction(action types.PMAControlAction) (string, map[string]interface{}, error) {
	data := make(map[string]interface{})

	switch action.Action {
	case "set_temperature":
		if temp, ok := action.Parameters["temperature"]; ok {
			data["temperature"] = temp
		}
		return "set_temperature", data, nil
	case "set_hvac_mode":
		if mode, ok := action.Parameters["hvac_mode"]; ok {
			data["hvac_mode"] = mode
		}
		return "set_hvac_mode", data, nil
	case "set_fan_mode":
		if mode, ok := action.Parameters["fan_mode"]; ok {
			data["fan_mode"] = mode
		}
		return "set_fan_mode", data, nil
	default:
		return "", nil, fmt.Errorf("unsupported climate action: %s", action.Action)
	}
}

func (m *StateMapper) mapMediaPlayerAction(action types.PMAControlAction) (string, map[string]interface{}, error) {
	data := make(map[string]interface{})

	switch action.Action {
	case "play":
		return "media_play", data, nil
	case "pause":
		return "media_pause", data, nil
	case "stop":
		return "media_stop", data, nil
	case "next":
		return "media_next_track", data, nil
	case "previous":
		return "media_previous_track", data, nil
	case "set_volume":
		if volume, ok := action.Parameters["volume"]; ok {
			data["volume_level"] = volume
		}
		return "volume_set", data, nil
	case "mute":
		if mute, ok := action.Parameters["mute"]; ok {
			data["is_volume_muted"] = mute
		}
		return "volume_mute", data, nil
	default:
		return "", nil, fmt.Errorf("unsupported media player action: %s", action.Action)
	}
}

func (m *StateMapper) mapCameraAction(action types.PMAControlAction) (string, map[string]interface{}, error) {
	data := make(map[string]interface{})

	switch action.Action {
	case "snapshot":
		if filename, ok := action.Parameters["filename"]; ok {
			data["filename"] = filename
		}
		return "snapshot", data, nil
	case "record":
		if duration, ok := action.Parameters["duration"]; ok {
			data["duration"] = duration
		}
		if filename, ok := action.Parameters["filename"]; ok {
			data["filename"] = filename
		}
		return "record", data, nil
	default:
		return "", nil, fmt.Errorf("unsupported camera action: %s", action.Action)
	}
}

// MapPMAStateToHA maps PMA state back to HomeAssistant format
func (m *StateMapper) MapPMAStateToHA(pmaState types.PMAEntityState, domain string) string {
	switch domain {
	case "light", "switch", "fan":
		switch pmaState {
		case types.StateOn:
			return "on"
		case types.StateOff:
			return "off"
		case types.StateUnavailable:
			return "unavailable"
		default:
			return "unknown"
		}
	case "cover":
		switch pmaState {
		case types.StateOpen:
			return "open"
		case types.StateClosed:
			return "closed"
		case types.StateActive:
			return "opening" // Could be opening or closing
		case types.StateUnavailable:
			return "unavailable"
		default:
			return "unknown"
		}
	case "lock":
		switch pmaState {
		case types.StateLocked:
			return "locked"
		case types.StateUnlocked:
			return "unlocked"
		case types.StateUnavailable:
			return "unavailable"
		default:
			return "unknown"
		}
	case "binary_sensor":
		switch pmaState {
		case types.StateActive:
			return "on"
		case types.StateIdle:
			return "off"
		case types.StateUnavailable:
			return "unavailable"
		default:
			return "unknown"
		}
	case "media_player":
		switch pmaState {
		case types.StateActive:
			return "playing"
		case types.StateIdle:
			return "paused"
		case types.StateOff:
			return "off"
		case types.StateUnavailable:
			return "unavailable"
		default:
			return "unknown"
		}
	default:
		if pmaState == types.StateUnavailable {
			return "unavailable"
		}
		return "unknown"
	}
}

// NormalizeValue normalizes attribute values for consistency
func (m *StateMapper) NormalizeValue(key string, value interface{}) interface{} {
	switch key {
	case "brightness":
		// Normalize brightness to 0-1 range
		if brightness, ok := value.(float64); ok {
			if brightness > 1.0 {
				return brightness / 255.0 // Convert from 0-255 to 0-1
			}
			return brightness
		}
		if brightness, ok := value.(int); ok {
			if brightness > 1 {
				return float64(brightness) / 255.0 // Convert from 0-255 to 0-1
			}
			return float64(brightness)
		}
	case "temperature":
		// Ensure temperature is a float
		if temp, ok := value.(string); ok {
			if parsed, err := strconv.ParseFloat(temp, 64); err == nil {
				return parsed
			}
		}
	case "position":
		// Normalize position to 0-1 range
		if pos, ok := value.(float64); ok {
			if pos > 1.0 {
				return pos / 100.0 // Convert from 0-100 to 0-1
			}
			return pos
		}
		if pos, ok := value.(int); ok {
			if pos > 1 {
				return float64(pos) / 100.0 // Convert from 0-100 to 0-1
			}
			return float64(pos)
		}
	}

	return value
}

// GetAvailableActions returns available actions for a given domain
func (m *StateMapper) GetAvailableActions(domain string) []string {
	switch domain {
	case "light":
		return []string{"turn_on", "turn_off", "toggle"}
	case "switch":
		return []string{"turn_on", "turn_off", "toggle"}
	case "cover":
		return []string{"open", "close", "stop", "set_position"}
	case "lock":
		return []string{"lock", "unlock"}
	case "fan":
		return []string{"turn_on", "turn_off", "set_speed"}
	case "climate":
		return []string{"set_temperature", "set_hvac_mode", "set_fan_mode"}
	case "media_player":
		return []string{"play", "pause", "stop", "next", "previous", "set_volume", "mute"}
	case "camera":
		return []string{"snapshot", "record"}
	default:
		return []string{}
	}
}

// Helper methods

func (m *StateMapper) convertPMAEntityIDToHA(pmaEntityID string) string {
	// Remove "ha_" prefix if present
	if len(pmaEntityID) > 3 && pmaEntityID[:3] == "ha_" {
		return pmaEntityID[3:]
	}
	return pmaEntityID
}

// ValidateAction validates that an action is supported for the given entity
func (m *StateMapper) ValidateAction(action types.PMAControlAction) error {
	// Extract domain from entity ID
	haEntityID := m.convertPMAEntityIDToHA(action.EntityID)
	parts := strings.Split(haEntityID, ".")
	if len(parts) != 2 {
		return fmt.Errorf("invalid entity ID format: %s", haEntityID)
	}
	domain := parts[0]

	// Check if action is supported for this domain
	availableActions := m.GetAvailableActions(domain)
	for _, availableAction := range availableActions {
		if availableAction == action.Action {
			return nil // Action is valid
		}
	}

	return fmt.Errorf("action '%s' is not supported for domain '%s'", action.Action, domain)
} 