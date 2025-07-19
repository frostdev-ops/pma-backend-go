package registries

import (
	"context"
	"testing"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPMAAdapter is a mock implementation of PMAAdapter for testing
type MockPMAAdapter struct {
	mock.Mock
}

func (m *MockPMAAdapter) GetID() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPMAAdapter) GetSourceType() types.PMASourceType {
	args := m.Called()
	return args.Get(0).(types.PMASourceType)
}

func (m *MockPMAAdapter) GetName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPMAAdapter) GetVersion() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPMAAdapter) Connect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPMAAdapter) Disconnect(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPMAAdapter) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockPMAAdapter) GetStatus() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockPMAAdapter) ConvertEntity(sourceEntity interface{}) (types.PMAEntity, error) {
	args := m.Called(sourceEntity)
	return args.Get(0).(types.PMAEntity), args.Error(1)
}

func (m *MockPMAAdapter) ConvertEntities(sourceEntities []interface{}) ([]types.PMAEntity, error) {
	args := m.Called(sourceEntities)
	return args.Get(0).([]types.PMAEntity), args.Error(1)
}

func (m *MockPMAAdapter) ConvertRoom(sourceRoom interface{}) (*types.PMARoom, error) {
	args := m.Called(sourceRoom)
	return args.Get(0).(*types.PMARoom), args.Error(1)
}

func (m *MockPMAAdapter) ConvertArea(sourceArea interface{}) (*types.PMAArea, error) {
	args := m.Called(sourceArea)
	return args.Get(0).(*types.PMAArea), args.Error(1)
}

func (m *MockPMAAdapter) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	args := m.Called(ctx, action)
	return args.Get(0).(*types.PMAControlResult), args.Error(1)
}

func (m *MockPMAAdapter) SyncEntities(ctx context.Context) ([]types.PMAEntity, error) {
	args := m.Called(ctx)
	return args.Get(0).([]types.PMAEntity), args.Error(1)
}

func (m *MockPMAAdapter) SyncRooms(ctx context.Context) ([]*types.PMARoom, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*types.PMARoom), args.Error(1)
}

func (m *MockPMAAdapter) GetLastSyncTime() *time.Time {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*time.Time)
}

func (m *MockPMAAdapter) GetSupportedEntityTypes() []types.PMAEntityType {
	args := m.Called()
	return args.Get(0).([]types.PMAEntityType)
}

func (m *MockPMAAdapter) GetSupportedCapabilities() []types.PMACapability {
	args := m.Called()
	return args.Get(0).([]types.PMACapability)
}

func (m *MockPMAAdapter) SupportsRealtime() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockPMAAdapter) GetHealth() *types.AdapterHealth {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*types.AdapterHealth)
}

func (m *MockPMAAdapter) GetMetrics() *types.AdapterMetrics {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*types.AdapterMetrics)
}

func TestNewDefaultAdapterRegistry(t *testing.T) {
	logger := logrus.New()
	registry := NewDefaultAdapterRegistry(logger)

	assert.NotNil(t, registry)
	assert.Empty(t, registry.GetAllAdapters())
}

func TestAdapterRegistry_RegisterAdapter(t *testing.T) {
	logger := logrus.New()
	registry := NewDefaultAdapterRegistry(logger)

	// Create mock adapter
	mockAdapter := &MockPMAAdapter{}
	mockAdapter.On("GetID").Return("test-adapter")
	mockAdapter.On("GetSourceType").Return(types.SourceHomeAssistant)
	mockAdapter.On("GetVersion").Return("1.0.0")

	// Test successful registration
	err := registry.RegisterAdapter(mockAdapter)
	assert.NoError(t, err)

	// Test duplicate registration
	err = registry.RegisterAdapter(mockAdapter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")

	// Test nil adapter
	err = registry.RegisterAdapter(nil)
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidAdapter, err)

	mockAdapter.AssertExpectations(t)
}

func TestAdapterRegistry_GetAdapter(t *testing.T) {
	logger := logrus.New()
	registry := NewDefaultAdapterRegistry(logger)

	// Create mock adapter
	mockAdapter := &MockPMAAdapter{}
	mockAdapter.On("GetID").Return("test-adapter")
	mockAdapter.On("GetSourceType").Return(types.SourceHomeAssistant)
	mockAdapter.On("GetVersion").Return("1.0.0")

	// Test adapter not found
	_, err := registry.GetAdapter("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Register adapter
	err = registry.RegisterAdapter(mockAdapter)
	assert.NoError(t, err)

	// Test successful retrieval
	retrieved, err := registry.GetAdapter("test-adapter")
	assert.NoError(t, err)
	assert.Equal(t, mockAdapter, retrieved)

	mockAdapter.AssertExpectations(t)
}

func TestAdapterRegistry_GetAdapterBySource(t *testing.T) {
	logger := logrus.New()
	registry := NewDefaultAdapterRegistry(logger)

	// Create mock adapter
	mockAdapter := &MockPMAAdapter{}
	mockAdapter.On("GetID").Return("test-adapter")
	mockAdapter.On("GetSourceType").Return(types.SourceHomeAssistant)
	mockAdapter.On("GetVersion").Return("1.0.0")

	// Test source not found
	_, err := registry.GetAdapterBySource(types.SourceHomeAssistant)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Register adapter
	err = registry.RegisterAdapter(mockAdapter)
	assert.NoError(t, err)

	// Test successful retrieval
	retrieved, err := registry.GetAdapterBySource(types.SourceHomeAssistant)
	assert.NoError(t, err)
	assert.Equal(t, mockAdapter, retrieved)

	mockAdapter.AssertExpectations(t)
}

func TestAdapterRegistry_UnregisterAdapter(t *testing.T) {
	logger := logrus.New()
	registry := NewDefaultAdapterRegistry(logger)

	// Test unregistering non-existent adapter
	err := registry.UnregisterAdapter("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Create and register mock adapter
	mockAdapter := &MockPMAAdapter{}
	mockAdapter.On("GetID").Return("test-adapter")
	mockAdapter.On("GetSourceType").Return(types.SourceHomeAssistant)
	mockAdapter.On("GetVersion").Return("1.0.0")

	err = registry.RegisterAdapter(mockAdapter)
	assert.NoError(t, err)

	// Verify adapter is registered
	_, err = registry.GetAdapter("test-adapter")
	assert.NoError(t, err)

	// Test successful unregistration
	err = registry.UnregisterAdapter("test-adapter")
	assert.NoError(t, err)

	// Verify adapter is no longer registered
	_, err = registry.GetAdapter("test-adapter")
	assert.Error(t, err)

	mockAdapter.AssertExpectations(t)
}

func TestAdapterRegistry_GetConnectedAdapters(t *testing.T) {
	logger := logrus.New()
	registry := NewDefaultAdapterRegistry(logger)

	// Create mock adapters
	mockAdapter1 := &MockPMAAdapter{}
	mockAdapter1.On("GetID").Return("adapter1")
	mockAdapter1.On("GetSourceType").Return(types.SourceHomeAssistant)
	mockAdapter1.On("GetVersion").Return("1.0.0")
	mockAdapter1.On("IsConnected").Return(true)

	mockAdapter2 := &MockPMAAdapter{}
	mockAdapter2.On("GetID").Return("adapter2")
	mockAdapter2.On("GetSourceType").Return(types.SourceRing)
	mockAdapter2.On("GetVersion").Return("1.0.0")
	mockAdapter2.On("IsConnected").Return(false)

	// Register adapters
	err := registry.RegisterAdapter(mockAdapter1)
	assert.NoError(t, err)
	err = registry.RegisterAdapter(mockAdapter2)
	assert.NoError(t, err)

	// Test getting connected adapters
	connectedAdapters := registry.GetConnectedAdapters()
	assert.Len(t, connectedAdapters, 1)
	assert.Equal(t, mockAdapter1, connectedAdapters[0])

	mockAdapter1.AssertExpectations(t)
	mockAdapter2.AssertExpectations(t)
}

func TestAdapterRegistry_GetAdapterMetrics(t *testing.T) {
	logger := logrus.New()
	registry := NewDefaultAdapterRegistry(logger)

	// Test metrics for non-existent adapter
	_, err := registry.GetAdapterMetrics("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// Create mock adapter
	mockAdapter := &MockPMAAdapter{}
	mockAdapter.On("GetID").Return("test-adapter")
	mockAdapter.On("GetSourceType").Return(types.SourceHomeAssistant)
	mockAdapter.On("GetVersion").Return("1.0.0")

	expectedMetrics := &types.AdapterMetrics{
		EntitiesManaged:     10,
		ActionsExecuted:     100,
		SuccessfulActions:   95,
		FailedActions:       5,
		AverageResponseTime: time.Millisecond * 500,
	}
	mockAdapter.On("GetMetrics").Return(expectedMetrics)

	// Register adapter
	err = registry.RegisterAdapter(mockAdapter)
	assert.NoError(t, err)

	// Test getting metrics
	metrics, err := registry.GetAdapterMetrics("test-adapter")
	assert.NoError(t, err)
	assert.Equal(t, expectedMetrics, metrics)

	mockAdapter.AssertExpectations(t)
}

func TestAdapterRegistry_SourceOverride(t *testing.T) {
	logger := logrus.New()
	registry := NewDefaultAdapterRegistry(logger)

	// Create first adapter
	mockAdapter1 := &MockPMAAdapter{}
	mockAdapter1.On("GetID").Return("adapter1")
	mockAdapter1.On("GetSourceType").Return(types.SourceHomeAssistant)
	mockAdapter1.On("GetVersion").Return("1.0.0")

	// Create second adapter for same source
	mockAdapter2 := &MockPMAAdapter{}
	mockAdapter2.On("GetID").Return("adapter2")
	mockAdapter2.On("GetSourceType").Return(types.SourceHomeAssistant)
	mockAdapter2.On("GetVersion").Return("2.0.0")

	// Register first adapter
	err := registry.RegisterAdapter(mockAdapter1)
	assert.NoError(t, err)

	// Verify first adapter is registered
	retrieved, err := registry.GetAdapterBySource(types.SourceHomeAssistant)
	assert.NoError(t, err)
	assert.Equal(t, mockAdapter1, retrieved)

	// Register second adapter (should replace first)
	err = registry.RegisterAdapter(mockAdapter2)
	assert.NoError(t, err)

	// Verify second adapter replaced first
	retrieved, err = registry.GetAdapterBySource(types.SourceHomeAssistant)
	assert.NoError(t, err)
	assert.Equal(t, mockAdapter2, retrieved)

	// Verify first adapter is no longer accessible by ID
	_, err = registry.GetAdapter("adapter1")
	assert.Error(t, err)

	mockAdapter1.AssertExpectations(t)
	mockAdapter2.AssertExpectations(t)
}
