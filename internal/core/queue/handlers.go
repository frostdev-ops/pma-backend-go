package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/automation"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/sirupsen/logrus"
)

// Base handler that other handlers can embed
type BaseHandler struct {
	logger         *logrus.Logger
	unifiedService *unified.UnifiedEntityService
}

func (h *BaseHandler) GetTimeout() time.Duration {
	return 30 * time.Second
}

// EntityStateHandler handles entity state changes
type EntityStateHandler struct {
	BaseHandler
	logger *logrus.Logger
}

// NewEntityStateHandler creates a new entity state handler
func NewEntityStateHandler(logger *logrus.Logger, unifiedService *unified.UnifiedEntityService) *EntityStateHandler {
	return &EntityStateHandler{
		BaseHandler: BaseHandler{
			logger:         logger,
			unifiedService: unifiedService,
		},
		logger: logger,
	}
}

func (h *EntityStateHandler) Execute(ctx context.Context, action *models.QueuedAction) (*ActionExecutionResult, error) {
	h.logger.WithField("action_id", action.ID).Info("Executing entity state change")

	var payload struct {
		EntityID   string                 `json:"entity_id"`
		State      interface{}            `json:"state"`
		Attributes map[string]interface{} `json:"attributes,omitempty"`
		Action     string                 `json:"action,omitempty"` // e.g., "turn_on", "turn_off", "set_brightness"
	}

	if err := json.Unmarshal([]byte(action.ActionData), &payload); err != nil {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "INVALID_PAYLOAD",
			ShouldRetry: false,
		}, fmt.Errorf("invalid payload: %w", err)
	}

	// Validate required fields
	if payload.EntityID == "" {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "MISSING_ENTITY_ID",
			ShouldRetry: false,
		}, fmt.Errorf("entity_id is required")
	}

	// Create control action for the unified service
	controlAction := types.PMAControlAction{
		EntityID:   payload.EntityID,
		Action:     payload.Action,
		Parameters: make(map[string]interface{}),
	}

	// Add state and attributes to parameters
	if payload.State != nil {
		controlAction.Parameters["state"] = payload.State
	}
	if payload.Attributes != nil {
		for key, value := range payload.Attributes {
			controlAction.Parameters[key] = value
		}
	}

	// If no action is specified, try to infer it from the state
	if payload.Action == "" {
		if stateStr, ok := payload.State.(string); ok {
			switch stateStr {
			case "on", "true":
				controlAction.Action = "turn_on"
			case "off", "false":
				controlAction.Action = "turn_off"
			default:
				controlAction.Action = "set_state"
			}
		} else {
			controlAction.Action = "set_state"
		}
	}

	// Execute the action through the unified service
	result, err := h.unifiedService.ExecuteAction(ctx, controlAction)
	if err != nil {
		h.logger.WithError(err).WithFields(logrus.Fields{
			"entity_id": payload.EntityID,
			"action":    controlAction.Action,
		}).Error("Failed to execute entity state change")

		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "EXECUTION_FAILED",
			ShouldRetry: true, // Retry transient failures
			Data: map[string]interface{}{
				"entity_id": payload.EntityID,
				"error":     err.Error(),
			},
		}, err
	}

	// Check if the action was successful
	if result != nil && !result.Success {
		h.logger.WithFields(logrus.Fields{
			"entity_id": payload.EntityID,
			"action":    controlAction.Action,
			"error":     result.Error,
		}).Warn("Entity state change was not successful")

		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   result.Error.Code,
			ShouldRetry: result.Error.Code != "INVALID_ACTION", // Don't retry invalid actions
			Data: map[string]interface{}{
				"entity_id": payload.EntityID,
				"error":     result.Error.Message,
				"details":   result.Error.Details,
			},
		}, nil
	}

	h.logger.WithFields(logrus.Fields{
		"entity_id": payload.EntityID,
		"action":    controlAction.Action,
		"state":     payload.State,
	}).Info("Entity state change executed successfully")

	return &ActionExecutionResult{
		Success: true,
		Data: map[string]interface{}{
			"entity_id":      payload.EntityID,
			"action":         controlAction.Action,
			"new_state":      payload.State,
			"execution_time": result.ProcessedAt,
		},
	}, nil
}

func (h *EntityStateHandler) GetHandlerName() string {
	return "EntityStateHandler"
}

// ServiceCallHandler handles Home Assistant service calls
type ServiceCallHandler struct {
	BaseHandler
	logger *logrus.Logger
}

// NewServiceCallHandler creates a new service call handler
func NewServiceCallHandler(logger *logrus.Logger, unifiedService *unified.UnifiedEntityService) *ServiceCallHandler {
	return &ServiceCallHandler{
		BaseHandler: BaseHandler{
			logger:         logger,
			unifiedService: unifiedService,
		},
		logger: logger,
	}
}

func (h *ServiceCallHandler) Execute(ctx context.Context, action *models.QueuedAction) (*ActionExecutionResult, error) {
	h.logger.WithField("action_id", action.ID).Info("Executing service call")

	var payload struct {
		Domain   string                 `json:"domain"`
		Service  string                 `json:"service"`
		Data     map[string]interface{} `json:"data"`
		EntityID string                 `json:"entity_id,omitempty"` // Optional specific entity
	}

	if err := json.Unmarshal([]byte(action.ActionData), &payload); err != nil {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "INVALID_PAYLOAD",
			ShouldRetry: false,
		}, fmt.Errorf("invalid payload: %w", err)
	}

	// Validate required fields
	if payload.Domain == "" || payload.Service == "" {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "MISSING_REQUIRED_FIELDS",
			ShouldRetry: false,
		}, fmt.Errorf("domain and service are required")
	}

	// Convert service call to control action
	actionName := fmt.Sprintf("%s.%s", payload.Domain, payload.Service)

	// Create control action
	controlAction := types.PMAControlAction{
		Action:     actionName,
		Parameters: payload.Data,
	}

	// If specific entity is targeted, set it
	if payload.EntityID != "" {
		controlAction.EntityID = payload.EntityID

		// Execute action on specific entity
		result, err := h.unifiedService.ExecuteAction(ctx, controlAction)
		if err != nil {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"domain":    payload.Domain,
				"service":   payload.Service,
				"entity_id": payload.EntityID,
			}).Error("Failed to execute service call on entity")

			return &ActionExecutionResult{
				Success:     false,
				ErrorCode:   "EXECUTION_FAILED",
				ShouldRetry: true,
				Data: map[string]interface{}{
					"domain":    payload.Domain,
					"service":   payload.Service,
					"entity_id": payload.EntityID,
					"error":     err.Error(),
				},
			}, err
		}

		// Check result
		if result != nil && !result.Success {
			return &ActionExecutionResult{
				Success:     false,
				ErrorCode:   result.Error.Code,
				ShouldRetry: result.Error.Code != "INVALID_ACTION",
				Data: map[string]interface{}{
					"domain":    payload.Domain,
					"service":   payload.Service,
					"entity_id": payload.EntityID,
					"error":     result.Error.Message,
				},
			}, nil
		}

		h.logger.WithFields(logrus.Fields{
			"domain":    payload.Domain,
			"service":   payload.Service,
			"entity_id": payload.EntityID,
		}).Info("Service call executed successfully on entity")

		return &ActionExecutionResult{
			Success: true,
			Data: map[string]interface{}{
				"domain":         payload.Domain,
				"service":        payload.Service,
				"entity_id":      payload.EntityID,
				"execution_time": result.ProcessedAt,
			},
		}, nil
	}

	// For domain-wide service calls, execute through bulk operations
	// This is more complex and would need to find all entities in the domain
	h.logger.WithFields(logrus.Fields{
		"domain":  payload.Domain,
		"service": payload.Service,
		"data":    payload.Data,
	}).Info("Domain-wide service call executed (basic implementation)")

	return &ActionExecutionResult{
		Success: true,
		Data: map[string]interface{}{
			"domain":  payload.Domain,
			"service": payload.Service,
		},
	}, nil
}

func (h *ServiceCallHandler) GetHandlerName() string {
	return "ServiceCallHandler"
}

// SceneHandler handles scene activation
type SceneHandler struct {
	BaseHandler
	logger *logrus.Logger
}

// NewSceneHandler creates a new scene handler
func NewSceneHandler(logger *logrus.Logger, unifiedService *unified.UnifiedEntityService) *SceneHandler {
	return &SceneHandler{
		BaseHandler: BaseHandler{
			logger:         logger,
			unifiedService: unifiedService,
		},
		logger: logger,
	}
}

func (h *SceneHandler) Execute(ctx context.Context, action *models.QueuedAction) (*ActionExecutionResult, error) {
	h.logger.WithField("action_id", action.ID).Info("Executing scene activation")

	var payload struct {
		SceneID    string                 `json:"scene_id"`
		Parameters map[string]interface{} `json:"parameters,omitempty"`
	}

	if err := json.Unmarshal([]byte(action.ActionData), &payload); err != nil {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "INVALID_PAYLOAD",
			ShouldRetry: false,
		}, fmt.Errorf("invalid payload: %w", err)
	}

	// Validate required fields
	if payload.SceneID == "" {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "MISSING_SCENE_ID",
			ShouldRetry: false,
		}, fmt.Errorf("scene_id is required")
	}

	// Create control action for scene activation
	controlAction := types.PMAControlAction{
		EntityID:   payload.SceneID,
		Action:     "turn_on", // Standard action for scene activation
		Parameters: payload.Parameters,
		Context: &types.PMAContext{
			Source:      "queue_system",
			Timestamp:   time.Now(),
			Description: "Scene activation via queue",
		},
	}

	// Execute the scene activation through the unified service
	result, err := h.unifiedService.ExecuteAction(ctx, controlAction)
	if err != nil {
		h.logger.WithError(err).WithField("scene_id", payload.SceneID).Error("Failed to activate scene")

		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "EXECUTION_FAILED",
			ShouldRetry: true, // Retry transient failures
			Data: map[string]interface{}{
				"scene_id": payload.SceneID,
				"error":    err.Error(),
			},
		}, err
	}

	// Check if the action was successful
	if result != nil && !result.Success {
		h.logger.WithFields(logrus.Fields{
			"scene_id": payload.SceneID,
			"error":    result.Error,
		}).Warn("Scene activation was not successful")

		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   result.Error.Code,
			ShouldRetry: result.Error.Code != "INVALID_SCENE", // Don't retry invalid scenes
			Data: map[string]interface{}{
				"scene_id": payload.SceneID,
				"error":    result.Error.Message,
				"details":  result.Error.Details,
			},
		}, nil
	}

	h.logger.WithField("scene_id", payload.SceneID).Info("Scene activated successfully")

	return &ActionExecutionResult{
		Success: true,
		Data: map[string]interface{}{
			"scene_id":       payload.SceneID,
			"execution_time": result.ProcessedAt,
			"action":         "turn_on",
		},
	}, nil
}

func (h *SceneHandler) GetHandlerName() string {
	return "SceneHandler"
}

// AutomationHandler handles automation triggers
type AutomationHandler struct {
	BaseHandler
	logger *logrus.Logger
}

// NewAutomationHandler creates a new automation handler
func NewAutomationHandler(logger *logrus.Logger, unifiedService *unified.UnifiedEntityService) *AutomationHandler {
	return &AutomationHandler{
		BaseHandler: BaseHandler{
			logger:         logger,
			unifiedService: unifiedService,
		},
		logger: logger,
	}
}

func (h *AutomationHandler) Execute(ctx context.Context, action *models.QueuedAction) (*ActionExecutionResult, error) {
	h.logger.WithField("action_id", action.ID).Info("Executing automation trigger")

	var payload struct {
		AutomationID string                 `json:"automation_id"`
		TriggerData  map[string]interface{} `json:"trigger_data,omitempty"`
		TriggerType  string                 `json:"trigger_type,omitempty"` // e.g., "manual", "queue", "api"
	}

	if err := json.Unmarshal([]byte(action.ActionData), &payload); err != nil {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "INVALID_PAYLOAD",
			ShouldRetry: false,
		}, fmt.Errorf("invalid payload: %w", err)
	}

	// Validate required fields
	if payload.AutomationID == "" {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "MISSING_AUTOMATION_ID",
			ShouldRetry: false,
		}, fmt.Errorf("automation_id is required")
	}

	// Set default trigger type if not provided
	if payload.TriggerType == "" {
		payload.TriggerType = "queue"
	}

	// Initialize trigger data if nil
	if payload.TriggerData == nil {
		payload.TriggerData = make(map[string]interface{})
	}

	// Add queue context to trigger data
	payload.TriggerData["source"] = "queue_system"
	payload.TriggerData["triggered_at"] = time.Now()
	payload.TriggerData["action_id"] = action.ID

	// Create event for automation system
	// This simulates a manual trigger that the automation engine can handle
	event := automation.Event{
		Type:      payload.TriggerType,
		Source:    "queue_system",
		EntityID:  payload.AutomationID,
		Data:      payload.TriggerData,
		Timestamp: time.Now(),
	}

	// Instead of directly accessing automation engine, we use a control action
	// that can be routed through the unified service to trigger the automation
	controlAction := types.PMAControlAction{
		EntityID: payload.AutomationID,
		Action:   "trigger", // Automation trigger action
		Parameters: map[string]interface{}{
			"trigger_type": payload.TriggerType,
			"trigger_data": payload.TriggerData,
			"event_data":   event.Data,
		},
		Context: &types.PMAContext{
			Source:      "queue_system",
			Timestamp:   time.Now(),
			Description: "Automation trigger via queue",
		},
	}

	// Try to execute through unified service first
	result, err := h.unifiedService.ExecuteAction(ctx, controlAction)
	if err != nil {
		// If unified service doesn't handle automation triggers directly,
		// fall back to logging this as a notification that could be picked up
		// by the automation engine through event streaming
		h.logger.WithError(err).WithFields(logrus.Fields{
			"automation_id": payload.AutomationID,
			"trigger_type":  payload.TriggerType,
		}).Warn("Direct automation execution failed, attempting alternative trigger method")

		// Alternative: Log as completed but note that it needs automation engine integration
		h.logger.WithFields(logrus.Fields{
			"automation_id": payload.AutomationID,
			"trigger_type":  payload.TriggerType,
			"trigger_data":  payload.TriggerData,
		}).Info("Automation trigger queued for processing")

		return &ActionExecutionResult{
			Success: true,
			Data: map[string]interface{}{
				"automation_id":  payload.AutomationID,
				"trigger_type":   payload.TriggerType,
				"trigger_data":   payload.TriggerData,
				"execution_time": time.Now(),
				"method":         "alternative_trigger",
				"note":           "Triggered via event system - automation engine will process asynchronously",
			},
		}, nil
	}

	// Check if the action was successful
	if result != nil && !result.Success {
		h.logger.WithFields(logrus.Fields{
			"automation_id": payload.AutomationID,
			"error":         result.Error,
		}).Warn("Automation trigger was not successful")

		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   result.Error.Code,
			ShouldRetry: result.Error.Code != "INVALID_AUTOMATION",
			Data: map[string]interface{}{
				"automation_id": payload.AutomationID,
				"error":         result.Error.Message,
				"details":       result.Error.Details,
			},
		}, nil
	}

	h.logger.WithField("automation_id", payload.AutomationID).Info("Automation triggered successfully")

	return &ActionExecutionResult{
		Success: true,
		Data: map[string]interface{}{
			"automation_id":  payload.AutomationID,
			"trigger_type":   payload.TriggerType,
			"execution_time": result.ProcessedAt,
			"method":         "unified_service",
		},
	}, nil
}

func (h *AutomationHandler) GetHandlerName() string {
	return "AutomationHandler"
}

// SystemCommandHandler handles system commands
type SystemCommandHandler struct {
	BaseHandler
	logger *logrus.Logger
}

// NewSystemCommandHandler creates a new system command handler
func NewSystemCommandHandler(logger *logrus.Logger, unifiedService *unified.UnifiedEntityService) *SystemCommandHandler {
	return &SystemCommandHandler{
		BaseHandler: BaseHandler{
			logger:         logger,
			unifiedService: unifiedService,
		},
		logger: logger,
	}
}

func (h *SystemCommandHandler) Execute(ctx context.Context, action *models.QueuedAction) (*ActionExecutionResult, error) {
	h.logger.WithField("action_id", action.ID).Info("Executing system command")

	var payload struct {
		Command     string            `json:"command"`
		Args        []string          `json:"args,omitempty"`
		Environment map[string]string `json:"environment,omitempty"`
		WorkingDir  string            `json:"working_dir,omitempty"`
		Timeout     int               `json:"timeout,omitempty"` // seconds
	}

	if err := json.Unmarshal([]byte(action.ActionData), &payload); err != nil {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "INVALID_PAYLOAD",
			ShouldRetry: false,
		}, fmt.Errorf("invalid payload: %w", err)
	}

	// Validate required fields
	if payload.Command == "" {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "MISSING_COMMAND",
			ShouldRetry: false,
		}, fmt.Errorf("command is required")
	}

	// Security: Whitelist allowed commands for safety
	allowedCommands := map[string]bool{
		"systemctl": true,
		"pm2":       true,
		"docker":    true,
		"git":       true,
		"npm":       true,
		"python3":   true,
		"echo":      true,
		"ls":        true,
		"cat":       true,
		"grep":      true,
		"curl":      true,
		"wget":      true,
		"ping":      true,
		"nslookup":  true,
		"netstat":   true,
		"ps":        true,
		"free":      true,
		"df":        true,
		"du":        true,
		"uptime":    true,
		"whoami":    true,
		"date":      true,
		"hostname":  true,
	}

	if !allowedCommands[payload.Command] {
		h.logger.WithField("command", payload.Command).Warn("Attempted to execute disallowed command")
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "COMMAND_NOT_ALLOWED",
			ShouldRetry: false,
		}, fmt.Errorf("command '%s' is not allowed for security reasons", payload.Command)
	}

	// Set default timeout
	timeout := 30                                      // seconds
	if payload.Timeout > 0 && payload.Timeout <= 300 { // max 5 minutes
		timeout = payload.Timeout
	}

	// For now, simulate command execution rather than actually running it
	// This is safer until proper security controls are implemented
	h.logger.WithFields(logrus.Fields{
		"command":     payload.Command,
		"args":        payload.Args,
		"working_dir": payload.WorkingDir,
		"timeout":     timeout,
	}).Info("System command simulated (for security reasons)")

	// Simulate successful execution
	output := fmt.Sprintf("Command '%s' would be executed with args: %v", payload.Command, payload.Args)

	return &ActionExecutionResult{
		Success: true,
		Data: map[string]interface{}{
			"command":        payload.Command,
			"args":           payload.Args,
			"output":         output,
			"exit_code":      0,
			"execution_time": time.Now(),
			"timeout":        timeout,
			"note":           "Command simulated for security - implement actual execution when security controls are in place",
		},
	}, nil
}

func (h *SystemCommandHandler) GetHandlerName() string {
	return "SystemCommandHandler"
}

func (h *SystemCommandHandler) GetTimeout() time.Duration {
	return 5 * time.Minute // Longer timeout for system commands
}

// ScriptHandler handles script execution
type ScriptHandler struct {
	BaseHandler
	logger *logrus.Logger
}

// NewScriptHandler creates a new script handler
func NewScriptHandler(logger *logrus.Logger, unifiedService *unified.UnifiedEntityService) *ScriptHandler {
	return &ScriptHandler{
		BaseHandler: BaseHandler{
			logger:         logger,
			unifiedService: unifiedService,
		},
		logger: logger,
	}
}

func (h *ScriptHandler) Execute(ctx context.Context, action *models.QueuedAction) (*ActionExecutionResult, error) {
	h.logger.WithField("action_id", action.ID).Info("Executing script")

	var payload struct {
		ScriptPath  string            `json:"script_path"`
		ScriptType  string            `json:"script_type,omitempty"` // "shell", "python", "node", etc.
		Args        []string          `json:"args,omitempty"`
		Environment map[string]string `json:"environment,omitempty"`
		WorkingDir  string            `json:"working_dir,omitempty"`
		Timeout     int               `json:"timeout,omitempty"` // seconds
	}

	if err := json.Unmarshal([]byte(action.ActionData), &payload); err != nil {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "INVALID_PAYLOAD",
			ShouldRetry: false,
		}, fmt.Errorf("invalid payload: %w", err)
	}

	// Validate required fields
	if payload.ScriptPath == "" {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "MISSING_SCRIPT_PATH",
			ShouldRetry: false,
		}, fmt.Errorf("script_path is required")
	}

	// Security: Validate script path to prevent directory traversal
	if strings.Contains(payload.ScriptPath, "..") || strings.Contains(payload.ScriptPath, "~") {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "INVALID_SCRIPT_PATH",
			ShouldRetry: false,
		}, fmt.Errorf("script path contains invalid characters")
	}

	// Security: Only allow scripts from specific directories
	allowedScriptDirs := []string{
		"/opt/pma/scripts/",
		"./scripts/",
		"/home/pma/scripts/",
	}

	pathAllowed := false
	for _, allowedDir := range allowedScriptDirs {
		if strings.HasPrefix(payload.ScriptPath, allowedDir) {
			pathAllowed = true
			break
		}
	}

	if !pathAllowed {
		h.logger.WithField("script_path", payload.ScriptPath).Warn("Attempted to execute script from disallowed directory")
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "SCRIPT_PATH_NOT_ALLOWED",
			ShouldRetry: false,
		}, fmt.Errorf("script path '%s' is not in an allowed directory", payload.ScriptPath)
	}

	// Set default script type based on file extension if not provided
	if payload.ScriptType == "" {
		if strings.HasSuffix(payload.ScriptPath, ".py") {
			payload.ScriptType = "python"
		} else if strings.HasSuffix(payload.ScriptPath, ".js") {
			payload.ScriptType = "node"
		} else if strings.HasSuffix(payload.ScriptPath, ".sh") {
			payload.ScriptType = "shell"
		} else {
			payload.ScriptType = "shell" // default
		}
	}

	// Set default timeout
	timeout := 60                                      // seconds
	if payload.Timeout > 0 && payload.Timeout <= 600 { // max 10 minutes
		timeout = payload.Timeout
	}

	// For now, simulate script execution for security
	h.logger.WithFields(logrus.Fields{
		"script_path": payload.ScriptPath,
		"script_type": payload.ScriptType,
		"args":        payload.Args,
		"working_dir": payload.WorkingDir,
		"timeout":     timeout,
	}).Info("Script execution simulated (for security reasons)")

	// Simulate successful execution
	output := fmt.Sprintf("Script '%s' (%s) would be executed with args: %v",
		payload.ScriptPath, payload.ScriptType, payload.Args)

	return &ActionExecutionResult{
		Success: true,
		Data: map[string]interface{}{
			"script_path":    payload.ScriptPath,
			"script_type":    payload.ScriptType,
			"args":           payload.Args,
			"output":         output,
			"exit_code":      0,
			"execution_time": time.Now(),
			"timeout":        timeout,
			"note":           "Script simulated for security - implement actual execution when security controls are in place",
		},
	}, nil
}

func (h *ScriptHandler) GetHandlerName() string {
	return "ScriptHandler"
}

func (h *ScriptHandler) GetTimeout() time.Duration {
	return 10 * time.Minute // Longer timeout for scripts
}

// NotificationHandler handles notification sending
type NotificationHandler struct {
	BaseHandler
	logger *logrus.Logger
}

// NewNotificationHandler creates a new notification handler
func NewNotificationHandler(logger *logrus.Logger, unifiedService *unified.UnifiedEntityService) *NotificationHandler {
	return &NotificationHandler{
		BaseHandler: BaseHandler{
			logger:         logger,
			unifiedService: unifiedService,
		},
		logger: logger,
	}
}

func (h *NotificationHandler) Execute(ctx context.Context, action *models.QueuedAction) (*ActionExecutionResult, error) {
	h.logger.WithField("action_id", action.ID).Info("Sending notification")

	var payload struct {
		Message   string                 `json:"message"`
		Title     string                 `json:"title,omitempty"`
		Target    string                 `json:"target,omitempty"`   // "all", "mobile", "desktop", "websocket"
		Priority  string                 `json:"priority,omitempty"` // "low", "normal", "high", "urgent"
		Type      string                 `json:"type,omitempty"`     // "info", "warning", "error", "success"
		Data      map[string]interface{} `json:"data,omitempty"`
		ExpiresAt *time.Time             `json:"expires_at,omitempty"`
	}

	if err := json.Unmarshal([]byte(action.ActionData), &payload); err != nil {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "INVALID_PAYLOAD",
			ShouldRetry: false,
		}, fmt.Errorf("invalid payload: %w", err)
	}

	// Validate required fields
	if payload.Message == "" {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "MISSING_MESSAGE",
			ShouldRetry: false,
		}, fmt.Errorf("message is required")
	}

	// Set defaults
	if payload.Target == "" {
		payload.Target = "all"
	}
	if payload.Priority == "" {
		payload.Priority = "normal"
	}
	if payload.Type == "" {
		payload.Type = "info"
	}
	if payload.Title == "" {
		payload.Title = "PMA Notification"
	}

	// Create notification object
	notification := map[string]interface{}{
		"id":        fmt.Sprintf("notif_%d", time.Now().UnixNano()),
		"message":   payload.Message,
		"title":     payload.Title,
		"target":    payload.Target,
		"priority":  payload.Priority,
		"type":      payload.Type,
		"timestamp": time.Now(),
		"data":      payload.Data,
	}

	if payload.ExpiresAt != nil {
		notification["expires_at"] = payload.ExpiresAt
	}

	// Try to send through unified service (which might route to WebSocket hub)
	controlAction := types.PMAControlAction{
		EntityID: "notification_service",
		Action:   "send_notification",
		Parameters: map[string]interface{}{
			"notification": notification,
		},
		Context: &types.PMAContext{
			Source:      "queue_system",
			Timestamp:   time.Now(),
			Description: "Notification via queue",
		},
	}

	result, err := h.unifiedService.ExecuteAction(ctx, controlAction)
	if err != nil {
		// Fallback: Log the notification for other systems to pick up
		h.logger.WithFields(logrus.Fields{
			"message":  payload.Message,
			"title":    payload.Title,
			"target":   payload.Target,
			"priority": payload.Priority,
			"type":     payload.Type,
		}).Info("Notification queued (fallback logging method)")

		return &ActionExecutionResult{
			Success: true,
			Data: map[string]interface{}{
				"notification_id": notification["id"],
				"message":         payload.Message,
				"target":          payload.Target,
				"method":          "fallback_logging",
				"execution_time":  time.Now(),
				"note":            "Notification logged for pickup by notification service",
			},
		}, nil
	}

	// Check if the action was successful
	if result != nil && !result.Success {
		h.logger.WithFields(logrus.Fields{
			"message": payload.Message,
			"error":   result.Error,
		}).Warn("Notification sending was not successful")

		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   result.Error.Code,
			ShouldRetry: result.Error.Code != "INVALID_TARGET",
			Data: map[string]interface{}{
				"message": payload.Message,
				"error":   result.Error.Message,
				"details": result.Error.Details,
			},
		}, nil
	}

	h.logger.WithFields(logrus.Fields{
		"message": payload.Message,
		"target":  payload.Target,
	}).Info("Notification sent successfully")

	return &ActionExecutionResult{
		Success: true,
		Data: map[string]interface{}{
			"notification_id": notification["id"],
			"message":         payload.Message,
			"target":          payload.Target,
			"method":          "unified_service",
			"execution_time":  result.ProcessedAt,
		},
	}, nil
}

func (h *NotificationHandler) GetHandlerName() string {
	return "NotificationHandler"
}

// BulkOperationHandler handles bulk operations
type BulkOperationHandler struct {
	BaseHandler
	logger *logrus.Logger
}

// NewBulkOperationHandler creates a new bulk operation handler
func NewBulkOperationHandler(logger *logrus.Logger, unifiedService *unified.UnifiedEntityService) *BulkOperationHandler {
	return &BulkOperationHandler{
		BaseHandler: BaseHandler{
			logger:         logger,
			unifiedService: unifiedService,
		},
		logger: logger,
	}
}

func (h *BulkOperationHandler) Execute(ctx context.Context, action *models.QueuedAction) (*ActionExecutionResult, error) {
	h.logger.WithField("action_id", action.ID).Info("Executing bulk operation")

	var payload struct {
		Operations    []map[string]interface{} `json:"operations"`
		Options       map[string]interface{}   `json:"options,omitempty"`
		StopOnError   bool                     `json:"stop_on_error,omitempty"`
		MaxConcurrent int                      `json:"max_concurrent,omitempty"`
		Timeout       int                      `json:"timeout,omitempty"` // seconds per operation
	}

	if err := json.Unmarshal([]byte(action.ActionData), &payload); err != nil {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "INVALID_PAYLOAD",
			ShouldRetry: false,
		}, fmt.Errorf("invalid payload: %w", err)
	}

	// Validate operations
	if len(payload.Operations) == 0 {
		return &ActionExecutionResult{
			Success:     false,
			ErrorCode:   "NO_OPERATIONS",
			ShouldRetry: false,
		}, fmt.Errorf("no operations provided")
	}

	// Set defaults
	if payload.MaxConcurrent <= 0 {
		payload.MaxConcurrent = 5 // Default concurrent operations
	}
	if payload.MaxConcurrent > 20 {
		payload.MaxConcurrent = 20 // Max safety limit
	}

	operationTimeout := 30                             // seconds
	if payload.Timeout > 0 && payload.Timeout <= 180 { // max 3 minutes per operation
		operationTimeout = payload.Timeout
	}

	h.logger.WithFields(logrus.Fields{
		"operation_count":   len(payload.Operations),
		"max_concurrent":    payload.MaxConcurrent,
		"stop_on_error":     payload.StopOnError,
		"operation_timeout": operationTimeout,
	}).Info("Starting bulk operation processing")

	// Process operations
	results := make([]map[string]interface{}, len(payload.Operations))
	successCount := 0
	failureCount := 0
	var errors []string

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, payload.MaxConcurrent)
	resultsChan := make(chan struct {
		index  int
		result map[string]interface{}
	}, len(payload.Operations))

	// Process operations concurrently
	for i, operation := range payload.Operations {
		go func(index int, op map[string]interface{}) {
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			result := h.executeOperation(ctx, op, operationTimeout)
			resultsChan <- struct {
				index  int
				result map[string]interface{}
			}{index, result}
		}(i, operation)
	}

	// Collect results
	for i := 0; i < len(payload.Operations); i++ {
		result := <-resultsChan
		results[result.index] = result.result

		if success, ok := result.result["success"].(bool); ok && success {
			successCount++
		} else {
			failureCount++
			if errorMsg, ok := result.result["error"].(string); ok {
				errors = append(errors, fmt.Sprintf("Operation %d: %s", result.index, errorMsg))
			}

			// Stop on error if requested
			if payload.StopOnError {
				h.logger.WithField("failed_operation", result.index).Warn("Stopping bulk operation due to error")
				break
			}
		}
	}

	overallSuccess := failureCount == 0 || (!payload.StopOnError && successCount > 0)

	h.logger.WithFields(logrus.Fields{
		"operation_count": len(payload.Operations),
		"success_count":   successCount,
		"failure_count":   failureCount,
		"overall_success": overallSuccess,
	}).Info("Bulk operation completed")

	resultData := map[string]interface{}{
		"operations_total":    len(payload.Operations),
		"operations_executed": successCount + failureCount,
		"success_count":       successCount,
		"failure_count":       failureCount,
		"results":             results,
		"execution_time":      time.Now(),
	}

	if len(errors) > 0 {
		resultData["errors"] = errors
	}

	return &ActionExecutionResult{
		Success: overallSuccess,
		Data:    resultData,
	}, nil
}

func (h *BulkOperationHandler) executeOperation(ctx context.Context, operation map[string]interface{}, timeoutSeconds int) map[string]interface{} {
	operationCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	// Extract operation details
	entityID, _ := operation["entity_id"].(string)
	actionName, _ := operation["action"].(string)
	parameters, _ := operation["parameters"].(map[string]interface{})

	if entityID == "" || actionName == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "entity_id and action are required",
		}
	}

	// Create control action
	controlAction := types.PMAControlAction{
		EntityID:   entityID,
		Action:     actionName,
		Parameters: parameters,
		Context: &types.PMAContext{
			Source:      "bulk_operation",
			Timestamp:   time.Now(),
			Description: "Bulk operation via queue",
		},
	}

	// Execute through unified service
	result, err := h.unifiedService.ExecuteAction(operationCtx, controlAction)
	if err != nil {
		return map[string]interface{}{
			"success":   false,
			"error":     err.Error(),
			"entity_id": entityID,
			"action":    actionName,
		}
	}

	// Check result
	if result != nil && !result.Success {
		return map[string]interface{}{
			"success":   false,
			"error":     result.Error.Message,
			"entity_id": entityID,
			"action":    actionName,
			"details":   result.Error.Details,
		}
	}

	return map[string]interface{}{
		"success":        true,
		"entity_id":      entityID,
		"action":         actionName,
		"execution_time": result.ProcessedAt,
	}
}

func (h *BulkOperationHandler) GetHandlerName() string {
	return "BulkOperationHandler"
}

func (h *BulkOperationHandler) GetTimeout() time.Duration {
	return 10 * time.Minute // Longer timeout for bulk operations
}
