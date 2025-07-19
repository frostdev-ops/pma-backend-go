package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/test"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// TestGenerateMockEntities handles POST /api/v1/test/mock-entities
func (h *Handlers) TestGenerateMockEntities(c *gin.Context) {
	// Check if test endpoints are enabled
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	var req struct {
		Count       int      `json:"count" binding:"required,min=1,max=100"`
		EntityTypes []string `json:"entity_types,omitempty"`
		Reset       bool     `json:"reset,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Initialize test service if needed
	if h.testService == nil {
		h.testService = test.NewService(h.cfg, h.repos, h.log, h.db)
	}

	// Reset existing mock data if requested
	if req.Reset {
		if err := h.testService.ResetTestData(); err != nil {
			utils.SendError(c, http.StatusInternalServerError, "Failed to reset test data: "+err.Error())
			return
		}
	}

	// Generate mock entities
	entities, err := h.testService.GenerateMockEntities(req.Count, req.EntityTypes)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to generate mock entities: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":      "Mock entities generated successfully",
		"count":        len(entities),
		"entities":     entities,
		"generated_at": time.Now(),
		"entity_types": req.EntityTypes,
	})
}

// TestGetMockEntities handles GET /api/v1/test/mock-entities
func (h *Handlers) TestGetMockEntities(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	if h.testService == nil {
		utils.SendSuccess(c, gin.H{
			"message":  "No mock entities available",
			"entities": make(map[string]interface{}),
			"count":    0,
		})
		return
	}

	entities := h.testService.GetMockEntities()

	utils.SendSuccess(c, gin.H{
		"message":      "Mock entities retrieved successfully",
		"entities":     entities,
		"count":        len(entities),
		"retrieved_at": time.Now(),
	})
}

// TestUpdateMockEntity handles PUT /api/v1/test/mock-entities/:id
func (h *Handlers) TestUpdateMockEntity(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	entityID := c.Param("id")
	if entityID == "" {
		utils.SendError(c, http.StatusBadRequest, "Entity ID is required")
		return
	}

	var req struct {
		State      string                 `json:"state"`
		Attributes map[string]interface{} `json:"attributes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if h.testService == nil {
		utils.SendError(c, http.StatusNotFound, "Test service not initialized")
		return
	}

	err := h.testService.UpdateMockEntity(entityID, req.State, req.Attributes)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "Failed to update mock entity: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":    "Mock entity updated successfully",
		"entity_id":  entityID,
		"state":      req.State,
		"updated_at": time.Now(),
	})
}

// TestDeleteMockEntity handles DELETE /api/v1/test/mock-entities/:id
func (h *Handlers) TestDeleteMockEntity(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	entityID := c.Param("id")
	if entityID == "" {
		utils.SendError(c, http.StatusBadRequest, "Entity ID is required")
		return
	}

	if h.testService == nil {
		utils.SendError(c, http.StatusNotFound, "Test service not initialized")
		return
	}

	err := h.testService.DeleteMockEntity(entityID)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "Failed to delete mock entity: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":    "Mock entity deleted successfully",
		"entity_id":  entityID,
		"deleted_at": time.Now(),
	})
}

// TestConnections handles POST /api/v1/test/connections
func (h *Handlers) TestConnections(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Initialize test service if needed
	if h.testService == nil {
		h.testService = test.NewService(h.cfg, h.repos, h.log, h.db)
	}

	results, err := h.testService.TestConnections(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to test connections: "+err.Error())
		return
	}

	// Count status types
	summary := map[string]int{
		"healthy":        0,
		"unhealthy":      0,
		"disabled":       0,
		"not_configured": 0,
	}

	for _, result := range results {
		summary[result.Status]++
	}

	utils.SendSuccess(c, gin.H{
		"message":     "Connection tests completed",
		"results":     results,
		"summary":     summary,
		"tested_at":   time.Now(),
		"total_tests": len(results),
	})
}

// TestRouterAPI handles POST /api/v1/test/router
func (h *Handlers) TestRouterAPI(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if h.testService == nil {
		h.testService = test.NewService(h.cfg, h.repos, h.log, h.db)
	}

	// Test PMA Router specifically
	results, err := h.testService.TestConnections(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to test router: "+err.Error())
		return
	}

	routerResult, exists := results["pma_router"]
	if !exists {
		utils.SendError(c, http.StatusInternalServerError, "Router test result not found")
		return
	}

	// Also test network interfaces
	networkResult := results["network"]

	utils.SendSuccess(c, gin.H{
		"message":        "Router connectivity test completed",
		"router_status":  routerResult,
		"network_status": networkResult,
		"tested_at":      time.Now(),
	})
}

// TestSystemHealth handles GET /api/v1/test/system-health
func (h *Handlers) TestSystemHealth(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if h.testService == nil {
		h.testService = test.NewService(h.cfg, h.repos, h.log, h.db)
	}

	results, err := h.testService.GetSystemHealth(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to check system health: "+err.Error())
		return
	}

	// Calculate overall health status
	overallStatus := "healthy"
	warningCount := 0
	errorCount := 0

	for _, result := range results {
		switch result.Status {
		case "warning":
			warningCount++
			if overallStatus == "healthy" {
				overallStatus = "warning"
			}
		case "unhealthy":
			errorCount++
			overallStatus = "unhealthy"
		}
	}

	utils.SendSuccess(c, gin.H{
		"message":        "System health check completed",
		"overall_status": overallStatus,
		"results":        results,
		"summary": gin.H{
			"warnings": warningCount,
			"errors":   errorCount,
			"checks":   len(results),
		},
		"checked_at": time.Now(),
	})
}

// TestResetData handles POST /api/v1/test/reset
func (h *Handlers) TestResetData(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	var req struct {
		ConfirmReset bool     `json:"confirm_reset" binding:"required"`
		ResetTypes   []string `json:"reset_types,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if !req.ConfirmReset {
		utils.SendError(c, http.StatusBadRequest, "Reset confirmation required")
		return
	}

	if h.testService == nil {
		h.testService = test.NewService(h.cfg, h.repos, h.log, h.db)
	}

	// Default reset types
	if len(req.ResetTypes) == 0 {
		req.ResetTypes = []string{"mock_entities", "test_data"}
	}

	resetActions := make([]string, 0)

	for _, resetType := range req.ResetTypes {
		switch resetType {
		case "mock_entities", "test_data":
			if err := h.testService.ResetTestData(); err != nil {
				utils.SendError(c, http.StatusInternalServerError, "Failed to reset test data: "+err.Error())
				return
			}
			resetActions = append(resetActions, "mock entities and test data")

		default:
			h.log.Warnf("Unknown reset type: %s", resetType)
		}
	}

	utils.SendSuccess(c, gin.H{
		"message":     "Test data reset completed",
		"reset_types": req.ResetTypes,
		"actions":     resetActions,
		"reset_at":    time.Now(),
	})
}

// TestGenerateData handles POST /api/v1/test/generate-data
func (h *Handlers) TestGenerateData(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	var req struct {
		DataType string                 `json:"data_type" binding:"required"`
		Count    int                    `json:"count,omitempty"`
		Options  map[string]interface{} `json:"options,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	if h.testService == nil {
		h.testService = test.NewService(h.cfg, h.repos, h.log, h.db)
	}

	switch req.DataType {
	case "entities":
		count := req.Count
		if count == 0 {
			count = 10
		}
		if count > 50 {
			count = 50
		}

		var entityTypes []string
		if types, ok := req.Options["entity_types"].([]interface{}); ok {
			for _, t := range types {
				if typeStr, ok := t.(string); ok {
					entityTypes = append(entityTypes, typeStr)
				}
			}
		}

		entities, err := h.testService.GenerateMockEntities(count, entityTypes)
		if err != nil {
			utils.SendError(c, http.StatusInternalServerError, "Failed to generate entities: "+err.Error())
			return
		}

		utils.SendSuccess(c, gin.H{
			"message":      "Test entities generated",
			"data_type":    req.DataType,
			"count":        len(entities),
			"entities":     entities,
			"generated_at": time.Now(),
		})

	default:
		utils.SendError(c, http.StatusBadRequest, fmt.Sprintf("Unsupported data type: %s", req.DataType))
	}
}

// TestConfig handles GET /api/v1/test/config
func (h *Handlers) TestConfig(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	if h.testService == nil {
		h.testService = test.NewService(h.cfg, h.repos, h.log, h.db)
	}

	config := h.testService.GetTestConfig()

	utils.SendSuccess(c, gin.H{
		"message":      "Test configuration retrieved",
		"config":       config,
		"server_mode":  h.cfg.Server.Mode,
		"retrieved_at": time.Now(),
	})
}

// TestIntegrations handles POST /api/v1/test/integrations
func (h *Handlers) TestIntegrations(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	var req struct {
		Integrations []string `json:"integrations,omitempty"`
		Timeout      int      `json:"timeout,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	timeout := time.Duration(req.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if h.testService == nil {
		h.testService = test.NewService(h.cfg, h.repos, h.log, h.db)
	}

	// Test all connections and filter by requested integrations
	allResults, err := h.testService.TestConnections(ctx)
	if err != nil {
		utils.SendError(c, http.StatusInternalServerError, "Failed to test integrations: "+err.Error())
		return
	}

	results := make(map[string]*test.ConnectionTestResult)

	if len(req.Integrations) == 0 {
		// Test all integrations
		results = allResults
	} else {
		// Test only requested integrations
		for _, integration := range req.Integrations {
			if result, exists := allResults[integration]; exists {
				results[integration] = result
			}
		}
	}

	utils.SendSuccess(c, gin.H{
		"message":     "Integration tests completed",
		"results":     results,
		"requested":   req.Integrations,
		"tested_at":   time.Now(),
		"timeout_sec": int(timeout.Seconds()),
	})
}

// TestPerformance handles POST /api/v1/test/performance
func (h *Handlers) TestPerformance(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	var req struct {
		TestType   string `json:"test_type" binding:"required"`
		Duration   int    `json:"duration,omitempty"`
		Iterations int    `json:"iterations,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Basic performance test placeholder
	start := time.Now()

	var results map[string]interface{}

	switch req.TestType {
	case "database":
		// Simple database performance test
		ctx := context.Background()
		var count int
		queryStart := time.Now()
		err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM entities").Scan(&count)
		queryTime := time.Since(queryStart)

		if err != nil {
			utils.SendError(c, http.StatusInternalServerError, "Database performance test failed: "+err.Error())
			return
		}

		results = map[string]interface{}{
			"test_type":      "database",
			"query_time_ms":  queryTime.Milliseconds(),
			"entity_count":   count,
			"queries_tested": 1,
		}

	default:
		utils.SendError(c, http.StatusBadRequest, fmt.Sprintf("Unsupported performance test type: %s", req.TestType))
		return
	}

	totalTime := time.Since(start)

	utils.SendSuccess(c, gin.H{
		"message":       "Performance test completed",
		"test_type":     req.TestType,
		"results":       results,
		"total_time_ms": totalTime.Milliseconds(),
		"tested_at":     time.Now(),
	})
}

// TestWebSocket handles GET /api/v1/test/websocket
func (h *Handlers) TestWebSocket(c *gin.Context) {
	if !h.isTestEndpointsEnabled() {
		utils.SendError(c, http.StatusForbidden, "Test endpoints are disabled in production mode")
		return
	}

	// Basic WebSocket endpoint test
	wsStats := make(map[string]interface{})

	if h.wsHub != nil {
		// Get WebSocket hub statistics if available
		wsStats["hub_initialized"] = true
		wsStats["endpoint_available"] = true
	} else {
		wsStats["hub_initialized"] = false
		wsStats["endpoint_available"] = false
	}

	utils.SendSuccess(c, gin.H{
		"message":    "WebSocket test completed",
		"status":     "available",
		"endpoint":   "/ws",
		"statistics": wsStats,
		"tested_at":  time.Now(),
	})
}

// isTestEndpointsEnabled checks if test endpoints should be available
func (h *Handlers) isTestEndpointsEnabled() bool {
	// Only enable test endpoints in development mode
	return h.cfg.Server.Mode == "development"
}

// TestEndpoint handles GET /api/v1/test/endpoint-status
func (h *Handlers) TestEndpointStatus(c *gin.Context) {
	status := "disabled"
	if h.isTestEndpointsEnabled() {
		status = "enabled"
	}

	utils.SendSuccess(c, gin.H{
		"message":           "Test endpoint status",
		"status":            status,
		"server_mode":       h.cfg.Server.Mode,
		"endpoints_enabled": h.isTestEndpointsEnabled(),
		"checked_at":        time.Now(),
	})
}
