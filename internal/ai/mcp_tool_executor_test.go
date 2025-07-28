package ai

import (
	"context"
	"testing"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/interfaces"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing

// MockEntityService implements interfaces.EntityServiceInterface for testing
type MockEntityService struct {
	mock.Mock
}

func (m *MockEntityService) GetByID(ctx context.Context, entityID string, options interfaces.EntityGetOptions) (*interfaces.EntityWithRoom, error) {
	args := m.Called(ctx, entityID, options)
	return args.Get(0).(*interfaces.EntityWithRoom), args.Error(1)
}

func (m *MockEntityService) GetByRoom(ctx context.Context, roomID string, options interfaces.EntityGetAllOptions) ([]*interfaces.EntityWithRoom, error) {
	args := m.Called(ctx, roomID, options)
	return args.Get(0).([]*interfaces.EntityWithRoom), args.Error(1)
}

func (m *MockEntityService) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	args := m.Called(ctx, action)
	return args.Get(0).(*types.PMAControlResult), args.Error(1)
}

// MockRoomService implements interfaces.RoomServiceInterface for testing
type MockRoomService struct {
	mock.Mock
}

func (m *MockRoomService) GetRoomByID(ctx context.Context, roomID string) (*types.PMARoom, error) {
	args := m.Called(ctx, roomID)
	return args.Get(0).(*types.PMARoom), args.Error(1)
}

func (m *MockRoomService) GetAllRooms(ctx context.Context) ([]*types.PMARoom, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*types.PMARoom), args.Error(1)
}

// MockSystemService implements interfaces.SystemServiceInterface for testing
type MockSystemService struct {
	mock.Mock
}

func (m *MockSystemService) GetSystemStatus(ctx context.Context) (*interfaces.SystemStatus, error) {
	args := m.Called(ctx)
	return args.Get(0).(*interfaces.SystemStatus), args.Error(1)
}

func (m *MockSystemService) GetDeviceInfo(ctx context.Context) (*interfaces.DeviceInfo, error) {
	args := m.Called(ctx)
	return args.Get(0).(*interfaces.DeviceInfo), args.Error(1)
}

// MockEnergyService implements interfaces.EnergyServiceInterface for testing
type MockEnergyService struct {
	mock.Mock
}

func (m *MockEnergyService) GetCurrentEnergyData(ctx context.Context, deviceID string) (*interfaces.EnergyData, error) {
	args := m.Called(ctx, deviceID)
	return args.Get(0).(*interfaces.EnergyData), args.Error(1)
}

func (m *MockEnergyService) GetEnergySettings(ctx context.Context) (*interfaces.EnergySettings, error) {
	args := m.Called(ctx)
	return args.Get(0).(*interfaces.EnergySettings), args.Error(1)
}

// MockAutomationService implements interfaces.AutomationServiceInterface for testing
type MockAutomationService struct {
	mock.Mock
}

func (m *MockAutomationService) AddAutomationRule(ctx context.Context, rule *interfaces.AutomationRule) (*interfaces.AutomationResult, error) {
	args := m.Called(ctx, rule)
	return args.Get(0).(*interfaces.AutomationResult), args.Error(1)
}

func (m *MockAutomationService) ExecuteScene(ctx context.Context, sceneID string) (*interfaces.SceneResult, error) {
	args := m.Called(ctx, sceneID)
	return args.Get(0).(*interfaces.SceneResult), args.Error(1)
}

// Mock PMA Entity for testing
type MockPMAEntity struct {
	mock.Mock
}

func (m *MockPMAEntity) GetID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPMAEntity) GetType() types.PMAEntityType {
	args := m.Called()
	return args.Get(0).(types.PMAEntityType)
}

func (m *MockPMAEntity) GetFriendlyName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPMAEntity) GetIcon() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPMAEntity) GetState() types.PMAEntityState {
	args := m.Called()
	return args.Get(0).(types.PMAEntityState)
}

func (m *MockPMAEntity) GetAttributes() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

func (m *MockPMAEntity) GetLastUpdated() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *MockPMAEntity) GetCapabilities() []types.PMACapability {
	args := m.Called()
	return args.Get(0).([]types.PMACapability)
}

func (m *MockPMAEntity) HasCapability(capability types.PMACapability) bool {
	args := m.Called(capability)
	return args.Bool(0)
}

func (m *MockPMAEntity) CanControl() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockPMAEntity) GetAvailableActions() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *MockPMAEntity) ExecuteAction(action types.PMAControlAction) (*types.PMAControlResult, error) {
	args := m.Called(action)
	return args.Get(0).(*types.PMAControlResult), args.Error(1)
}

func (m *MockPMAEntity) GetRoomID() *string {
	args := m.Called()
	return args.Get(0).(*string)
}

func (m *MockPMAEntity) GetAreaID() *string {
	args := m.Called()
	return args.Get(0).(*string)
}

func (m *MockPMAEntity) GetDeviceID() *string {
	args := m.Called()
	return args.Get(0).(*string)
}

func (m *MockPMAEntity) GetMetadata() *types.PMAMetadata {
	args := m.Called()
	return args.Get(0).(*types.PMAMetadata)
}

func (m *MockPMAEntity) GetSource() types.PMASourceType {
	args := m.Called()
	return args.Get(0).(types.PMASourceType)
}

func (m *MockPMAEntity) IsAvailable() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockPMAEntity) GetQualityScore() float64 {
	args := m.Called()
	return args.Get(0).(float64)
}

func (m *MockPMAEntity) ToJSON() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}

// Test setup helper
func setupMCPTestExecutor() (*MCPToolExecutor, *MockEntityService, *MockRoomService, *MockSystemService, *MockEnergyService, *MockAutomationService) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests

	mockEntityService := &MockEntityService{}
	mockRoomService := &MockRoomService{}
	mockSystemService := &MockSystemService{}
	mockEnergyService := &MockEnergyService{}
	mockAutomationService := &MockAutomationService{}

	executor := NewMCPToolExecutor(logger)

	// Create wrappers with mock services
	entityWrapper := NewUnifiedEntityServiceWrapper(mockEntityService)
	roomWrapper := NewRoomServiceWrapper(mockRoomService)
	systemWrapper := NewSystemServiceWrapper(mockSystemService)
	energyWrapper := NewEnergyServiceWrapper(mockEnergyService)
	automationWrapper := NewAutomationServiceWrapper(mockAutomationService)

	executor.SetServices(entityWrapper, roomWrapper, systemWrapper, energyWrapper, automationWrapper)

	return executor, mockEntityService, mockRoomService, mockSystemService, mockEnergyService, mockAutomationService
}

// Test MCPToolExecutor creation
func TestNewMCPToolExecutor(t *testing.T) {
	logger := logrus.New()
	executor := NewMCPToolExecutor(logger)

	assert.NotNil(t, executor)
	assert.Equal(t, logger, executor.logger)
}

// Test GetEntityState tool execution
func TestExecuteGetEntityState(t *testing.T) {
	executor, mockEntityService, _, _, _, _ := setupMCPTestExecutor()

	// Setup mock entity
	mockEntity := &MockPMAEntity{}
	mockEntity.On("GetID").Return("light.test")
	mockEntity.On("GetState").Return(types.StateOn)
	mockEntity.On("GetFriendlyName").Return("Test Light")
	mockEntity.On("GetType").Return(types.EntityTypeLight)
	mockEntity.On("GetSource").Return(types.SourceHomeAssistant)
	mockEntity.On("GetAttributes").Return(map[string]interface{}{"brightness": 255})
	mockEntity.On("GetLastUpdated").Return(time.Now())
	mockEntity.On("GetCapabilities").Return([]types.PMACapability{types.CapabilityDimmable})
	mockEntity.On("GetMetadata").Return(&types.PMAMetadata{
		Source:       types.SourceHomeAssistant,
		LastSynced:   time.Now(),
		QualityScore: 1.0,
	})

	entityWithRoom := &interfaces.EntityWithRoom{
		Entity: mockEntity,
		Room:   nil,
		Area:   nil,
	}

	// Setup mock expectations
	mockEntityService.On("GetByID", mock.Anything, "light.test", mock.Anything).Return(entityWithRoom, nil)

	// Create a mock tool
	tool := &MCPTool{
		ID:      "get_entity_state",
		Name:    "GetEntityState",
		Handler: "GetEntityState",
		Schema: map[string]interface{}{
			"required": []interface{}{"entity_id"},
		},
	}

	// Test parameters
	params := map[string]interface{}{
		"entity_id": "light.test",
	}

	// Execute the tool
	result, err := executor.ExecuteTool(context.Background(), tool, params)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Nil(t, result.Error)

	// Verify the result structure
	entityResult, ok := result.Result.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "light.test", entityResult["entity_id"])
	assert.Equal(t, "on", entityResult["state"])
	assert.Equal(t, "Test Light", entityResult["friendly_name"])

	// Verify all mocks were called as expected
	mockEntityService.AssertExpectations(t)
	mockEntity.AssertExpectations(t)
}

// Test SetEntityState tool execution
func TestExecuteSetEntityState(t *testing.T) {
	executor, mockEntityService, _, _, _, _ := setupMCPTestExecutor()

	// Setup mock entity
	mockEntity := &MockPMAEntity{}
	mockEntity.On("GetID").Return("light.test")
	mockEntity.On("GetState").Return(types.StateOff)
	mockEntity.On("GetFriendlyName").Return("Test Light")
	mockEntity.On("GetType").Return(types.EntityTypeLight)
	mockEntity.On("GetSource").Return(types.SourceHomeAssistant)
	mockEntity.On("GetAttributes").Return(map[string]interface{}{"brightness": 0})
	mockEntity.On("GetLastUpdated").Return(time.Now())
	mockEntity.On("GetCapabilities").Return([]types.PMACapability{types.CapabilityDimmable})
	mockEntity.On("GetMetadata").Return(&types.PMAMetadata{
		Source:       types.SourceHomeAssistant,
		LastSynced:   time.Now(),
		QualityScore: 1.0,
	})

	entityWithRoom := &interfaces.EntityWithRoom{
		Entity: mockEntity,
		Room:   nil,
		Area:   nil,
	}

	controlResult := &types.PMAControlResult{
		Success:     true,
		EntityID:    "light.test",
		Action:      "turn_on",
		NewState:    types.StateOn,
		ProcessedAt: time.Now(),
	}

	// Setup mock expectations
	mockEntityService.On("GetByID", mock.Anything, "light.test", mock.Anything).Return(entityWithRoom, nil)
	mockEntityService.On("ExecuteAction", mock.Anything, mock.AnythingOfType("types.PMAControlAction")).Return(controlResult, nil)

	// Create a mock tool
	tool := &MCPTool{
		ID:      "set_entity_state",
		Name:    "SetEntityState",
		Handler: "SetEntityState",
		Schema: map[string]interface{}{
			"required": []interface{}{"entity_id", "state"},
		},
	}

	// Test parameters
	params := map[string]interface{}{
		"entity_id": "light.test",
		"state":     "on",
	}

	// Execute the tool
	result, err := executor.ExecuteTool(context.Background(), tool, params)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Nil(t, result.Error)

	// Verify all mocks were called as expected
	mockEntityService.AssertExpectations(t)
	mockEntity.AssertExpectations(t)
}

// Test GetSystemStatus tool execution
func TestExecuteGetSystemStatus(t *testing.T) {
	executor, _, _, mockSystemService, _, _ := setupMCPTestExecutor()

	// Setup mock system status
	systemStatus := &interfaces.SystemStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		DeviceID:  "test-device",
		CPU: &interfaces.CPUInfo{
			Usage: 15.5,
			Cores: 4,
			Model: "Test CPU",
		},
		Memory: &interfaces.MemoryInfo{
			Total:       8 * 1024 * 1024 * 1024,
			Used:        4 * 1024 * 1024 * 1024,
			UsedPercent: 50.0,
		},
	}

	// Setup mock expectations
	mockSystemService.On("GetSystemStatus", mock.Anything).Return(systemStatus, nil)

	// Create a mock tool
	tool := &MCPTool{
		ID:      "get_system_status",
		Name:    "GetSystemStatus",
		Handler: "GetSystemStatus",
		Schema:  map[string]interface{}{},
	}

	// Execute the tool
	result, err := executor.ExecuteTool(context.Background(), tool, map[string]interface{}{})

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Nil(t, result.Error)

	// Verify the result structure
	statusResult, ok := result.Result.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "healthy", statusResult["status"])
	assert.Equal(t, "test-device", statusResult["device_id"])

	// Verify all mocks were called as expected
	mockSystemService.AssertExpectations(t)
}

// Test GetEnergyData tool execution
func TestExecuteGetEnergyData(t *testing.T) {
	executor, _, _, _, mockEnergyService, _ := setupMCPTestExecutor()

	// Setup mock energy data
	energyData := &interfaces.EnergyData{
		Timestamp:             time.Now(),
		TotalPowerConsumption: 1250.5,
		TotalEnergyUsage:      30.2,
		TotalCost:             4.85,
	}

	// Setup mock expectations
	mockEnergyService.On("GetCurrentEnergyData", mock.Anything, "").Return(energyData, nil)

	// Create a mock tool
	tool := &MCPTool{
		ID:      "get_energy_data",
		Name:    "GetEnergyData",
		Handler: "GetEnergyData",
		Schema:  map[string]interface{}{},
	}

	// Execute the tool
	result, err := executor.ExecuteTool(context.Background(), tool, map[string]interface{}{})

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Nil(t, result.Error)

	// Verify the result structure
	energyResult, ok := result.Result.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 1250.5, energyResult["total_power_consumption"])
	assert.Equal(t, 30.2, energyResult["total_energy_usage"])

	// Verify all mocks were called as expected
	mockEnergyService.AssertExpectations(t)
}

// Test error handling
func TestExecuteToolWithInvalidHandler(t *testing.T) {
	executor, _, _, _, _, _ := setupMCPTestExecutor()

	// Create a mock tool with invalid handler
	tool := &MCPTool{
		ID:      "invalid_tool",
		Name:    "InvalidTool",
		Handler: "InvalidHandler",
		Schema:  map[string]interface{}{},
	}

	// Execute the tool
	result, err := executor.ExecuteTool(context.Background(), tool, map[string]interface{}{})

	// Assertions
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
	assert.NotNil(t, result.Error)
	assert.Contains(t, *result.Error, "unknown tool handler")
}

// Test parameter validation
func TestValidateParameters(t *testing.T) {
	executor := NewMCPToolExecutor(logrus.New())

	// Test tool with required parameters
	tool := &MCPTool{
		ID:   "test_tool",
		Name: "TestTool",
		Schema: map[string]interface{}{
			"required": []interface{}{"entity_id"},
			"properties": map[string]interface{}{
				"entity_id": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}

	// Test with missing required parameter
	err := executor.ValidateParameters(tool, map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required parameter 'entity_id' is missing")

	// Test with valid parameters
	err = executor.ValidateParameters(tool, map[string]interface{}{
		"entity_id": "light.test",
	})
	assert.NoError(t, err)

	// Test with invalid parameter type
	err = executor.ValidateParameters(tool, map[string]interface{}{
		"entity_id": 123, // Should be string
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid type")
}
