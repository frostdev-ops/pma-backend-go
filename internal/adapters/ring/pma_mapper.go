package ring

import (
	"context"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
)

// mapPMAActionToRingAction maps PMA control actions to Ring-specific actions
func (a *RingAdapter) mapPMAActionToRingAction(action types.PMAControlAction) (ringAction string, params map[string]interface{}, err error) {
	switch action.Action {
	case "get_snapshot":
		return "get_snapshot", nil, nil
	case "get_live_stream":
		return "get_live_stream", nil, nil
	case "trigger_siren":
		return "trigger_siren", map[string]interface{}{
			"enabled": true,
		}, nil
	case "stop_siren":
		return "stop_siren", map[string]interface{}{
			"enabled": false,
		}, nil
	case "turn_on_lights":
		return "turn_on_lights", nil, nil
	case "turn_off_lights":
		return "turn_off_lights", nil, nil
	default:
		return "", nil, fmt.Errorf("unsupported action: %s", action.Action)
	}
}

// executeRingAction executes Ring-specific actions on devices
func (a *RingAdapter) executeRingAction(ctx context.Context, deviceID, action string, params map[string]interface{}) (*types.PMAControlResult, error) {
	// Get the device
	device, err := a.GetDevice(deviceID)
	if err != nil {
		return &types.PMAControlResult{
			Success:     false,
			EntityID:    deviceID,
			Action:      action,
			ProcessedAt: time.Now(),
			Error: &types.PMAError{
				Code:     "DEVICE_NOT_FOUND",
				Message:  err.Error(),
				Source:   "ring",
				EntityID: deviceID,
			},
		}, nil
	}

	startTime := time.Now()
	var result *types.PMAControlResult

	// Execute action based on device type and action
	switch d := device.(type) {
	case *RingDoorbell:
		result = a.executeDoorbellAction(ctx, d, action, params)
	case *RingCamera:
		result = a.executeCameraAction(ctx, d, action, params)
	case *RingChime:
		result = a.executeChimeAction(ctx, d, action, params)
	default:
		result = &types.PMAControlResult{
			Success:     false,
			EntityID:    deviceID,
			Action:      action,
			ProcessedAt: time.Now(),
			Error: &types.PMAError{
				Code:     "UNSUPPORTED_DEVICE_TYPE",
				Message:  fmt.Sprintf("unsupported device type: %T", device),
				Source:   "ring",
				EntityID: deviceID,
			},
		}
	}

	result.Duration = time.Since(startTime)
	return result, nil
}

// executeDoorbellAction executes actions specific to Ring doorbells
func (a *RingAdapter) executeDoorbellAction(ctx context.Context, doorbell *RingDoorbell, action string, params map[string]interface{}) *types.PMAControlResult {
	deviceIDStr := fmt.Sprintf("%d", doorbell.deviceData.ID)

	switch action {
	case "get_snapshot":
		url, err := a.client.GetSnapshot(ctx, deviceIDStr)
		if err != nil {
			return &types.PMAControlResult{
				Success:     false,
				EntityID:    doorbell.GetID(),
				Action:      action,
				ProcessedAt: time.Now(),
				Error: &types.PMAError{
					Code:     "SNAPSHOT_FAILED",
					Message:  err.Error(),
					Source:   "ring",
					EntityID: doorbell.GetID(),
				},
			}
		}
		return &types.PMAControlResult{
			Success:     true,
			EntityID:    doorbell.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Attributes: map[string]interface{}{
				"snapshot_url": url,
			},
		}

	case "get_live_stream":
		url, err := a.client.GetLiveStreamURL(ctx, deviceIDStr)
		if err != nil {
			return &types.PMAControlResult{
				Success:     false,
				EntityID:    doorbell.GetID(),
				Action:      action,
				ProcessedAt: time.Now(),
				Error: &types.PMAError{
					Code:     "STREAM_FAILED",
					Message:  err.Error(),
					Source:   "ring",
					EntityID: doorbell.GetID(),
				},
			}
		}
		return &types.PMAControlResult{
			Success:     true,
			EntityID:    doorbell.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Attributes: map[string]interface{}{
				"stream_url": url,
			},
		}

	case "trigger_siren":
		err := a.client.SetSiren(ctx, deviceIDStr, true)
		if err != nil {
			return &types.PMAControlResult{
				Success:     false,
				EntityID:    doorbell.GetID(),
				Action:      action,
				ProcessedAt: time.Now(),
				Error: &types.PMAError{
					Code:     "SIREN_FAILED",
					Message:  err.Error(),
					Source:   "ring",
					EntityID: doorbell.GetID(),
				},
			}
		}
		return &types.PMAControlResult{
			Success:     true,
			EntityID:    doorbell.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Attributes: map[string]interface{}{
				"siren_enabled": true,
			},
		}

	case "stop_siren":
		err := a.client.SetSiren(ctx, deviceIDStr, false)
		if err != nil {
			return &types.PMAControlResult{
				Success:     false,
				EntityID:    doorbell.GetID(),
				Action:      action,
				ProcessedAt: time.Now(),
				Error: &types.PMAError{
					Code:     "SIREN_FAILED",
					Message:  err.Error(),
					Source:   "ring",
					EntityID: doorbell.GetID(),
				},
			}
		}
		return &types.PMAControlResult{
			Success:     true,
			EntityID:    doorbell.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Attributes: map[string]interface{}{
				"siren_enabled": false,
			},
		}

	case "turn_on_lights":
		err := a.client.SetLights(ctx, deviceIDStr, true)
		if err != nil {
			return &types.PMAControlResult{
				Success:     false,
				EntityID:    doorbell.GetID(),
				Action:      action,
				ProcessedAt: time.Now(),
				Error: &types.PMAError{
					Code:     "LIGHTS_FAILED",
					Message:  err.Error(),
					Source:   "ring",
					EntityID: doorbell.GetID(),
				},
			}
		}
		return &types.PMAControlResult{
			Success:     true,
			EntityID:    doorbell.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Attributes: map[string]interface{}{
				"lights_enabled": true,
			},
		}

	case "turn_off_lights":
		err := a.client.SetLights(ctx, deviceIDStr, false)
		if err != nil {
			return &types.PMAControlResult{
				Success:     false,
				EntityID:    doorbell.GetID(),
				Action:      action,
				ProcessedAt: time.Now(),
				Error: &types.PMAError{
					Code:     "LIGHTS_FAILED",
					Message:  err.Error(),
					Source:   "ring",
					EntityID: doorbell.GetID(),
				},
			}
		}
		return &types.PMAControlResult{
			Success:     true,
			EntityID:    doorbell.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Attributes: map[string]interface{}{
				"lights_enabled": false,
			},
		}

	default:
		return &types.PMAControlResult{
			Success:     false,
			EntityID:    doorbell.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Error: &types.PMAError{
				Code:     "UNSUPPORTED_ACTION",
				Message:  fmt.Sprintf("action %s not supported for doorbell", action),
				Source:   "ring",
				EntityID: doorbell.GetID(),
			},
		}
	}
}

// executeCameraAction executes actions specific to Ring cameras
func (a *RingAdapter) executeCameraAction(ctx context.Context, camera *RingCamera, action string, params map[string]interface{}) *types.PMAControlResult {
	deviceIDStr := fmt.Sprintf("%d", camera.deviceData.ID)

	switch action {
	case "get_snapshot":
		url, err := a.client.GetSnapshot(ctx, deviceIDStr)
		if err != nil {
			return &types.PMAControlResult{
				Success:     false,
				EntityID:    camera.GetID(),
				Action:      action,
				ProcessedAt: time.Now(),
				Error: &types.PMAError{
					Code:     "SNAPSHOT_FAILED",
					Message:  err.Error(),
					Source:   "ring",
					EntityID: camera.GetID(),
				},
			}
		}
		return &types.PMAControlResult{
			Success:     true,
			EntityID:    camera.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Attributes: map[string]interface{}{
				"snapshot_url": url,
			},
		}

	case "get_live_stream":
		url, err := a.client.GetLiveStreamURL(ctx, deviceIDStr)
		if err != nil {
			return &types.PMAControlResult{
				Success:     false,
				EntityID:    camera.GetID(),
				Action:      action,
				ProcessedAt: time.Now(),
				Error: &types.PMAError{
					Code:     "STREAM_FAILED",
					Message:  err.Error(),
					Source:   "ring",
					EntityID: camera.GetID(),
				},
			}
		}
		return &types.PMAControlResult{
			Success:     true,
			EntityID:    camera.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Attributes: map[string]interface{}{
				"stream_url": url,
			},
		}

	case "turn_on_lights":
		err := a.client.SetLights(ctx, deviceIDStr, true)
		if err != nil {
			return &types.PMAControlResult{
				Success:     false,
				EntityID:    camera.GetID(),
				Action:      action,
				ProcessedAt: time.Now(),
				Error: &types.PMAError{
					Code:     "LIGHTS_FAILED",
					Message:  err.Error(),
					Source:   "ring",
					EntityID: camera.GetID(),
				},
			}
		}
		return &types.PMAControlResult{
			Success:     true,
			EntityID:    camera.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Attributes: map[string]interface{}{
				"lights_enabled": true,
			},
		}

	case "turn_off_lights":
		err := a.client.SetLights(ctx, deviceIDStr, false)
		if err != nil {
			return &types.PMAControlResult{
				Success:     false,
				EntityID:    camera.GetID(),
				Action:      action,
				ProcessedAt: time.Now(),
				Error: &types.PMAError{
					Code:     "LIGHTS_FAILED",
					Message:  err.Error(),
					Source:   "ring",
					EntityID: camera.GetID(),
				},
			}
		}
		return &types.PMAControlResult{
			Success:     true,
			EntityID:    camera.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Attributes: map[string]interface{}{
				"lights_enabled": false,
			},
		}

	default:
		return &types.PMAControlResult{
			Success:     false,
			EntityID:    camera.GetID(),
			Action:      action,
			ProcessedAt: time.Now(),
			Error: &types.PMAError{
				Code:     "UNSUPPORTED_ACTION",
				Message:  fmt.Sprintf("action %s not supported for camera", action),
				Source:   "ring",
				EntityID: camera.GetID(),
			},
		}
	}
}

// executeChimeAction executes actions specific to Ring chimes
func (a *RingAdapter) executeChimeAction(ctx context.Context, chime *RingChime, action string, params map[string]interface{}) *types.PMAControlResult {
	// Ring chimes don't have much control via API - mostly just status information
	return &types.PMAControlResult{
		Success:     false,
		EntityID:    chime.GetID(),
		Action:      action,
		ProcessedAt: time.Now(),
		Error: &types.PMAError{
			Code:     "CHIME_CONTROL_NOT_SUPPORTED",
			Message:  "Chime control not supported via current API",
			Source:   "ring",
			EntityID: chime.GetID(),
		},
	}
}

// getSupportedActionsForDevice returns supported actions for Ring devices
func (a *RingAdapter) getSupportedActionsForDevice(device interface{}) []string {
	switch device.(type) {
	case *RingDoorbell:
		return []string{
			"get_snapshot",
			"get_live_stream",
			"trigger_siren",
			"stop_siren",
			"turn_on_lights",
			"turn_off_lights",
		}
	case *RingCamera:
		return []string{
			"get_snapshot",
			"get_live_stream",
			"turn_on_lights",
			"turn_off_lights",
		}
	case *RingChime:
		return []string{
			// Limited control available for chimes
		}
	default:
		return []string{}
	}
}
