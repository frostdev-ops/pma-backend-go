package integration

import (
	"context"
	"testing"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type PMAIntegrationTestSuite struct {
	suite.Suite
	logger         *logrus.Logger
	config         *config.Config
	unifiedService *unified.UnifiedEntityService
	testAdapter    *MockPMAAdapter
	typeRegistry   *types.PMATypeRegistry
}

func (suite *PMAIntegrationTestSuite) SetupSuite() {
	// Initialize logger
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.DebugLevel)

	// Create test config
	suite.config = &config.Config{
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}

	// Create registries
	suite.typeRegistry = types.NewPMATypeRegistry(suite.logger)

	// Create unified service
	suite.unifiedService = unified.NewUnifiedEntityService(
		suite.typeRegistry,
		suite.logger,
	)

	// Create and register mock adapter
	suite.testAdapter = NewMockPMAAdapter()
	err := suite.unifiedService.RegisterAdapter(suite.testAdapter)
	suite.Require().NoError(err)
}

func (suite *PMAIntegrationTestSuite) TestCompleteEntityFlow() {
	ctx := context.Background()

	// 1. Connect adapter
	err := suite.testAdapter.Connect(ctx)
	suite.Assert().NoError(err)
	suite.Assert().True(suite.testAdapter.IsConnected())

	// 2. Sync entities from adapter
	syncResults, err := suite.unifiedService.SyncFromSource(ctx, types.SourceHomeAssistant)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(syncResults)
	suite.Assert().True(syncResults.Success)
	suite.Assert().Greater(syncResults.EntitiesFound, 0)

	// 3. Get all entities
	options := unified.GetAllOptions{
		IncludeRoom: true,
	}
	entities, err := suite.unifiedService.GetAll(ctx, options)
	suite.Assert().NoError(err)
	suite.Assert().NotEmpty(entities)

	// 4. Get specific entity
	testEntityID := "ha_light.test_light"
	entity, err := suite.unifiedService.GetByID(ctx, testEntityID, unified.GetEntityOptions{})
	suite.Assert().NoError(err)
	suite.Assert().NotNil(entity)
	suite.Assert().Equal(testEntityID, entity.Entity.GetID())

	// 5. Execute action on entity
	action := types.PMAControlAction{
		EntityID: testEntityID,
		Action:   "turn_on",
		Parameters: map[string]interface{}{
			"brightness": 100,
		},
	}

	result, err := suite.unifiedService.ExecuteAction(ctx, action)
	suite.Assert().NoError(err)
	suite.Assert().NotNil(result)
	suite.Assert().True(result.Success)

	// 6. Verify entity state updated
	entity, err = suite.unifiedService.GetByID(ctx, testEntityID, unified.GetEntityOptions{})
	suite.Assert().NoError(err)
	suite.Assert().Equal(types.StateOn, entity.Entity.GetState())
}

func (suite *PMAIntegrationTestSuite) TestMultiSourceConflictResolution() {
	ctx := context.Background()

	// Register second adapter with same entity
	secondAdapter := NewMockPMAAdapter()
	secondAdapter.id = "mock_adapter_2"
	secondAdapter.sourceType = types.SourceShelly

	err := suite.unifiedService.RegisterAdapter(secondAdapter)
	suite.Assert().NoError(err)

	// Connect both adapters
	err = secondAdapter.Connect(ctx)
	suite.Assert().NoError(err)

	// Sync from both sources
	_, err = suite.unifiedService.SyncFromAllSources(ctx)
	suite.Assert().NoError(err)

	// Verify conflict was resolved based on priority
	conflictEntity, err := suite.unifiedService.GetByID(ctx, "ha_light.conflict_test", unified.GetEntityOptions{})
	suite.Assert().NoError(err)

	// Should prefer HomeAssistant (priority 1) over Shelly (priority 3)
	suite.Assert().Equal(types.SourceHomeAssistant, conflictEntity.Entity.GetSource())
}

func (suite *PMAIntegrationTestSuite) TestEntityRegistryOperations() {
	ctx := context.Background()

	// Connect and sync
	err := suite.testAdapter.Connect(ctx)
	suite.Assert().NoError(err)

	_, err = suite.unifiedService.SyncFromSource(ctx, types.SourceHomeAssistant)
	suite.Assert().NoError(err)

	// Get registry manager
	registryManager := suite.unifiedService.GetRegistryManager()
	suite.Assert().NotNil(registryManager)

	// Get all registry stats
	stats := registryManager.GetAllRegistryStats()
	suite.Assert().NotNil(stats)
	suite.Assert().Contains(stats, "entities")
}

func (suite *PMAIntegrationTestSuite) TestActionValidation() {
	ctx := context.Background()

	// Connect and sync first
	err := suite.testAdapter.Connect(ctx)
	suite.Assert().NoError(err)

	_, err = suite.unifiedService.SyncFromSource(ctx, types.SourceHomeAssistant)
	suite.Assert().NoError(err)

	// Test invalid entity ID
	invalidAction := types.PMAControlAction{
		EntityID: "non_existent_entity",
		Action:   "turn_on",
	}

	result, err := suite.unifiedService.ExecuteAction(ctx, invalidAction)
	suite.Assert().NoError(err) // Should not error, but result should indicate failure
	suite.Assert().False(result.Success)
	suite.Assert().NotNil(result.Error)
	suite.Assert().Equal("ENTITY_NOT_FOUND", result.Error.Code)

	// Test invalid action for entity type
	invalidAction2 := types.PMAControlAction{
		EntityID: "ha_sensor.test_sensor",
		Action:   "turn_on", // Sensors can't be turned on
	}

	result2, err := suite.unifiedService.ExecuteAction(ctx, invalidAction2)
	suite.Assert().NoError(err)
	suite.Assert().False(result2.Success)
	suite.Assert().NotNil(result2.Error)
}

func (suite *PMAIntegrationTestSuite) TestAdapterHealthMonitoring() {
	ctx := context.Background()

	// Connect adapter
	err := suite.testAdapter.Connect(ctx)
	suite.Assert().NoError(err)

	// Check health
	health := suite.testAdapter.GetHealth()
	suite.Assert().NotNil(health)
	suite.Assert().True(health.IsHealthy)
	suite.Assert().Empty(health.Issues)

	// Check metrics
	metrics := suite.testAdapter.GetMetrics()
	suite.Assert().NotNil(metrics)
	suite.Assert().GreaterOrEqual(metrics.EntitiesManaged, 0)
}

func (suite *PMAIntegrationTestSuite) TestEntityCapabilities() {
	ctx := context.Background()

	// Connect and sync
	err := suite.testAdapter.Connect(ctx)
	suite.Assert().NoError(err)

	_, err = suite.unifiedService.SyncFromSource(ctx, types.SourceHomeAssistant)
	suite.Assert().NoError(err)

	// Test light entity capabilities
	lightEntity, err := suite.unifiedService.GetByID(ctx, "ha_light.test_light", unified.GetEntityOptions{})
	suite.Assert().NoError(err)
	suite.Assert().True(lightEntity.Entity.HasCapability(types.CapabilityDimmable))

	// Test available actions
	availableActions := lightEntity.Entity.GetAvailableActions()
	suite.Assert().Contains(availableActions, "turn_on")
	suite.Assert().Contains(availableActions, "turn_off")
}

func (suite *PMAIntegrationTestSuite) TestRoomAndAreaAssignment() {
	ctx := context.Background()

	// Connect and sync
	err := suite.testAdapter.Connect(ctx)
	suite.Assert().NoError(err)

	_, err = suite.unifiedService.SyncFromSource(ctx, types.SourceHomeAssistant)
	suite.Assert().NoError(err)

	// Test entity with room assignment
	entity, err := suite.unifiedService.GetByID(ctx, "ha_light.test_light", unified.GetEntityOptions{})
	suite.Assert().NoError(err)

	if entity.Entity.GetRoomID() != nil {
		suite.Assert().NotEmpty(*entity.Entity.GetRoomID())
	}
}

func (suite *PMAIntegrationTestSuite) TestConcurrentOperations() {
	ctx := context.Background()

	// Connect adapter
	err := suite.testAdapter.Connect(ctx)
	suite.Assert().NoError(err)

	// Sync entities
	_, err = suite.unifiedService.SyncFromSource(ctx, types.SourceHomeAssistant)
	suite.Assert().NoError(err)

	// Execute multiple actions concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(actionNum int) {
			defer func() { done <- true }()

			action := types.PMAControlAction{
				EntityID: "ha_light.test_light",
				Action:   "turn_on",
				Parameters: map[string]interface{}{
					"brightness": 50 + actionNum,
				},
			}

			result, err := suite.unifiedService.ExecuteAction(ctx, action)
			suite.Assert().NoError(err)
			suite.Assert().True(result.Success)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestPMAIntegrationSuite(t *testing.T) {
	suite.Run(t, new(PMAIntegrationTestSuite))
}
