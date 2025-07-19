package integration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
)

type MockPMAAdapter struct {
	id         string
	sourceType types.PMASourceType
	connected  bool
	entities   map[string]types.PMAEntity
	lastSync   *time.Time
	mutex      sync.RWMutex
	actionLog  []types.PMAControlAction
	health     *types.AdapterHealth
	metrics    *types.AdapterMetrics
	startTime  time.Time
}

func NewMockPMAAdapter() *MockPMAAdapter {
	return &MockPMAAdapter{
		id:         "mock_adapter",
		sourceType: types.SourceHomeAssistant,
		entities:   make(map[string]types.PMAEntity),
		actionLog:  make([]types.PMAControlAction, 0),
		startTime:  time.Now(),
		health: &types.AdapterHealth{
			IsHealthy:       true,
			LastHealthCheck: time.Now(),
			Issues:          []string{},
			ResponseTime:    50 * time.Millisecond,
			ErrorRate:       0.0,
		},
		metrics: &types.AdapterMetrics{
			EntitiesManaged:     0,
			RoomsManaged:        0,
			ActionsExecuted:     0,
			SuccessfulActions:   0,
			FailedActions:       0,
			AverageResponseTime: 50 * time.Millisecond,
			SyncErrors:          0,
			Uptime:              0,
		},
	}
}

// Implement PMAAdapter interface
func (m *MockPMAAdapter) GetID() string                      { return m.id }
func (m *MockPMAAdapter) GetSourceType() types.PMASourceType { return m.sourceType }
func (m *MockPMAAdapter) GetName() string                    { return "Mock Adapter" }
func (m *MockPMAAdapter) GetVersion() string                 { return "1.0.0" }

func (m *MockPMAAdapter) Connect(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Simulate connection delay
	time.Sleep(100 * time.Millisecond)

	m.connected = true

	// Add test entities
	roomID := "living_room"
	areaID := "ground_floor"

	m.entities["light.test_light"] = &types.PMALightEntity{
		PMABaseEntity: &types.PMABaseEntity{
			ID:           "ha_light.test_light",
			Type:         types.EntityTypeLight,
			FriendlyName: "Test Light",
			State:        types.StateOff,
			LastUpdated:  time.Now(),
			Available:    true,
			Capabilities: []types.PMACapability{types.CapabilityDimmable},
			RoomID:       &roomID,
			AreaID:       &areaID,
			Metadata: &types.PMAMetadata{
				Source:         m.sourceType,
				SourceEntityID: "light.test_light",
				QualityScore:   0.9,
				LastSynced:     time.Now(),
			},
		},
		Brightness: intPtr(0),
	}

	m.entities["sensor.test_sensor"] = &types.PMASensorEntity{
		PMABaseEntity: &types.PMABaseEntity{
			ID:           "ha_sensor.test_sensor",
			Type:         types.EntityTypeSensor,
			FriendlyName: "Test Sensor",
			State:        types.StateActive,
			LastUpdated:  time.Now(),
			Available:    true,
			RoomID:       &roomID,
			Metadata: &types.PMAMetadata{
				Source:         m.sourceType,
				SourceEntityID: "sensor.test_sensor",
				QualityScore:   0.95,
				LastSynced:     time.Now(),
			},
		},
		Unit:         "°C",
		DeviceClass:  "temperature",
		NumericValue: float64Ptr(22.5),
	}

	// Add conflict test entity
	m.entities["light.conflict_test"] = &types.PMALightEntity{
		PMABaseEntity: &types.PMABaseEntity{
			ID:           "ha_light.conflict_test",
			Type:         types.EntityTypeLight,
			FriendlyName: "Conflict Test Light",
			State:        types.StateOn,
			LastUpdated:  time.Now(),
			Available:    true,
			RoomID:       &roomID,
			Metadata: &types.PMAMetadata{
				Source:         m.sourceType,
				SourceEntityID: "light.conflict_test",
				QualityScore:   0.8,
				LastSynced:     time.Now(),
			},
		},
	}

	// Update metrics
	m.metrics.EntitiesManaged = len(m.entities)
	m.metrics.Uptime = time.Since(m.startTime)

	return nil
}

func (m *MockPMAAdapter) Disconnect(ctx context.Context) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.connected = false
	return nil
}

func (m *MockPMAAdapter) IsConnected() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.connected
}

func (m *MockPMAAdapter) GetStatus() string {
	if m.IsConnected() {
		return "connected"
	}
	return "disconnected"
}

func (m *MockPMAAdapter) ConvertEntity(sourceEntity interface{}) (types.PMAEntity, error) {
	// Mock implementation
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *MockPMAAdapter) ConvertEntities(sourceEntities []interface{}) ([]types.PMAEntity, error) {
	// Mock implementation
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *MockPMAAdapter) ConvertRoom(sourceRoom interface{}) (*types.PMARoom, error) {
	// Mock implementation
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *MockPMAAdapter) ConvertArea(sourceArea interface{}) (*types.PMAArea, error) {
	// Mock implementation
	return nil, fmt.Errorf("not implemented in mock")
}

func (m *MockPMAAdapter) SyncEntities(ctx context.Context) ([]types.PMAEntity, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if !m.connected {
		return nil, fmt.Errorf("adapter not connected")
	}

	entities := make([]types.PMAEntity, 0, len(m.entities))
	for _, entity := range m.entities {
		entities = append(entities, entity)
	}

	now := time.Now()
	m.lastSync = &now

	return entities, nil
}

func (m *MockPMAAdapter) SyncRooms(ctx context.Context) ([]*types.PMARoom, error) {
	if !m.connected {
		return nil, fmt.Errorf("adapter not connected")
	}

	// Return a mock room
	rooms := []*types.PMARoom{
		{
			ID:          "living_room",
			Name:        "Living Room",
			Description: "Main living area",
			EntityIDs:   []string{"ha_light.test_light", "ha_sensor.test_sensor"},
			ParentID:    stringPtr("ground_floor"),
			Metadata: &types.PMAMetadata{
				Source:       m.sourceType,
				QualityScore: 1.0,
				LastSynced:   time.Now(),
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	return rooms, nil
}

func (m *MockPMAAdapter) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Log the action
	m.actionLog = append(m.actionLog, action)
	m.metrics.ActionsExecuted++

	// Find the entity
	entityID := action.EntityID[3:] // Remove "ha_" prefix
	entity, exists := m.entities[entityID]
	if !exists {
		m.metrics.FailedActions++
		return &types.PMAControlResult{
			Success:  false,
			EntityID: action.EntityID,
			Action:   action.Action,
			Error: &types.PMAError{
				Code:     "ENTITY_NOT_FOUND",
				Message:  "Entity not found in mock adapter",
				Source:   m.GetID(),
				EntityID: action.EntityID,
			},
		}, nil
	}

	// Simulate action execution
	switch action.Action {
	case "turn_on":
		if light, ok := entity.(*types.PMALightEntity); ok {
			light.State = types.StateOn
			if brightness, ok := action.Parameters["brightness"].(int); ok {
				light.Brightness = &brightness
			}
			light.LastUpdated = time.Now()
		} else {
			m.metrics.FailedActions++
			return &types.PMAControlResult{
				Success:  false,
				EntityID: action.EntityID,
				Action:   action.Action,
				Error: &types.PMAError{
					Code:     "UNSUPPORTED_ACTION",
					Message:  fmt.Sprintf("Action %s not supported for entity type %s", action.Action, entity.GetType()),
					Source:   m.GetID(),
					EntityID: action.EntityID,
				},
			}, nil
		}
	case "turn_off":
		if baseEntity, ok := entity.(*types.PMALightEntity); ok {
			baseEntity.State = types.StateOff
			baseEntity.LastUpdated = time.Now()
		} else {
			m.metrics.FailedActions++
			return &types.PMAControlResult{
				Success:  false,
				EntityID: action.EntityID,
				Action:   action.Action,
				Error: &types.PMAError{
					Code:     "UNSUPPORTED_ACTION",
					Message:  fmt.Sprintf("Action %s not supported for entity type %s", action.Action, entity.GetType()),
					Source:   m.GetID(),
					EntityID: action.EntityID,
				},
			}, nil
		}
	default:
		m.metrics.FailedActions++
		return &types.PMAControlResult{
			Success:  false,
			EntityID: action.EntityID,
			Action:   action.Action,
			Error: &types.PMAError{
				Code:     "UNSUPPORTED_ACTION",
				Message:  fmt.Sprintf("Action %s not supported", action.Action),
				Source:   m.GetID(),
				EntityID: action.EntityID,
			},
		}, nil
	}

	m.metrics.SuccessfulActions++
	return &types.PMAControlResult{
		Success:     true,
		EntityID:    action.EntityID,
		Action:      action.Action,
		NewState:    entity.GetState(),
		ProcessedAt: time.Now(),
	}, nil
}

func (m *MockPMAAdapter) GetLastSyncTime() *time.Time {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.lastSync
}

func (m *MockPMAAdapter) GetSupportedEntityTypes() []types.PMAEntityType {
	return []types.PMAEntityType{
		types.EntityTypeLight,
		types.EntityTypeSwitch,
		types.EntityTypeSensor,
	}
}

func (m *MockPMAAdapter) GetSupportedCapabilities() []types.PMACapability {
	return []types.PMACapability{
		types.CapabilityDimmable,
		types.CapabilityTemperature,
		types.CapabilityBrightness,
	}
}

func (m *MockPMAAdapter) SupportsRealtime() bool {
	return false
}

func (m *MockPMAAdapter) GetHealth() *types.AdapterHealth {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Update health check timestamp
	m.health.LastHealthCheck = time.Now()
	m.health.ResponseTime = 50 * time.Millisecond

	return m.health
}

func (m *MockPMAAdapter) GetMetrics() *types.AdapterMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Update uptime
	m.metrics.Uptime = time.Since(m.startTime)

	// Calculate success rate
	if m.metrics.ActionsExecuted > 0 {
		successRate := float64(m.metrics.SuccessfulActions) / float64(m.metrics.ActionsExecuted)
		m.metrics.AverageResponseTime = time.Duration(float64(50*time.Millisecond) * (1 + (1 - successRate)))
	}

	return m.metrics
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

func stringPtr(s string) *string {
	return &s
}

// NewLargeMockAdapter creates a mock adapter with many entities for performance testing
func NewLargeMockAdapter(entityCount int) *MockPMAAdapter {
	adapter := NewMockPMAAdapter()
	adapter.id = "large_mock_adapter"

	// Add many entities
	for i := 0; i < entityCount; i++ {
		entityID := fmt.Sprintf("light.test_light_%d", i)
		adapter.entities[entityID] = &types.PMALightEntity{
			PMABaseEntity: &types.PMABaseEntity{
				ID:           fmt.Sprintf("ha_%s", entityID),
				Type:         types.EntityTypeLight,
				FriendlyName: fmt.Sprintf("Test Light %d", i),
				State:        types.StateOff,
				LastUpdated:  time.Now(),
				Available:    true,
				Capabilities: []types.PMACapability{types.CapabilityDimmable},
				Metadata: &types.PMAMetadata{
					Source:         adapter.sourceType,
					SourceEntityID: entityID,
					QualityScore:   0.9,
					LastSynced:     time.Now(),
				},
			},
			Brightness: intPtr(0),
		}
	}

	return adapter
}
