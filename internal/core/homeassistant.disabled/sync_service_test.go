package homeassistant

import (
	"context"
	"testing"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/internal/core/entities"
	"github.com/frostdev-ops/pma-backend-go/internal/core/rooms"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations for testing

type MockHAClient struct {
	mock.Mock
}

func (m *MockHAClient) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockHAClient) IsConnected() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockHAClient) GetStates(ctx context.Context) ([]homeassistant.EntityState, error) {
	args := m.Called(ctx)
	return args.Get(0).([]homeassistant.EntityState), args.Error(1)
}

func (m *MockHAClient) GetState(ctx context.Context, entityID string) (*homeassistant.EntityState, error) {
	args := m.Called(ctx, entityID)
	return args.Get(0).(*homeassistant.EntityState), args.Error(1)
}

func (m *MockHAClient) GetAreas(ctx context.Context) ([]homeassistant.Area, error) {
	args := m.Called(ctx)
	return args.Get(0).([]homeassistant.Area), args.Error(1)
}

func (m *MockHAClient) CallService(ctx context.Context, domain, service string, data map[string]interface{}) error {
	args := m.Called(ctx, domain, service, data)
	return args.Error(0)
}

func (m *MockHAClient) SubscribeToStateChanges(entityID string, handler homeassistant.StateChangeHandler) (int, error) {
	args := m.Called(entityID, handler)
	return args.Int(0), args.Error(1)
}

type MockEntityService struct {
	mock.Mock
}

func (m *MockEntityService) GetByID(ctx context.Context, entityID string, includeRoom bool) (*entities.EntityWithRoom, error) {
	args := m.Called(ctx, entityID, includeRoom)
	return args.Get(0).(*entities.EntityWithRoom), args.Error(1)
}

func (m *MockEntityService) CreateOrUpdate(ctx context.Context, entity *models.Entity) error {
	args := m.Called(ctx, entity)
	return args.Error(0)
}

type MockRoomService struct {
	mock.Mock
}

func (m *MockRoomService) GetByID(ctx context.Context, roomID int, includeEntities bool) (*rooms.RoomWithEntities, error) {
	args := m.Called(ctx, roomID, includeEntities)
	return args.Get(0).(*rooms.RoomWithEntities), args.Error(1)
}

func (m *MockRoomService) Create(ctx context.Context, room *models.Room) error {
	args := m.Called(ctx, room)
	return args.Error(0)
}

type MockConfigRepo struct {
	mock.Mock
}

func (m *MockConfigRepo) GetValue(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

type MockWSHub struct {
	mock.Mock
}

func (m *MockWSHub) BroadcastToAll(message interface{}) {
	m.Called(message)
}

// Test fixtures

func createTestSyncService() (*SyncService, *MockHAClient, *MockEntityService, *MockRoomService) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	mockHAClient := &MockHAClient{}
	mockEntitySvc := &MockEntityService{}
	mockRoomSvc := &MockRoomService{}
	mockConfigRepo := &MockConfigRepo{}
	mockWSHub := &MockWSHub{}

	config := &SyncConfig{
		Enabled:              true,
		FullSyncInterval:     time.Hour,
		SupportedDomains:     []string{"light", "switch", "sensor"},
		ConflictResolution:   "homeassistant_wins",
		BatchSize:            10,
		RetryAttempts:        3,
		RetryDelay:           time.Second,
		EventBufferSize:      100,
		EventProcessingDelay: 100 * time.Millisecond,
	}

	syncService := NewSyncService(
		mockHAClient,
		mockEntitySvc,
		mockRoomSvc,
		mockConfigRepo,
		mockWSHub,
		logger,
		config,
	)

	return syncService, mockHAClient, mockEntitySvc, mockRoomSvc
}

func createTestEntityState(entityID, domain, state string) homeassistant.EntityState {
	return homeassistant.EntityState{
		EntityID: entityID,
		State:    state,
		Attributes: map[string]interface{}{
			"friendly_name": "Test " + domain,
		},
		LastUpdated: time.Now(),
	}
}

// Tests

func TestSyncService_NewSyncService(t *testing.T) {
	service, _, _, _ := createTestSyncService()

	assert.NotNil(t, service)
	assert.NotNil(t, service.config)
	assert.NotNil(t, service.mapper)
	assert.Equal(t, 100, cap(service.eventBuffer))
	assert.False(t, service.isRunning)
}

func TestSyncService_Start_Success(t *testing.T) {
	service, mockHA, _, _ := createTestSyncService()

	mockHA.On("Initialize", mock.Anything).Return(nil)
	mockHA.On("SubscribeToStateChanges", "", mock.Anything).Return(1, nil)
	mockHA.On("GetAreas", mock.Anything).Return([]homeassistant.Area{}, nil)
	mockHA.On("GetStates", mock.Anything).Return([]homeassistant.EntityState{}, nil)

	ctx := context.Background()
	err := service.Start(ctx)

	assert.NoError(t, err)
	assert.True(t, service.IsRunning())

	// Cleanup
	service.Stop(ctx)
	mockHA.AssertExpectations(t)
}

func TestSyncService_Start_AlreadyRunning(t *testing.T) {
	service, mockHA, _, _ := createTestSyncService()

	mockHA.On("Initialize", mock.Anything).Return(nil)
	mockHA.On("SubscribeToStateChanges", "", mock.Anything).Return(1, nil)
	mockHA.On("GetAreas", mock.Anything).Return([]homeassistant.Area{}, nil)
	mockHA.On("GetStates", mock.Anything).Return([]homeassistant.EntityState{}, nil)

	ctx := context.Background()

	// Start service first time
	err := service.Start(ctx)
	assert.NoError(t, err)

	// Try to start again
	err = service.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")

	// Cleanup
	service.Stop(ctx)
}

func TestSyncService_Stop_Success(t *testing.T) {
	service, mockHA, _, _ := createTestSyncService()

	mockHA.On("Initialize", mock.Anything).Return(nil)
	mockHA.On("SubscribeToStateChanges", "", mock.Anything).Return(1, nil)
	mockHA.On("GetAreas", mock.Anything).Return([]homeassistant.Area{}, nil)
	mockHA.On("GetStates", mock.Anything).Return([]homeassistant.EntityState{}, nil)

	ctx := context.Background()

	// Start service
	err := service.Start(ctx)
	assert.NoError(t, err)
	assert.True(t, service.IsRunning())

	// Stop service
	err = service.Stop(ctx)
	assert.NoError(t, err)
	assert.False(t, service.IsRunning())
}

func TestSyncService_SyncEntity_Success(t *testing.T) {
	service, mockHA, mockEntity, _ := createTestSyncService()

	entityState := createTestEntityState("light.test", "light", "on")
	mockHA.On("GetState", mock.Anything, "light.test").Return(&entityState, nil)

	// Mock entity doesn't exist, so create new
	mockEntity.On("GetByID", mock.Anything, "light.test", false).Return(nil, assert.AnError)
	mockEntity.On("CreateOrUpdate", mock.Anything, mock.AnythingOfType("*models.Entity")).Return(nil)

	service.isRunning = true // Bypass start for test

	ctx := context.Background()
	err := service.SyncEntity(ctx, "light.test")

	assert.NoError(t, err)
	mockHA.AssertExpectations(t)
	mockEntity.AssertExpectations(t)
}

func TestSyncService_ShouldProcessEntity(t *testing.T) {
	service, _, _, _ := createTestSyncService()

	tests := []struct {
		entityID string
		expected bool
	}{
		{"light.living_room", true},  // light is supported
		{"switch.bedroom", true},     // switch is supported
		{"sensor.temperature", true}, // sensor is supported
		{"camera.front_door", false}, // camera not supported
		{"invalid_id", false},        // invalid format
	}

	for _, test := range tests {
		t.Run(test.entityID, func(t *testing.T) {
			result := service.shouldProcessEntity(test.entityID)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestSyncService_ExtractDomain(t *testing.T) {
	service, _, _, _ := createTestSyncService()

	tests := []struct {
		entityID string
		expected string
	}{
		{"light.living_room", "light"},
		{"switch.bedroom", "switch"},
		{"sensor.temperature", "sensor"},
		{"invalid_id", "invalid_id"}, // no dot
	}

	for _, test := range tests {
		t.Run(test.entityID, func(t *testing.T) {
			result := service.extractDomain(test.entityID)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestSyncService_FilterSupportedEntities(t *testing.T) {
	service, _, _, _ := createTestSyncService()

	entities := []*homeassistant.EntityState{
		{EntityID: "light.living_room"},  // supported
		{EntityID: "switch.bedroom"},     // supported
		{EntityID: "camera.front_door"},  // not supported
		{EntityID: "sensor.temperature"}, // supported
	}

	filtered := service.filterSupportedEntities(entities)

	assert.Len(t, filtered, 3)
	assert.Equal(t, "light.living_room", filtered[0].EntityID)
	assert.Equal(t, "switch.bedroom", filtered[1].EntityID)
	assert.Equal(t, "sensor.temperature", filtered[2].EntityID)
}

func TestSyncService_CreateBatches(t *testing.T) {
	service, _, _, _ := createTestSyncService()

	entities := make([]*homeassistant.EntityState, 25)
	for i := 0; i < 25; i++ {
		entities[i] = &homeassistant.EntityState{EntityID: "entity_" + string(rune(i))}
	}

	batches := service.createBatches(entities, 10)

	assert.Len(t, batches, 3)
	assert.Len(t, batches[0], 10)
	assert.Len(t, batches[1], 10)
	assert.Len(t, batches[2], 5)
}

func TestSyncService_GetSyncStats(t *testing.T) {
	service, _, _, _ := createTestSyncService()

	// Set some test data
	service.stats.EntitiesSynced = 100
	service.stats.EventsProcessed = 500

	stats := service.GetSyncStats()

	assert.Equal(t, 100, stats.EntitiesSynced)
	assert.Equal(t, int64(500), stats.EventsProcessed)
}

func TestSyncService_RecordError(t *testing.T) {
	service, _, _, _ := createTestSyncService()

	err := assert.AnError
	service.recordError("test_error", "entity.test", "test_operation", err, true)

	stats := service.GetSyncStats()
	assert.Len(t, stats.SyncErrors, 1)
	assert.Equal(t, "test_error", stats.SyncErrors[0].Type)
	assert.Equal(t, "entity.test", stats.SyncErrors[0].EntityID)
	assert.True(t, stats.SyncErrors[0].Retryable)
}

func TestSyncService_UpdateStats(t *testing.T) {
	service, _, _, _ := createTestSyncService()

	service.updateStats(func(stats *SyncStats) {
		stats.EntitiesSynced = 42
		stats.EventsProcessed = 123
	})

	assert.Equal(t, 42, service.stats.EntitiesSynced)
	assert.Equal(t, int64(123), service.stats.EventsProcessed)
}

// Benchmark tests

func BenchmarkSyncService_ShouldProcessEntity(b *testing.B) {
	service, _, _, _ := createTestSyncService()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.shouldProcessEntity("light.living_room")
	}
}

func BenchmarkSyncService_FilterSupportedEntities(b *testing.B) {
	service, _, _, _ := createTestSyncService()

	entities := make([]*homeassistant.EntityState, 1000)
	for i := 0; i < 1000; i++ {
		domain := "light"
		if i%3 == 0 {
			domain = "camera" // unsupported
		}
		entities[i] = &homeassistant.EntityState{
			EntityID: domain + ".entity_" + string(rune(i)),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.filterSupportedEntities(entities)
	}
}

// Integration test helpers

func TestSyncService_IntegrationSetup(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test demonstrates how to set up integration tests
	// with a real Home Assistant instance

	t.Log("Integration tests would require:")
	t.Log("1. Running Home Assistant instance")
	t.Log("2. Valid authentication token")
	t.Log("3. Test entities with known states")
	t.Log("4. Network connectivity")
}

// Error scenarios

func TestSyncService_SyncEntity_NotRunning(t *testing.T) {
	service, _, _, _ := createTestSyncService()

	ctx := context.Background()
	err := service.SyncEntity(ctx, "light.test")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not running")
}

func TestSyncService_SyncEntity_HAError(t *testing.T) {
	service, mockHA, _, _ := createTestSyncService()

	mockHA.On("GetState", mock.Anything, "light.test").Return(nil, assert.AnError)

	service.isRunning = true // Bypass start for test

	ctx := context.Background()
	err := service.SyncEntity(ctx, "light.test")

	assert.Error(t, err)
	mockHA.AssertExpectations(t)
}
