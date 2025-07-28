package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// GetAdapters returns information about all registered adapters
func (h *Handlers) GetAdapters(c *gin.Context) {
	if h.unifiedService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Unified service not available")
		return
	}

	registry := h.unifiedService.GetRegistryManager().GetAdapterRegistry()
	adapters := registry.GetAllAdapters()

	adapterList := make([]gin.H, 0, len(adapters))
	for _, adapter := range adapters {
		adapterInfo := gin.H{
			"id":        adapter.GetID(),
			"type":      string(adapter.GetSourceType()),
			"version":   adapter.GetVersion(),
			"connected": adapter.IsConnected(),
		}

		// Get adapter health if available
		if health := adapter.GetHealth(); health != nil {
			adapterInfo["health"] = health
		}

		adapterList = append(adapterList, adapterInfo)
	}

	utils.SendSuccess(c, gin.H{
		"adapters": adapterList,
		"count":    len(adapterList),
	})
}

// GetAdapterHealth returns detailed health information for a specific adapter
func (h *Handlers) GetAdapterHealth(c *gin.Context) {
	if h.unifiedService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Unified service not available")
		return
	}

	adapterID := c.Param("id")
	registry := h.unifiedService.GetRegistryManager().GetAdapterRegistry()

	adapter, err := registry.GetAdapter(adapterID)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "Adapter not found: "+err.Error())
		return
	}

	healthData := gin.H{
		"id":        adapter.GetID(),
		"type":      string(adapter.GetSourceType()),
		"version":   adapter.GetVersion(),
		"connected": adapter.IsConnected(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Get detailed health if available
	if health := adapter.GetHealth(); health != nil {
		healthData["health"] = health
	}

	utils.SendSuccess(c, healthData)
}

// ConnectAdapter attempts to connect a specific adapter
func (h *Handlers) ConnectAdapter(c *gin.Context) {
	if h.unifiedService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Unified service not available")
		return
	}

	adapterID := c.Param("id")
	registry := h.unifiedService.GetRegistryManager().GetAdapterRegistry()

	adapter, err := registry.GetAdapter(adapterID)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "Adapter not found: "+err.Error())
		return
	}

	ctx := context.Background()
	if err := adapter.Connect(ctx); err != nil {
		h.log.WithError(err).Errorf("Failed to connect adapter: %s", adapterID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to connect adapter: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Adapter connected successfully",
		"adapter":   adapterID,
		"connected": adapter.IsConnected(),
	})
}

// DisconnectAdapter attempts to disconnect a specific adapter
func (h *Handlers) DisconnectAdapter(c *gin.Context) {
	if h.unifiedService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Unified service not available")
		return
	}

	adapterID := c.Param("id")
	registry := h.unifiedService.GetRegistryManager().GetAdapterRegistry()

	adapter, err := registry.GetAdapter(adapterID)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "Adapter not found: "+err.Error())
		return
	}

	ctx := context.Background()
	if err := adapter.Disconnect(ctx); err != nil {
		h.log.WithError(err).Errorf("Failed to disconnect adapter: %s", adapterID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to disconnect adapter: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Adapter disconnected successfully",
		"adapter":   adapterID,
		"connected": adapter.IsConnected(),
	})
}

// GetAdapterMetrics returns performance and usage metrics for a specific adapter
func (h *Handlers) GetAdapterMetrics(c *gin.Context) {
	if h.unifiedService == nil {
		utils.SendError(c, http.StatusServiceUnavailable, "Unified service not available")
		return
	}

	adapterID := c.Param("id")
	registry := h.unifiedService.GetRegistryManager().GetAdapterRegistry()

	adapter, err := registry.GetAdapter(adapterID)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, "Adapter not found: "+err.Error())
		return
	}

	metrics := gin.H{
		"id":        adapter.GetID(),
		"type":      string(adapter.GetSourceType()),
		"version":   adapter.GetVersion(),
		"connected": adapter.IsConnected(),
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Get basic metrics from the adapter if available
	if health := adapter.GetHealth(); health != nil {
		metrics["health"] = health
	}

	// Get detailed metrics from the adapter
	if adapterMetrics := adapter.GetMetrics(); adapterMetrics != nil {
		metrics["detailed_metrics"] = adapterMetrics

		// Extract key metrics for easier access
		metrics["entities_managed"] = adapterMetrics.EntitiesManaged
		metrics["rooms_managed"] = adapterMetrics.RoomsManaged
		metrics["actions_executed"] = adapterMetrics.ActionsExecuted
		metrics["successful_actions"] = adapterMetrics.SuccessfulActions
		metrics["failed_actions"] = adapterMetrics.FailedActions
		metrics["average_response_time"] = adapterMetrics.AverageResponseTime.String()
		metrics["sync_errors"] = adapterMetrics.SyncErrors
		metrics["uptime"] = adapterMetrics.Uptime.String()

		if adapterMetrics.LastSync != nil {
			metrics["last_sync"] = adapterMetrics.LastSync.Format(time.RFC3339)
		}

		// Calculate additional derived metrics
		totalActions := adapterMetrics.ActionsExecuted
		if totalActions > 0 {
			metrics["success_rate"] = float64(adapterMetrics.SuccessfulActions) / float64(totalActions)
			metrics["failure_rate"] = float64(adapterMetrics.FailedActions) / float64(totalActions)
		} else {
			metrics["success_rate"] = 0.0
			metrics["failure_rate"] = 0.0
		}
	}

	// Get supported capabilities
	supportedEntityTypes := adapter.GetSupportedEntityTypes()
	supportedCapabilities := adapter.GetSupportedCapabilities()

	metrics["supported_entity_types"] = supportedEntityTypes
	metrics["supported_capabilities"] = supportedCapabilities
	metrics["entity_type_count"] = len(supportedEntityTypes)
	metrics["capability_count"] = len(supportedCapabilities)
	metrics["supports_realtime"] = adapter.SupportsRealtime()

	// Get sync information if available
	if lastSyncTime := adapter.GetLastSyncTime(); lastSyncTime != nil {
		metrics["last_sync_time"] = lastSyncTime.Format(time.RFC3339)
		metrics["time_since_last_sync"] = time.Since(*lastSyncTime).String()
	}

	// Add adapter identification details
	metrics["adapter_details"] = map[string]interface{}{
		"id":          adapter.GetID(),
		"name":        adapter.GetName(),
		"version":     adapter.GetVersion(),
		"source_type": adapter.GetSourceType(),
		"status":      adapter.GetStatus(),
		"connected":   adapter.IsConnected(),
	}

	utils.SendSuccess(c, metrics)
}
