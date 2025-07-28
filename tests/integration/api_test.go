package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type APIIntegrationTestSuite struct {
	suite.Suite
	router         *gin.Engine
	unifiedService *unified.UnifiedEntityService
	testAdapter    *MockPMAAdapter
	logger         *logrus.Logger
}

func (suite *APIIntegrationTestSuite) SetupSuite() {
	// Set gin to test mode
	gin.SetMode(gin.TestMode)

	// Initialize logger
	suite.logger = logrus.New()
	suite.logger.SetLevel(logrus.DebugLevel)

	// Create test config
	config := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Host: "localhost",
			Mode: "test",
		},
		HomeAssistant: config.HomeAssistantConfig{
			URL:   "http://localhost:8123",
			Token: "test-token",
		},
	}

	// Create type registry and unified service
	typeRegistry := types.NewPMATypeRegistry(suite.logger)
	suite.unifiedService = unified.NewUnifiedEntityService(typeRegistry, config, suite.logger)

	// Create and register mock adapter
	suite.testAdapter = NewMockPMAAdapter()
	err := suite.unifiedService.RegisterAdapter(suite.testAdapter)
	suite.Require().NoError(err)

	// Connect adapter and sync entities
	err = suite.testAdapter.Connect(nil)
	suite.Require().NoError(err)

	// Setup router
	suite.router = setupTestRouter(config, suite.unifiedService, suite.logger)
}

func setupTestRouter(config *config.Config, unifiedService *unified.UnifiedEntityService, logger *logrus.Logger) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// Add test auth middleware that accepts any Bearer token
	r.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || authHeader != "Bearer test-token" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	})

	// Setup API routes
	api := r.Group("/api/v1")

	// Mock entity routes that use unified service directly
	api.GET("/entities", func(c *gin.Context) {
		options := unified.GetAllOptions{
			IncludeRoom: c.Query("include_room") == "true",
			IncludeArea: c.Query("include_area") == "true",
		}

		entities, err := unifiedService.GetAll(c.Request.Context(), options)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": entities})
	})

	api.GET("/entities/:id", func(c *gin.Context) {
		entityID := c.Param("id")
		options := unified.GetEntityOptions{
			IncludeRoom: c.Query("include_room") == "true",
			IncludeArea: c.Query("include_area") == "true",
		}

		entity, err := unifiedService.GetByID(c.Request.Context(), entityID, options)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Entity not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": entity})
	})

	api.POST("/entities/:id/action", func(c *gin.Context) {
		entityID := c.Param("id")

		var actionPayload struct {
			Action     string                 `json:"action"`
			Parameters map[string]interface{} `json:"parameters"`
		}

		if err := c.ShouldBindJSON(&actionPayload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		action := types.PMAControlAction{
			EntityID:   entityID,
			Action:     actionPayload.Action,
			Parameters: actionPayload.Parameters,
		}

		result, err := unifiedService.ExecuteAction(c.Request.Context(), action)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
	})

	// Adapter routes
	api.GET("/adapters", func(c *gin.Context) {
		adapters := []map[string]interface{}{
			{
				"id":          "mock_adapter",
				"name":        "Mock Adapter",
				"source_type": "homeassistant",
				"status":      "connected",
				"health": map[string]interface{}{
					"is_healthy": true,
					"issues":     []string{},
				},
			},
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": adapters})
	})

	return r
}

func (suite *APIIntegrationTestSuite) TestAPIEntityRetrieval() {
	// First sync entities
	_, err := suite.unifiedService.SyncFromSource(nil, types.SourceHomeAssistant)
	suite.Require().NoError(err)

	// Test getting all entities
	req := httptest.NewRequest("GET", "/api/v1/entities", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	// Verify response contains PMA entities
	data := response["data"].([]interface{})
	assert.NotEmpty(suite.T(), data)

	// Check first entity is PMA format
	if len(data) > 0 {
		firstEntity := data[0].(map[string]interface{})
		entity := firstEntity["entity"].(map[string]interface{})
		assert.Contains(suite.T(), entity["id"].(string), "ha_") // PMA ID format
		assert.NotNil(suite.T(), entity["metadata"])
	}
}

func (suite *APIIntegrationTestSuite) TestAPISpecificEntity() {
	// First sync entities
	_, err := suite.unifiedService.SyncFromSource(nil, types.SourceHomeAssistant)
	suite.Require().NoError(err)

	// Test getting specific entity
	req := httptest.NewRequest("GET", "/api/v1/entities/ha_light.test_light", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	// Verify entity details
	entityWithRoom := response["data"].(map[string]interface{})
	entity := entityWithRoom["entity"].(map[string]interface{})
	assert.Equal(suite.T(), "ha_light.test_light", entity["id"])
	assert.Equal(suite.T(), "light", entity["type"])
	assert.NotNil(suite.T(), entity["metadata"])
}

func (suite *APIIntegrationTestSuite) TestAPIActionExecution() {
	// First sync entities
	_, err := suite.unifiedService.SyncFromSource(nil, types.SourceHomeAssistant)
	suite.Require().NoError(err)

	// Test executing action
	actionPayload := map[string]interface{}{
		"action": "turn_on",
		"parameters": map[string]interface{}{
			"brightness": 75,
		},
	}

	body, _ := json.Marshal(actionPayload)
	req := httptest.NewRequest("POST", "/api/v1/entities/ha_light.test_light/action", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	// Verify action result
	result := response["data"].(map[string]interface{})
	assert.True(suite.T(), result["success"].(bool))
	assert.Equal(suite.T(), "turn_on", result["action"])
}

func (suite *APIIntegrationTestSuite) TestAPIUnauthorizedAccess() {
	// Test without authorization header
	req := httptest.NewRequest("GET", "/api/v1/entities", nil)
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)

	// Test with invalid token
	req = httptest.NewRequest("GET", "/api/v1/entities", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusUnauthorized, w.Code)
}

func (suite *APIIntegrationTestSuite) TestAPIAdapterStatus() {
	// Test getting adapter status
	req := httptest.NewRequest("GET", "/api/v1/adapters", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), response["success"].(bool))

	// Verify adapter data
	adapters := response["data"].([]interface{})
	assert.NotEmpty(suite.T(), adapters)

	adapter := adapters[0].(map[string]interface{})
	assert.Equal(suite.T(), "mock_adapter", adapter["id"])
	assert.Equal(suite.T(), "connected", adapter["status"])
}

func (suite *APIIntegrationTestSuite) TestAPIErrorHandling() {
	// Test getting non-existent entity
	req := httptest.NewRequest("GET", "/api/v1/entities/non_existent_entity", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)

	// Test executing action on non-existent entity
	actionPayload := map[string]interface{}{
		"action": "turn_on",
	}

	body, _ := json.Marshal(actionPayload)
	req = httptest.NewRequest("POST", "/api/v1/entities/non_existent_entity/action", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	// Should return error but not necessarily 404 - depends on implementation
	assert.NotEqual(suite.T(), http.StatusOK, w.Code)
}

func TestAPIIntegrationSuite(t *testing.T) {
	suite.Run(t, new(APIIntegrationTestSuite))
}
