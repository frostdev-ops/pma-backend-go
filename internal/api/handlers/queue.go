package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/gin-gonic/gin"
)

// GetQueueStatus returns queue status and statistics
func (h *Handlers) GetQueueStatus(c *gin.Context) {
	stats, err := h.queueService.GetStatistics(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get queue statistics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue statistics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "ok",
		"statistics": stats,
	})
}

// GetQueueActions lists queued actions with filtering
func (h *Handlers) GetQueueActions(c *gin.Context) {
	filter := &models.QueueFilter{}

	// Parse query parameters
	if statuses, exists := c.GetQueryArray("status"); exists && len(statuses) > 0 {
		filter.Status = statuses
	}

	if priorities, exists := c.GetQueryArray("priority"); exists && len(priorities) > 0 {
		filter.Priority = priorities
	}

	if actionTypes, exists := c.GetQueryArray("action_type"); exists && len(actionTypes) > 0 {
		filter.ActionType = actionTypes
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.Atoi(userIDStr); err == nil {
			filter.UserID = &userID
		}
	}

	if correlationID := c.Query("correlation_id"); correlationID != "" {
		filter.CorrelationID = correlationID
	}

	if targetEntityID := c.Query("target_entity_id"); targetEntityID != "" {
		filter.TargetEntityID = targetEntityID
	}

	if createdAfterStr := c.Query("created_after"); createdAfterStr != "" {
		if createdAfter, err := time.Parse(time.RFC3339, createdAfterStr); err == nil {
			filter.CreatedAfter = &createdAfter
		}
	}

	if createdBeforeStr := c.Query("created_before"); createdBeforeStr != "" {
		if createdBefore, err := time.Parse(time.RFC3339, createdBeforeStr); err == nil {
			filter.CreatedBefore = &createdBefore
		}
	}

	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	} else {
		filter.Limit = 50 // Default limit
	}

	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	if orderBy := c.Query("order_by"); orderBy != "" {
		filter.OrderBy = orderBy
	}

	if orderDirection := c.Query("order_direction"); orderDirection != "" {
		filter.OrderDirection = orderDirection
	}

	actions, err := h.queueService.GetActions(c.Request.Context(), filter)
	if err != nil {
		h.log.WithError(err).Error("Failed to get queue actions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue actions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"actions": actions,
		"filter":  filter,
	})
}

// CreateQueueAction creates a new queued action
func (h *Handlers) CreateQueueAction(c *gin.Context) {
	var req models.CreateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get user ID from context if available
	var userID *int
	if userIDInterface, exists := c.Get("userID"); exists {
		if uid, ok := userIDInterface.(int); ok {
			userID = &uid
		}
	}

	action, err := h.queueService.EnqueueAction(c.Request.Context(), &req, userID)
	if err != nil {
		h.log.WithError(err).Error("Failed to create queue action")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Action queued successfully",
		"action":  action,
	})
}

// GetQueueAction gets a specific queued action
func (h *Handlers) GetQueueAction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action ID"})
		return
	}

	action, err := h.queueService.GetAction(c.Request.Context(), id)
	if err != nil {
		h.log.WithError(err).WithField("action_id", id).Error("Failed to get queue action")
		c.JSON(http.StatusNotFound, gin.H{"error": "Action not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"action": action,
	})
}

// UpdateQueueAction updates a queued action
func (h *Handlers) UpdateQueueAction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action ID"})
		return
	}

	var req models.UpdateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	action, err := h.queueService.UpdateAction(c.Request.Context(), id, &req)
	if err != nil {
		h.log.WithError(err).WithField("action_id", id).Error("Failed to update queue action")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Action updated successfully",
		"action":  action,
	})
}

// DeleteQueueAction deletes a queued action
func (h *Handlers) DeleteQueueAction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action ID"})
		return
	}

	err = h.queueService.DeleteAction(c.Request.Context(), id)
	if err != nil {
		h.log.WithError(err).WithField("action_id", id).Error("Failed to delete queue action")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Action deleted successfully",
	})
}

// CancelQueueAction cancels a queued action
func (h *Handlers) CancelQueueAction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action ID"})
		return
	}

	err = h.queueService.CancelAction(c.Request.Context(), id)
	if err != nil {
		h.log.WithError(err).WithField("action_id", id).Error("Failed to cancel queue action")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Action cancelled successfully",
	})
}

// ProcessQueue manually triggers queue processing
func (h *Handlers) ProcessQueue(c *gin.Context) {
	var req models.QueueProcessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body for simple processing
		req = models.QueueProcessRequest{}
	}

	processedCount, err := h.queueService.ProcessQueue(c.Request.Context(), &req)
	if err != nil {
		h.log.WithError(err).Error("Failed to process queue")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Queue processing completed",
		"processed_count": processedCount,
	})
}

// ClearQueue clears queue items based on criteria
func (h *Handlers) ClearQueue(c *gin.Context) {
	var req models.QueueClearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	deletedCount, err := h.queueService.ClearQueue(c.Request.Context(), &req)
	if err != nil {
		h.log.WithError(err).Error("Failed to clear queue")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Queue cleared successfully",
		"deleted_count": deletedCount,
	})
}

// CreateBulkActions creates multiple actions at once
func (h *Handlers) CreateBulkActions(c *gin.Context) {
	var req models.BulkActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Get user ID from context if available
	var userID *int
	if userIDInterface, exists := c.Get("userID"); exists {
		if uid, ok := userIDInterface.(int); ok {
			userID = &uid
		}
	}

	actions, err := h.queueService.EnqueueBulkActions(c.Request.Context(), &req, userID)
	if err != nil {
		h.log.WithError(err).Error("Failed to create bulk actions")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Bulk actions queued successfully",
		"actions":       actions,
		"created_count": len(actions),
	})
}

// GetQueueStatistics returns detailed queue statistics
func (h *Handlers) GetQueueStatistics(c *gin.Context) {
	stats, err := h.queueService.GetStatistics(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get queue statistics")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetQueueHealth returns queue system health check
func (h *Handlers) GetQueueHealth(c *gin.Context) {
	stats, err := h.queueService.GetStatistics(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get queue health")
		c.JSON(http.StatusInternalServerError, gin.H{
			"status": "error",
			"error":  "Failed to get queue health",
		})
		return
	}

	// Determine overall health status
	var status string
	var healthy bool

	switch stats.QueueHealth {
	case "healthy":
		status = "healthy"
		healthy = true
	case "active":
		status = "active"
		healthy = true
	case "warning":
		status = "warning"
		healthy = false
	case "critical":
		status = "critical"
		healthy = false
	default:
		status = "unknown"
		healthy = false
	}

	response := gin.H{
		"status":             status,
		"healthy":            healthy,
		"queue_health":       stats.QueueHealth,
		"total_actions":      stats.TotalActions,
		"pending_actions":    stats.PendingActions,
		"processing_actions": stats.ProcessingActions,
		"failed_actions":     stats.FailedActions,
		"success_rate":       stats.SuccessRate,
		"avg_execution_time": stats.AvgExecutionTime,
		"worker_status":      stats.WorkerStatus,
	}

	if healthy {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusServiceUnavailable, response)
	}
}

// GetQueueSettings returns queue configuration settings
func (h *Handlers) GetQueueSettings(c *gin.Context) {
	settings, err := h.queueService.GetSettings(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get queue settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queue settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"settings": settings,
	})
}

// UpdateQueueSettings updates queue configuration
func (h *Handlers) UpdateQueueSettings(c *gin.Context) {
	var req struct {
		Settings map[string]string `json:"settings" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	updatedCount := 0
	errors := make(map[string]string)

	for key, value := range req.Settings {
		if err := h.queueService.UpdateSetting(c.Request.Context(), key, value); err != nil {
			h.log.WithError(err).WithFields(map[string]interface{}{
				"key":   key,
				"value": value,
			}).Error("Failed to update queue setting")
			errors[key] = err.Error()
		} else {
			updatedCount++
		}
	}

	response := gin.H{
		"message":       "Settings update completed",
		"updated_count": updatedCount,
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	if len(errors) > 0 && updatedCount == 0 {
		c.JSON(http.StatusBadRequest, response)
	} else {
		c.JSON(http.StatusOK, response)
	}
}

// GetActionTypes returns available action types
func (h *Handlers) GetActionTypes(c *gin.Context) {
	actionTypes, err := h.queueService.GetActionTypes(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to get action types")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get action types"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"action_types": actionTypes,
	})
}

// CleanupOldActions removes old completed/failed actions
func (h *Handlers) CleanupOldActions(c *gin.Context) {
	deletedCount, err := h.queueService.CleanupOldActions(c.Request.Context())
	if err != nil {
		h.log.WithError(err).Error("Failed to cleanup old actions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup old actions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Cleanup completed",
		"deleted_count": deletedCount,
	})
}

// RetryFailedAction retries a specific failed action
func (h *Handlers) RetryFailedAction(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action ID"})
		return
	}

	// Create a process request for this specific action with force retry
	req := &models.QueueProcessRequest{
		ActionIDs:  []int{id},
		ForceRetry: true,
	}

	processedCount, err := h.queueService.ProcessQueue(c.Request.Context(), req)
	if err != nil {
		h.log.WithError(err).WithField("action_id", id).Error("Failed to retry action")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message := "Action retry completed"
	if processedCount == 0 {
		message = "Action could not be retried (may not be in failed state)"
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   message,
		"processed": processedCount > 0,
	})
}
