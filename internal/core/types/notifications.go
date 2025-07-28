package types

import (
	"fmt"
	"time"
)

// NotificationType represents the type of notification
type NotificationType string

const (
	// Device discovery and configuration notifications
	NotificationTypeDeviceDiscovery      NotificationType = "device_discovery"
	NotificationTypeDeviceConfirmation   NotificationType = "device_confirmation"
	NotificationTypeConfigurationReady   NotificationType = "configuration_ready"
	NotificationTypeConfigurationUpdate  NotificationType = "configuration_update"
	NotificationTypeConfigurationSuccess NotificationType = "configuration_success"
	NotificationTypeConfigurationFailed  NotificationType = "configuration_failed"

	// System notifications
	NotificationTypeSystemAlert   NotificationType = "system_alert"
	NotificationTypeSystemWarning NotificationType = "system_warning"
	NotificationTypeSystemInfo    NotificationType = "system_info"

	// Automation notifications
	NotificationTypeAutomationTriggered NotificationType = "automation_triggered"
	NotificationTypeAutomationFailed    NotificationType = "automation_failed"

	// Security notifications
	NotificationTypeSecurityAlert   NotificationType = "security_alert"
	NotificationTypeSecurityWarning NotificationType = "security_warning"
)

// NotificationPriority represents the priority level of a notification
type NotificationPriority string

const (
	NotificationPriorityLow      NotificationPriority = "low"
	NotificationPriorityMedium   NotificationPriority = "medium"
	NotificationPriorityHigh     NotificationPriority = "high"
	NotificationPriorityCritical NotificationPriority = "critical"
)

// ActionType represents the type of action that can be taken
type ActionType string

const (
	ActionTypeButton   ActionType = "button"
	ActionTypeInput    ActionType = "input"
	ActionTypeSelect   ActionType = "select"
	ActionTypeConfirm  ActionType = "confirm"
	ActionTypeCancel   ActionType = "cancel"
	ActionTypeRedirect ActionType = "redirect"
)

// AINotification represents a notification to be sent through AI channels
type AINotification struct {
	ID             string                 `json:"id"`
	Type           NotificationType       `json:"type"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Priority       NotificationPriority   `json:"priority"`
	Timestamp      time.Time              `json:"timestamp"`
	ExpiresAt      *time.Time             `json:"expires_at,omitempty"`
	Data           map[string]interface{} `json:"data,omitempty"`
	Actions        []NotificationAction   `json:"actions,omitempty"`
	Categories     []string               `json:"categories,omitempty"`
	Source         string                 `json:"source,omitempty"`
	UserID         string                 `json:"user_id,omitempty"`
	RoomID         string                 `json:"room_id,omitempty"`
	DeviceID       string                 `json:"device_id,omitempty"`
	Acknowledged   bool                   `json:"acknowledged"`
	AcknowledgedAt *time.Time             `json:"acknowledged_at,omitempty"`
	AcknowledgedBy string                 `json:"acknowledged_by,omitempty"`
}

// NotificationAction represents an action that can be taken on a notification
type NotificationAction struct {
	ID         string                 `json:"id"`
	Label      string                 `json:"label"`
	Type       ActionType             `json:"type"`
	Data       map[string]interface{} `json:"data,omitempty"`
	URL        string                 `json:"url,omitempty"`
	Confirm    bool                   `json:"confirm,omitempty"`
	Icon       string                 `json:"icon,omitempty"`
	Style      string                 `json:"style,omitempty"` // primary, secondary, danger, success
	Disabled   bool                   `json:"disabled,omitempty"`
	Validation *ActionValidation      `json:"validation,omitempty"`
}

// ActionValidation represents validation rules for action inputs
type ActionValidation struct {
	Required  bool     `json:"required,omitempty"`
	MinLength int      `json:"min_length,omitempty"`
	MaxLength int      `json:"max_length,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Options   []string `json:"options,omitempty"` // For select type actions
}

// NotificationChannel represents where notifications should be sent
type NotificationChannel string

const (
	NotificationChannelWebSocket NotificationChannel = "websocket"
	NotificationChannelChat      NotificationChannel = "chat"
	NotificationChannelEmail     NotificationChannel = "email"
	NotificationChannelSMS       NotificationChannel = "sms"
	NotificationChannelPush      NotificationChannel = "push"
	NotificationChannelDesktop   NotificationChannel = "desktop"
	NotificationChannelMobile    NotificationChannel = "mobile"
)

// NotificationPreferences represents user preferences for notifications
type NotificationPreferences struct {
	UserID              string                        `json:"user_id"`
	EnabledChannels     []NotificationChannel         `json:"enabled_channels"`
	PriorityFilters     map[NotificationPriority]bool `json:"priority_filters"`
	TypeFilters         map[NotificationType]bool     `json:"type_filters"`
	QuietHours          *QuietHours                   `json:"quiet_hours,omitempty"`
	CategoryPreferences map[string]bool               `json:"category_preferences,omitempty"`
	DeviceFilters       []string                      `json:"device_filters,omitempty"`
	RoomFilters         []string                      `json:"room_filters,omitempty"`
}

// QuietHours represents times when notifications should be suppressed
type QuietHours struct {
	Enabled   bool     `json:"enabled"`
	StartTime string   `json:"start_time"` // HH:MM format
	EndTime   string   `json:"end_time"`   // HH:MM format
	Days      []string `json:"days"`       // monday, tuesday, etc.
	Timezone  string   `json:"timezone"`
}

// NotificationResponse represents a user's response to a notification action
type NotificationResponse struct {
	NotificationID string                 `json:"notification_id"`
	ActionID       string                 `json:"action_id"`
	UserID         string                 `json:"user_id"`
	Response       map[string]interface{} `json:"response"`
	Timestamp      time.Time              `json:"timestamp"`
}

// DeviceDiscoveryNotificationData represents data for device discovery notifications
type DeviceDiscoveryNotificationData struct {
	DeviceMAC       string                 `json:"device_mac"`
	DeviceModel     string                 `json:"device_model"`
	DeviceName      string                 `json:"device_name"`
	DeviceIP        string                 `json:"device_ip"`
	WiFiMode        string                 `json:"wifi_mode"`
	RequiresConfig  bool                   `json:"requires_config"`
	DiscoveryMethod string                 `json:"discovery_method"`
	Capabilities    []string               `json:"capabilities"`
	DeviceInfo      map[string]interface{} `json:"device_info,omitempty"`
}

// ConfigurationSessionNotificationData represents data for configuration session notifications
type ConfigurationSessionNotificationData struct {
	SessionID      string           `json:"session_id"`
	DeviceMAC      string           `json:"device_mac"`
	DeviceName     string           `json:"device_name"`
	ConfigFlow     string           `json:"config_flow"`
	CurrentStep    string           `json:"current_step"`
	Progress       float64          `json:"progress"` // 0.0 to 1.0
	EstimatedTime  *time.Duration   `json:"estimated_time,omitempty"`
	Steps          []ConfigStepData `json:"steps,omitempty"`
	Error          string           `json:"error,omitempty"`
	ConfigURL      string           `json:"config_url,omitempty"`
	ConversationID string           `json:"conversation_id,omitempty"`
}

// ConfigStepData represents data for a configuration step
type ConfigStepData struct {
	Step        string                 `json:"step"`
	Status      string                 `json:"status"`
	Description string                 `json:"description,omitempty"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

// CreateDeviceDiscoveryNotification creates a notification for a newly discovered device
func CreateDeviceDiscoveryNotification(device DeviceDiscoveryNotificationData) *AINotification {
	priority := NotificationPriorityMedium
	if device.RequiresConfig {
		priority = NotificationPriorityHigh
	}

	message := fmt.Sprintf("Discovered %s device (%s)", device.DeviceModel, device.DeviceName)
	if device.RequiresConfig {
		message += " that needs configuration"
	}

	actions := []NotificationAction{}

	if device.RequiresConfig {
		actions = append(actions,
			NotificationAction{
				ID:    "configure_manual",
				Label: "Configure Manually",
				Type:  ActionTypeButton,
				Style: "primary",
				Data: map[string]interface{}{
					"device_mac": device.DeviceMAC,
					"flow":       "manual",
				},
			},
			NotificationAction{
				ID:    "configure_ai",
				Label: "Configure with AI",
				Type:  ActionTypeButton,
				Style: "secondary",
				Data: map[string]interface{}{
					"device_mac": device.DeviceMAC,
					"flow":       "ai_assisted",
				},
			},
		)
	}

	actions = append(actions,
		NotificationAction{
			ID:    "view_details",
			Label: "View Details",
			Type:  ActionTypeButton,
			Style: "secondary",
			Data: map[string]interface{}{
				"device_mac": device.DeviceMAC,
			},
		},
		NotificationAction{
			ID:    "ignore",
			Label: "Ignore",
			Type:  ActionTypeButton,
			Style: "secondary",
			Data: map[string]interface{}{
				"device_mac": device.DeviceMAC,
			},
		},
	)

	return &AINotification{
		Type:      NotificationTypeDeviceDiscovery,
		Title:     "New Shelly Device Discovered",
		Message:   message,
		Priority:  priority,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"device": device,
		},
		Actions:    actions,
		Categories: []string{"devices", "shelly", "discovery"},
		Source:     "shelly_autoconfig",
		DeviceID:   device.DeviceMAC,
	}
}

// CreateConfigurationUpdateNotification creates a notification for configuration progress updates
func CreateConfigurationUpdateNotification(session ConfigurationSessionNotificationData) *AINotification {
	priority := NotificationPriorityMedium
	if session.Error != "" {
		priority = NotificationPriorityHigh
	}

	var title, message string
	var notificationType NotificationType

	if session.Error != "" {
		notificationType = NotificationTypeConfigurationFailed
		title = "Device Configuration Failed"
		message = fmt.Sprintf("Configuration of %s failed: %s", session.DeviceName, session.Error)
	} else if session.Progress >= 1.0 {
		notificationType = NotificationTypeConfigurationSuccess
		title = "Device Configuration Complete"
		message = fmt.Sprintf("Successfully configured %s", session.DeviceName)
		priority = NotificationPriorityHigh
	} else {
		notificationType = NotificationTypeConfigurationUpdate
		title = "Device Configuration in Progress"
		message = fmt.Sprintf("Configuring %s - %s (%.0f%% complete)",
			session.DeviceName, session.CurrentStep, session.Progress*100)
	}

	actions := []NotificationAction{}

	if session.Error != "" {
		actions = append(actions,
			NotificationAction{
				ID:    "retry_configuration",
				Label: "Retry",
				Type:  ActionTypeButton,
				Style: "primary",
				Data: map[string]interface{}{
					"session_id": session.SessionID,
					"device_mac": session.DeviceMAC,
				},
			},
			NotificationAction{
				ID:      "cancel_configuration",
				Label:   "Cancel",
				Type:    ActionTypeButton,
				Style:   "danger",
				Confirm: true,
				Data: map[string]interface{}{
					"session_id": session.SessionID,
				},
			},
		)
	} else if session.Progress < 1.0 {
		actions = append(actions,
			NotificationAction{
				ID:    "view_progress",
				Label: "View Progress",
				Type:  ActionTypeButton,
				Style: "secondary",
				Data: map[string]interface{}{
					"session_id": session.SessionID,
				},
			},
		)

		if session.ConfigFlow == "manual" {
			actions = append(actions,
				NotificationAction{
					ID:    "continue_manual",
					Label: "Continue Configuration",
					Type:  ActionTypeRedirect,
					URL:   session.ConfigURL,
					Style: "primary",
				},
			)
		}
	}

	return &AINotification{
		Type:      notificationType,
		Title:     title,
		Message:   message,
		Priority:  priority,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"session": session,
		},
		Actions:    actions,
		Categories: []string{"devices", "shelly", "configuration"},
		Source:     "shelly_autoconfig",
		DeviceID:   session.DeviceMAC,
	}
}
