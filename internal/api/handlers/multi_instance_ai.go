package handlers

import (
	"fmt"
	"net/http"

	"github.com/frostdev-ops/pma-backend-go/internal/ai/providers"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GetMultiInstanceStatus returns the status of the multi-instance model manager
func (h *Handlers) GetMultiInstanceStatus(c *gin.Context) {
	// Check if we're using the multi-instance provider
	multiProvider, ok := h.llmManager.GetProvider("multi-llamacpp")
	if !ok {
		utils.SendError(c, http.StatusNotFound, "Multi-instance provider not available")
		return
	}

	// Cast to multi-instance provider
	provider, ok := multiProvider.(*providers.MultiInstanceLlamaCppProvider)
	if !ok {
		utils.SendError(c, http.StatusInternalServerError, "Provider is not multi-instance type")
		return
	}

	// Get manager status
	status := provider.GetManagerStatus()
	instanceStats := provider.GetInstanceStats()

	response := gin.H{
		"status":         "healthy",
		"manager_status": status,
		"instance_stats": instanceStats,
		"provider_stats": provider.GetStats(),
	}

	h.log.WithFields(logrus.Fields{
		"total_instances":   status["total_instances"],
		"running_instances": status["running_instances"],
	}).Info("📊 Multi-instance status requested")

	utils.SendSuccess(c, response)
}

// ConfigureMultiInstanceModels configures which models should be enabled/disabled
func (h *Handlers) ConfigureMultiInstanceModels(c *gin.Context) {
	var req struct {
		EnabledModels map[string]bool `json:"enabled_models" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.WithError(err).Error("❌ Failed to bind multi-instance configuration request")
		utils.SendError(c, http.StatusBadRequest, "Invalid request format")
		return
	}

	// Check if we're using the multi-instance provider
	multiProvider, ok := h.llmManager.GetProvider("multi-llamacpp")
	if !ok {
		utils.SendError(c, http.StatusNotFound, "Multi-instance provider not available")
		return
	}

	// Cast to multi-instance provider
	provider, ok := multiProvider.(*providers.MultiInstanceLlamaCppProvider)
	if !ok {
		utils.SendError(c, http.StatusInternalServerError, "Provider is not multi-instance type")
		return
	}

	// Configure models
	if err := provider.ConfigureModels(req.EnabledModels); err != nil {
		h.log.WithError(err).Error("❌ Failed to configure multi-instance models")
		utils.SendError(c, http.StatusInternalServerError, "Failed to configure models")
		return
	}

	h.log.WithFields(logrus.Fields{
		"enabled_models": req.EnabledModels,
	}).Info("🔧 Multi-instance model configuration updated")

	utils.SendSuccess(c, gin.H{
		"message":        "Model configuration updated successfully",
		"enabled_models": req.EnabledModels,
	})
}

// GetMultiInstanceRecommendations provides recommendations for model configuration
func (h *Handlers) GetMultiInstanceRecommendations(c *gin.Context) {
	// Check if we're using the multi-instance provider
	multiProvider, ok := h.llmManager.GetProvider("multi-llamacpp")
	if !ok {
		utils.SendError(c, http.StatusNotFound, "Multi-instance provider not available")
		return
	}

	// Cast to multi-instance provider
	provider, ok := multiProvider.(*providers.MultiInstanceLlamaCppProvider)
	if !ok {
		utils.SendError(c, http.StatusInternalServerError, "Provider is not multi-instance type")
		return
	}

	// Get current status for recommendations
	status := provider.GetManagerStatus()

	// Generate recommendations based on resource usage
	var recommendations []string

	if resourceUsage, ok := status["resource_usage"].(map[string]interface{}); ok {
		if memoryPercent, ok := resourceUsage["memory_percent"].(int64); ok {
			if memoryPercent > 85 {
				recommendations = append(recommendations, "High memory usage detected - consider disabling F16 model")
			}
			if memoryPercent < 50 {
				recommendations = append(recommendations, "Low memory usage - consider enabling more model quantizations")
			}
		}
	}

	if totalInstances, ok := status["total_instances"].(int); ok {
		if runningInstances, ok := status["running_instances"].(int); ok {
			if runningInstances < totalInstances {
				recommendations = append(recommendations, "Some instances are not running - check logs for issues")
			}
			if runningInstances == 0 {
				recommendations = append(recommendations, "No instances running - AI functionality may be unavailable")
			}
		}
	}

	// Default recommendations
	if len(recommendations) == 0 {
		recommendations = append(recommendations,
			"System is running optimally",
			"Consider enabling Q8 model for high-precision tasks if memory allows",
			"Monitor instance performance for optimal load balancing",
		)
	}

	response := gin.H{
		"recommendations": recommendations,
		"current_status":  status,
		"optimal_config": gin.H{
			"LFM2-1.2B-Q2":     true,  // Always enable for speed
			"LFM2-1.2B-Q4_K_M": true,  // Always enable for balance
			"LFM2-1.2B-Q8":     true,  // Enable for precision
			"LFM2-1.2B":        false, // Disable F16 by default due to memory
		},
	}

	h.log.WithField("recommendation_count", len(recommendations)).Info("💡 Multi-instance recommendations generated")

	utils.SendSuccess(c, response)
}

// RestartMultiInstanceModel restarts a specific model instance
func (h *Handlers) RestartMultiInstanceModel(c *gin.Context) {
	modelName := c.Param("model")
	if modelName == "" {
		utils.SendError(c, http.StatusBadRequest, "Model name is required")
		return
	}

	// Check if we're using the multi-instance provider
	multiProvider, ok := h.llmManager.GetProvider("multi-llamacpp")
	if !ok {
		utils.SendError(c, http.StatusNotFound, "Multi-instance provider not available")
		return
	}

	// Cast to multi-instance provider
	provider, ok := multiProvider.(*providers.MultiInstanceLlamaCppProvider)
	if !ok {
		utils.SendError(c, http.StatusInternalServerError, "Provider is not multi-instance type")
		return
	}

	// Restart the model instance
	ctx := c.Request.Context()
	if err := provider.RestartModel(ctx, modelName); err != nil {
		h.log.WithError(err).WithField("model", modelName).Error("❌ Failed to restart model instance")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to restart model: %v", err))
		return
	}

	h.log.WithField("model", modelName).Info("🔄 Model instance restart completed")

	utils.SendSuccess(c, gin.H{
		"message": "Model restart completed successfully",
		"model":   modelName,
		"status":  "restarted",
	})
}

// GetMultiInstanceHealth performs a comprehensive health check of all instances
func (h *Handlers) GetMultiInstanceHealth(c *gin.Context) {
	// Check if we're using the multi-instance provider
	multiProvider, ok := h.llmManager.GetProvider("multi-llamacpp")
	if !ok {
		utils.SendError(c, http.StatusNotFound, "Multi-instance provider not available")
		return
	}

	// Cast to multi-instance provider
	provider, ok := multiProvider.(*providers.MultiInstanceLlamaCppProvider)
	if !ok {
		utils.SendError(c, http.StatusInternalServerError, "Provider is not multi-instance type")
		return
	}

	// Perform health check
	ctx := c.Request.Context()
	healthErr := provider.HealthCheck(ctx)

	status := provider.GetManagerStatus()
	instanceStats := provider.GetInstanceStats()

	// Determine overall health
	overallHealth := "healthy"
	if healthErr != nil {
		overallHealth = "unhealthy"
	}

	// Count healthy instances
	healthyCount := 0
	totalCount := 0

	if totalInstances, ok := status["total_instances"].(int); ok {
		totalCount = totalInstances
	}

	if runningInstances, ok := status["running_instances"].(int); ok {
		healthyCount = runningInstances
	}

	if healthyCount == 0 {
		overallHealth = "critical"
	} else if healthyCount < totalCount {
		overallHealth = "degraded"
	}

	response := gin.H{
		"overall_health":     overallHealth,
		"healthy_instances":  healthyCount,
		"total_instances":    totalCount,
		"manager_status":     status,
		"instance_stats":     instanceStats,
		"health_check_error": nil,
	}

	if healthErr != nil {
		response["health_check_error"] = healthErr.Error()
	}

	h.log.WithFields(logrus.Fields{
		"overall_health":    overallHealth,
		"healthy_instances": healthyCount,
		"total_instances":   totalCount,
	}).Info("🏥 Multi-instance health check completed")

	utils.SendSuccess(c, response)
}
