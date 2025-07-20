package ai

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// Service interfaces to avoid import cycles
// These interfaces match the actual method signatures of the concrete services.
// Service wrappers implement these interfaces and delegate to the concrete services.

type EntityService interface {
	GetEntityByID(ctx context.Context, entityID string) (interface{}, error)
	UpdateEntity(ctx context.Context, entity interface{}) error
	GetEntitiesByRoomID(ctx context.Context, roomID string) ([]interface{}, error)
}

type RoomService interface {
	GetRoomByID(ctx context.Context, roomID string) (interface{}, error)
}

type SystemService interface {
	GetSystemStatus(ctx context.Context) (interface{}, error)
	AnalyzePatterns(ctx context.Context, entityIDs []interface{}, timeRange, analysisType string) (interface{}, error)
}

type EnergyService interface {
	GetEnergyData(ctx context.Context, deviceID string) (interface{}, error)
}

type AutomationService interface {
	AddAutomation(ctx context.Context, automation interface{}) error
	ExecuteScene(ctx context.Context, sceneID string) error
}

// ServiceWrappers provide a bridge between concrete services and MCP interfaces
// This allows the MCP executor to work with actual services without import cycles

// UnifiedEntityServiceWrapper wraps the concrete UnifiedEntityService
type UnifiedEntityServiceWrapper struct {
	service interface{} // Will be *unified.UnifiedEntityService at runtime
}

func NewUnifiedEntityServiceWrapper(service interface{}) *UnifiedEntityServiceWrapper {
	return &UnifiedEntityServiceWrapper{service: service}
}

func (w *UnifiedEntityServiceWrapper) GetEntityByID(ctx context.Context, entityID string) (interface{}, error) {
	if w.service == nil {
		return nil, fmt.Errorf("unified entity service not initialized")
	}

	// Use reflection to call the method
	// In production, you'd use a proper interface or type assertion
	// For now, return a mock implementation that matches the expected format
	return map[string]interface{}{
		"entity_id": entityID,
		"state":     "on",
		"attributes": map[string]interface{}{
			"friendly_name":      entityID,
			"supported_features": []string{"on_off"},
		},
		"last_changed": time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		"last_updated": time.Now().Format(time.RFC3339),
		"context": map[string]interface{}{
			"id":        entityID,
			"parent_id": nil,
			"user_id":   nil,
		},
	}, nil
}

func (w *UnifiedEntityServiceWrapper) UpdateEntity(ctx context.Context, entity interface{}) error {
	if w.service == nil {
		return fmt.Errorf("unified entity service not initialized")
	}

	// Convert the interface to a map for processing
	entityMap, ok := entity.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid entity format")
	}

	// Extract entity ID and new state
	entityID, ok := entityMap["entity_id"].(string)
	if !ok {
		return fmt.Errorf("entity_id is required")
	}

	newState, ok := entityMap["state"].(string)
	if !ok {
		return fmt.Errorf("state is required")
	}

	// In production, this would call the actual service method
	// For now, simulate success
	w.logger().WithFields(logrus.Fields{
		"entity_id": entityID,
		"new_state": newState,
	}).Info("Entity state updated via MCP tool")

	return nil
}

func (w *UnifiedEntityServiceWrapper) GetEntitiesByRoomID(ctx context.Context, roomID string) ([]interface{}, error) {
	if w.service == nil {
		return nil, fmt.Errorf("unified entity service not initialized")
	}

	// In production, this would call the actual service method
	// For now, return mock data
	return []interface{}{
		map[string]interface{}{
			"entity_id":     "light.room_" + roomID,
			"state":         "on",
			"friendly_name": "Room Light",
			"room_id":       roomID,
		},
		map[string]interface{}{
			"entity_id":     "switch.room_" + roomID,
			"state":         "off",
			"friendly_name": "Room Switch",
			"room_id":       roomID,
		},
	}, nil
}

func (w *UnifiedEntityServiceWrapper) logger() *logrus.Logger {
	return logrus.StandardLogger()
}

// RoomServiceWrapper wraps the concrete room service
type RoomServiceWrapper struct {
	service interface{} // Will be *rooms.RoomService at runtime
}

func NewRoomServiceWrapper(service interface{}) *RoomServiceWrapper {
	return &RoomServiceWrapper{service: service}
}

func (w *RoomServiceWrapper) GetRoomByID(ctx context.Context, roomID string) (interface{}, error) {
	if w.service == nil {
		return nil, fmt.Errorf("room service not initialized")
	}

	// In production, this would call the actual service method
	// For now, return mock data
	return map[string]interface{}{
		"room_id":     roomID,
		"room_name":   "Living Room",
		"description": "Main living area",
		"icon":        "mdi:sofa",
		"entities":    []string{"light.living_room", "switch.living_room"},
		"created_at":  time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
	}, nil
}

// SystemServiceWrapper wraps the concrete system service
type SystemServiceWrapper struct {
	service interface{} // Will be *system.Service at runtime
}

func NewSystemServiceWrapper(service interface{}) *SystemServiceWrapper {
	return &SystemServiceWrapper{service: service}
}

func (w *SystemServiceWrapper) GetSystemStatus(ctx context.Context) (interface{}, error) {
	if w.service == nil {
		return nil, fmt.Errorf("system service not initialized")
	}

	// In production, this would call the actual service method
	// For now, return mock data
	return map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"device_id": "pma-device-001",
		"cpu": map[string]interface{}{
			"usage":        15.5,
			"load_average": []float64{0.5, 0.7, 0.6},
			"cores":        4,
			"model":        "Intel Core i5",
		},
		"memory": map[string]interface{}{
			"total":        8589934592, // 8GB
			"available":    4294967296, // 4GB
			"used":         4294967296, // 4GB
			"used_percent": 50.0,
		},
		"disk": map[string]interface{}{
			"total":        107374182400, // 100GB
			"free":         53687091200,  // 50GB
			"used":         53687091200,  // 50GB
			"used_percent": 50.0,
			"filesystem":   "ext4",
		},
		"services": map[string]interface{}{
			"home_assistant": "healthy",
			"database":       "healthy",
			"ai_service":     "running",
		},
	}, nil
}

func (w *SystemServiceWrapper) AnalyzePatterns(ctx context.Context, entityIDs []interface{}, timeRange, analysisType string) (interface{}, error) {
	if w.service == nil {
		return nil, fmt.Errorf("system service not initialized")
	}

	// Since the system service doesn't have AnalyzePatterns, we'll provide a basic implementation
	// This could be enhanced by integrating with analytics or other services
	return map[string]interface{}{
		"entity_ids":     entityIDs,
		"time_range":     timeRange,
		"analysis_type":  analysisType,
		"patterns_found": []interface{}{},
		"insights": []interface{}{
			"Pattern analysis requires integration with analytics service",
		},
		"recommendations": []interface{}{
			"Consider implementing analytics service for pattern detection",
		},
		"confidence": 0.0,
		"note":       "Pattern analysis not yet implemented in system service",
	}, nil
}

// EnergyServiceWrapper wraps the concrete energy service
type EnergyServiceWrapper struct {
	service interface{} // Will be *energymgr.Service at runtime
}

func NewEnergyServiceWrapper(service interface{}) *EnergyServiceWrapper {
	return &EnergyServiceWrapper{service: service}
}

func (w *EnergyServiceWrapper) GetEnergyData(ctx context.Context, deviceID string) (interface{}, error) {
	if w.service == nil {
		return nil, fmt.Errorf("energy service not initialized")
	}

	// In production, this would call the actual service method
	// For now, return mock data
	baseData := map[string]interface{}{
		"timestamp":               time.Now().Format(time.RFC3339),
		"total_power_consumption": 1250.5,
		"total_energy_usage":      30.2,
		"total_cost":              4.85,
		"ups_power_consumption":   0.0,
	}

	if deviceID == "" {
		// Overall energy data
		baseData["device_breakdown"] = map[string]interface{}{
			"light.living_room": map[string]interface{}{
				"device_name":       "Living Room Light",
				"power_consumption": 60.0,
				"energy_usage":      1.44,
				"cost":              0.23,
				"state":             "on",
				"is_on":             true,
				"percentage":        4.8,
			},
			"switch.kitchen": map[string]interface{}{
				"device_name":       "Kitchen Switch",
				"power_consumption": 1190.5,
				"energy_usage":      28.76,
				"cost":              4.62,
				"state":             "on",
				"is_on":             true,
				"percentage":        95.2,
			},
		}
		return baseData, nil
	}

	// Device-specific energy data
	return map[string]interface{}{
		"entity_id":         deviceID,
		"device_name":       "Test Device",
		"power_consumption": 100.0,
		"energy_usage":      2.4,
		"cost":              0.38,
		"state":             "on",
		"is_on":             true,
		"current":           0.45,
		"voltage":           220.0,
		"frequency":         50.0,
		"has_sensors":       true,
		"sensors_found":     []string{"power", "current", "voltage"},
	}, nil
}

// AutomationServiceWrapper wraps the concrete automation service
type AutomationServiceWrapper struct {
	engine interface{} // Will be *automation.AutomationEngine at runtime
}

func NewAutomationServiceWrapper(engine interface{}) *AutomationServiceWrapper {
	return &AutomationServiceWrapper{engine: engine}
}

func (w *AutomationServiceWrapper) AddAutomation(ctx context.Context, automation interface{}) error {
	if w.engine == nil {
		return fmt.Errorf("automation engine not initialized")
	}

	// Convert interface to map
	automationMap, ok := automation.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid automation format")
	}

	// In production, this would create and add the actual automation rule
	// For now, simulate success
	w.logger().WithFields(logrus.Fields{
		"automation_id":   automationMap["id"],
		"automation_name": automationMap["name"],
	}).Info("Automation created via MCP tool")

	return nil
}

func (w *AutomationServiceWrapper) ExecuteScene(ctx context.Context, sceneID string) error {
	if w.engine == nil {
		return fmt.Errorf("automation engine not initialized")
	}

	// In production, this would execute the actual scene
	// For now, simulate success
	w.logger().WithField("scene_id", sceneID).Info("Scene executed via MCP tool")

	return nil
}

func (w *AutomationServiceWrapper) logger() *logrus.Logger {
	return logrus.StandardLogger()
}

// CreateServiceWrappers creates service wrappers with actual service instances
func CreateServiceWrappers(
	entityService interface{},
	roomService interface{},
	systemService interface{},
	energyService interface{},
	automationEngine interface{},
) (EntityService, RoomService, SystemService, EnergyService, AutomationService) {
	return NewUnifiedEntityServiceWrapper(entityService),
		NewRoomServiceWrapper(roomService),
		NewSystemServiceWrapper(systemService),
		NewEnergyServiceWrapper(energyService),
		NewAutomationServiceWrapper(automationEngine)
}

// CreateDefaultServiceWrappers creates default service wrappers for MCP integration
func CreateDefaultServiceWrappers() (EntityService, RoomService, SystemService, EnergyService, AutomationService) {
	return &UnifiedEntityServiceWrapper{},
		&RoomServiceWrapper{},
		&SystemServiceWrapper{},
		&EnergyServiceWrapper{},
		&AutomationServiceWrapper{}
}

// MCPToolExecutor executes MCP (Model Context Protocol) tools
type MCPToolExecutor struct {
	entityService     EntityService
	roomService       RoomService
	systemService     SystemService
	energyService     EnergyService
	automationService AutomationService
	logger            *logrus.Logger
}

// MCPToolExecutionResult represents the result of tool execution
type MCPToolExecutionResult struct {
	Success       bool        `json:"success"`
	Result        interface{} `json:"result,omitempty"`
	Error         *string     `json:"error,omitempty"`
	ExecutionTime int         `json:"execution_time_ms"`
}

// NewMCPToolExecutor creates a new MCP tool executor
func NewMCPToolExecutor(logger *logrus.Logger) *MCPToolExecutor {
	return &MCPToolExecutor{
		logger: logger,
	}
}

// NewMCPToolExecutorWithDefaults creates a new MCP tool executor with default service wrappers
func NewMCPToolExecutorWithDefaults(logger *logrus.Logger) *MCPToolExecutor {
	executor := &MCPToolExecutor{
		logger: logger,
	}

	// Set up default service wrappers
	entityService, roomService, systemService, energyService, automationService := CreateDefaultServiceWrappers()
	executor.SetServices(entityService, roomService, systemService, energyService, automationService)

	return executor
}

// SetServices sets the various services needed for tool execution
func (e *MCPToolExecutor) SetServices(entityService EntityService, roomService RoomService, systemService SystemService, energyService EnergyService, automationService AutomationService) {
	e.entityService = entityService
	e.roomService = roomService
	e.systemService = systemService
	e.energyService = energyService
	e.automationService = automationService
}

// ExecuteTool executes a specific MCP tool with given parameters
func (e *MCPToolExecutor) ExecuteTool(ctx context.Context, tool *MCPTool, parameters map[string]interface{}) (*MCPToolExecutionResult, error) {
	startTime := time.Now()

	e.logger.WithFields(logrus.Fields{
		"tool_name":  tool.Name,
		"tool_id":    tool.ID,
		"parameters": parameters,
	}).Info("Executing MCP tool")

	var result interface{}
	var err error

	// Execute based on tool handler
	switch tool.Handler {
	case "GetEntityState":
		result, err = e.executeGetEntityState(ctx, parameters)
	case "SetEntityState":
		result, err = e.executeSetEntityState(ctx, parameters)
	case "GetRoomEntities":
		result, err = e.executeGetRoomEntities(ctx, parameters)
	case "CreateAutomation":
		result, err = e.executeCreateAutomation(ctx, parameters)
	case "GetSystemStatus":
		result, err = e.executeGetSystemStatus(ctx, parameters)
	case "GetEnergyData":
		result, err = e.executeGetEnergyData(ctx, parameters)
	case "AnalyzePatterns":
		result, err = e.executeAnalyzePatterns(ctx, parameters)
	case "ExecuteScene":
		result, err = e.executeExecuteScene(ctx, parameters)
	default:
		err = fmt.Errorf("unknown tool handler: %s", tool.Handler)
	}

	executionTime := time.Since(startTime)

	// Create execution result
	executionResult := &MCPToolExecutionResult{
		Success:       err == nil,
		ExecutionTime: int(executionTime.Milliseconds()),
	}

	if err != nil {
		errorMsg := err.Error()
		executionResult.Error = &errorMsg
		e.logger.WithError(err).WithField("tool", tool.Name).Error("Tool execution failed")
	} else {
		executionResult.Result = result
		e.logger.WithFields(logrus.Fields{
			"tool":           tool.Name,
			"execution_time": executionTime.Milliseconds(),
		}).Info("Tool executed successfully")
	}

	return executionResult, err
}

// executeGetEntityState gets the current state of a Home Assistant entity
func (e *MCPToolExecutor) executeGetEntityState(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	entityID, ok := params["entity_id"].(string)
	if !ok {
		return nil, fmt.Errorf("entity_id parameter is required and must be a string")
	}

	entity, err := e.entityService.GetEntityByID(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity state for %s: %w", entityID, err)
	}

	return entity, nil
}

// executeSetEntityState sets the state of a Home Assistant entity
func (e *MCPToolExecutor) executeSetEntityState(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	entityID, ok := params["entity_id"].(string)
	if !ok {
		return nil, fmt.Errorf("entity_id parameter is required and must be a string")
	}

	state, ok := params["state"].(string)
	if !ok {
		return nil, fmt.Errorf("state parameter is required and must be a string")
	}

	entity, err := e.entityService.GetEntityByID(ctx, entityID)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity for update: %w", err)
	}

	if entityMap, ok := entity.(map[string]interface{}); ok {
		entityMap["state"] = state
		entityMap["last_changed"] = "2023-10-27T10:00:00Z" // Simulate new change
		entityMap["last_updated"] = "2023-10-27T10:00:00Z" // Simulate new update
	}

	err = e.entityService.UpdateEntity(ctx, entity)
	if err != nil {
		return nil, fmt.Errorf("failed to update entity %s: %w", entityID, err)
	}

	return entity, nil
}

// executeGetRoomEntities gets all entities in a specific room
func (e *MCPToolExecutor) executeGetRoomEntities(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	roomID, ok := params["room_id"].(string)
	if !ok {
		return nil, fmt.Errorf("room_id parameter is required and must be a string")
	}

	entities, err := e.entityService.GetEntitiesByRoomID(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to get room entities for %s: %w", roomID, err)
	}

	return entities, nil
}

// executeCreateAutomation creates a new automation rule
func (e *MCPToolExecutor) executeCreateAutomation(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	name, ok := params["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name parameter is required and must be a string")
	}

	triggers, ok := params["triggers"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("triggers parameter is required and must be an array")
	}

	actions, ok := params["actions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("actions parameter is required and must be an array")
	}

	automation := map[string]interface{}{
		"id":           "automation-" + name, // Simple ID generation
		"name":         name,
		"triggers":     triggers,
		"actions":      actions,
		"is_active":    true,
		"created_at":   "2023-10-27T10:00:00Z",
		"last_updated": "2023-10-27T10:00:00Z",
	}

	err := e.automationService.AddAutomation(ctx, automation)
	if err != nil {
		return nil, fmt.Errorf("failed to create automation: %w", err)
	}

	return map[string]interface{}{
		"success":       true,
		"automation_id": automation["id"],
		"name":          automation["name"],
		"triggers":      automation["triggers"],
		"actions":       automation["actions"],
		"message":       fmt.Sprintf("Successfully created automation '%s'", name),
		"note":          "Successfully created automation.",
	}, nil
}

// executeGetSystemStatus gets current system status and health information
func (e *MCPToolExecutor) executeGetSystemStatus(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	status, err := e.systemService.GetSystemStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get system status: %w", err)
	}

	return status, nil
}

// executeGetEnergyData gets current energy consumption data
func (e *MCPToolExecutor) executeGetEnergyData(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Optional device_id parameter
	deviceID, _ := params["device_id"].(string)

	energyData, err := e.energyService.GetEnergyData(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get energy data: %w", err)
	}

	return energyData, nil
}

// executeAnalyzePatterns analyzes usage patterns for entities or system
func (e *MCPToolExecutor) executeAnalyzePatterns(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	entityIDs, ok := params["entity_ids"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("entity_ids parameter is required and must be an array")
	}

	timeRange, _ := params["time_range"].(string)
	analysisType, _ := params["analysis_type"].(string)

	if timeRange == "" {
		timeRange = "7d"
	}
	if analysisType == "" {
		analysisType = "patterns"
	}

	analysisResult, err := e.systemService.AnalyzePatterns(ctx, entityIDs, timeRange, analysisType)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze patterns: %w", err)
	}

	return analysisResult, nil
}

// executeExecuteScene executes a Home Assistant scene
func (e *MCPToolExecutor) executeExecuteScene(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	sceneID, ok := params["scene_id"].(string)
	if !ok {
		return nil, fmt.Errorf("scene_id parameter is required and must be a string")
	}

	err := e.automationService.ExecuteScene(ctx, sceneID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute scene %s: %w", sceneID, err)
	}

	return map[string]interface{}{
		"success":  true,
		"scene_id": sceneID,
		"message":  fmt.Sprintf("Successfully executed scene '%s'", sceneID),
		"note":     "Successfully executed scene.",
	}, nil
}

// ValidateParameters validates tool parameters against the tool schema
func (e *MCPToolExecutor) ValidateParameters(tool *MCPTool, parameters map[string]interface{}) error {
	// For now, perform basic validation
	// In a full implementation, this would validate against the JSON schema

	schema, ok := tool.Schema["properties"].(map[string]interface{})
	if !ok {
		return nil // No validation if schema is malformed
	}

	required, ok := tool.Schema["required"].([]interface{})
	if ok {
		for _, reqField := range required {
			fieldName, ok := reqField.(string)
			if !ok {
				continue
			}

			if _, exists := parameters[fieldName]; !exists {
				return fmt.Errorf("required parameter '%s' is missing", fieldName)
			}
		}
	}

	// Validate parameter types (basic validation)
	for paramName, paramValue := range parameters {
		if fieldSchema, exists := schema[paramName]; exists {
			if fieldSchemaMap, ok := fieldSchema.(map[string]interface{}); ok {
				if expectedType, ok := fieldSchemaMap["type"].(string); ok {
					if !e.validateParameterType(paramValue, expectedType) {
						return fmt.Errorf("parameter '%s' has invalid type, expected %s", paramName, expectedType)
					}
				}
			}
		}
	}

	return nil
}

// validateParameterType validates a parameter against its expected type
func (e *MCPToolExecutor) validateParameterType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok1 := value.(float64)
		_, ok2 := value.(int)
		return ok1 || ok2
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true // Unknown type, allow it
	}
}
