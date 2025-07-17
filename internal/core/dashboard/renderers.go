package dashboard

import (
	"fmt"
	"time"
)

// WelcomeRenderer renders welcome widgets
type WelcomeRenderer struct{}

// RenderWidget renders the welcome widget
func (r *WelcomeRenderer) RenderWidget(widget *Widget, context RenderContext) (*WidgetData, error) {
	message := "Welcome to PMA Home Control"
	if customMessage, exists := widget.Config["message"]; exists {
		if msg, ok := customMessage.(string); ok && msg != "" {
			message = msg
		}
	}

	data := map[string]interface{}{
		"message": message,
		"user_id": context.UserID,
		"time":    time.Now().Format("15:04:05"),
		"actions": []map[string]interface{}{
			{"label": "Dashboard", "action": "navigate", "target": "/dashboard"},
			{"label": "Devices", "action": "navigate", "target": "/devices"},
			{"label": "Settings", "action": "navigate", "target": "/settings"},
		},
	}

	return &WidgetData{
		WidgetID:    widget.ID,
		Type:        widget.Type,
		Title:       widget.Title,
		Data:        data,
		Metadata:    map[string]interface{}{"version": "1.0"},
		LastUpdated: time.Now(),
		Status:      WidgetStatusReady,
	}, nil
}

// ValidateConfig validates the welcome widget configuration
func (r *WelcomeRenderer) ValidateConfig(config map[string]interface{}) error {
	if message, exists := config["message"]; exists {
		if _, ok := message.(string); !ok {
			return fmt.Errorf("message must be a string")
		}
	}
	return nil
}

// GetDefaultConfig returns default configuration for welcome widget
func (r *WelcomeRenderer) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"message": "Welcome to PMA Home Control",
	}
}

// SupportsRefresh indicates if the widget supports refresh
func (r *WelcomeRenderer) SupportsRefresh() bool {
	return false
}

// GetRefreshRate returns the refresh rate for the widget
func (r *WelcomeRenderer) GetRefreshRate() time.Duration {
	return 0
}

// DeviceControlRenderer renders device control widgets
type DeviceControlRenderer struct{}

// RenderWidget renders the device control widget
func (r *DeviceControlRenderer) RenderWidget(widget *Widget, context RenderContext) (*WidgetData, error) {
	deviceID, exists := widget.Config["device_id"]
	if !exists {
		return nil, fmt.Errorf("device_id is required")
	}

	deviceIDStr, ok := deviceID.(string)
	if !ok {
		return nil, fmt.Errorf("device_id must be a string")
	}

	showPower := true
	if sp, exists := widget.Config["show_power"]; exists {
		if spBool, ok := sp.(bool); ok {
			showPower = spBool
		}
	}

	// Mock device data - in real implementation, this would fetch from device manager
	data := map[string]interface{}{
		"device_id":   deviceIDStr,
		"name":        fmt.Sprintf("Device %s", deviceIDStr),
		"state":       "on",
		"brightness":  75,
		"power_usage": 45.2,
		"show_power":  showPower,
		"controls": []map[string]interface{}{
			{"type": "toggle", "label": "Power", "value": true},
			{"type": "slider", "label": "Brightness", "value": 75, "min": 0, "max": 100},
		},
	}

	return &WidgetData{
		WidgetID:    widget.ID,
		Type:        widget.Type,
		Title:       widget.Title,
		Data:        data,
		Metadata:    map[string]interface{}{"device_id": deviceIDStr},
		LastUpdated: time.Now(),
		Status:      WidgetStatusReady,
	}, nil
}

// ValidateConfig validates the device control widget configuration
func (r *DeviceControlRenderer) ValidateConfig(config map[string]interface{}) error {
	deviceID, exists := config["device_id"]
	if !exists {
		return fmt.Errorf("device_id is required")
	}

	if _, ok := deviceID.(string); !ok {
		return fmt.Errorf("device_id must be a string")
	}

	if showPower, exists := config["show_power"]; exists {
		if _, ok := showPower.(bool); !ok {
			return fmt.Errorf("show_power must be a boolean")
		}
	}

	return nil
}

// GetDefaultConfig returns default configuration for device control widget
func (r *DeviceControlRenderer) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"device_id":  "",
		"show_power": true,
	}
}

// SupportsRefresh indicates if the widget supports refresh
func (r *DeviceControlRenderer) SupportsRefresh() bool {
	return true
}

// GetRefreshRate returns the refresh rate for the widget
func (r *DeviceControlRenderer) GetRefreshRate() time.Duration {
	return 30 * time.Second
}

// SystemStatusRenderer renders system status widgets
type SystemStatusRenderer struct{}

// RenderWidget renders the system status widget
func (r *SystemStatusRenderer) RenderWidget(widget *Widget, context RenderContext) (*WidgetData, error) {
	metrics := []string{"cpu", "memory", "disk"}
	if configMetrics, exists := widget.Config["metrics"]; exists {
		if metricsSlice, ok := configMetrics.([]interface{}); ok {
			metrics = make([]string, len(metricsSlice))
			for i, m := range metricsSlice {
				if metricStr, ok := m.(string); ok {
					metrics[i] = metricStr
				}
			}
		}
	}

	// Mock system data - in real implementation, this would fetch from system monitor
	data := map[string]interface{}{
		"metrics": metrics,
		"status": map[string]interface{}{
			"cpu": map[string]interface{}{
				"usage":       65.4,
				"cores":       4,
				"temperature": 58.2,
			},
			"memory": map[string]interface{}{
				"usage":     72.1,
				"total":     "8GB",
				"available": "2.2GB",
			},
			"disk": map[string]interface{}{
				"usage": 45.8,
				"total": "128GB",
				"free":  "69GB",
			},
			"network": map[string]interface{}{
				"download": "125.4 KB/s",
				"upload":   "45.2 KB/s",
			},
			"temperature": map[string]interface{}{
				"cpu":  58.2,
				"case": 42.1,
			},
		},
		"uptime":    "2d 14h 32m",
		"timestamp": time.Now().Unix(),
	}

	return &WidgetData{
		WidgetID:    widget.ID,
		Type:        widget.Type,
		Title:       widget.Title,
		Data:        data,
		Metadata:    map[string]interface{}{"metrics": metrics},
		LastUpdated: time.Now(),
		Status:      WidgetStatusReady,
	}, nil
}

// ValidateConfig validates the system status widget configuration
func (r *SystemStatusRenderer) ValidateConfig(config map[string]interface{}) error {
	if metrics, exists := config["metrics"]; exists {
		if metricsSlice, ok := metrics.([]interface{}); ok {
			validMetrics := map[string]bool{
				"cpu": true, "memory": true, "disk": true,
				"network": true, "temperature": true,
			}

			for _, m := range metricsSlice {
				if metricStr, ok := m.(string); ok {
					if !validMetrics[metricStr] {
						return fmt.Errorf("invalid metric: %s", metricStr)
					}
				}
			}
		} else {
			return fmt.Errorf("metrics must be an array of strings")
		}
	}

	return nil
}

// GetDefaultConfig returns default configuration for system status widget
func (r *SystemStatusRenderer) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"metrics": []string{"cpu", "memory", "disk"},
	}
}

// SupportsRefresh indicates if the widget supports refresh
func (r *SystemStatusRenderer) SupportsRefresh() bool {
	return true
}

// GetRefreshRate returns the refresh rate for the widget
func (r *SystemStatusRenderer) GetRefreshRate() time.Duration {
	return 5 * time.Second
}
