package handlers

import (
	"time"

	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// Health returns the health status of the service
func (h *Handlers) Health(c *gin.Context) {
	health := gin.H{
		"status":           "healthy",
		"timestamp":        time.Now().Format(time.RFC3339),
		"service":          "pma-backend-go",
		"version":          "1.0.0",
		"connected":        true,
		"backendAvailable": true,
	}

	// Add adapter health if unified service exists
	if h.unifiedService != nil {
		registry := h.unifiedService.GetRegistryManager().GetAdapterRegistry()
		adapters := registry.GetAllAdapters()

		adapterHealth := make(map[string]interface{})
		for _, adapter := range adapters {
			adapterInfo := gin.H{
				"connected": adapter.IsConnected(),
				"type":      string(adapter.GetSourceType()),
				"version":   adapter.GetVersion(),
			}

			// Get adapter health
			if adapterHealthData := adapter.GetHealth(); adapterHealthData != nil {
				adapterInfo["health"] = adapterHealthData
			}

			adapterHealth[adapter.GetID()] = adapterInfo
		}

		health["adapters"] = adapterHealth
		health["adapter_count"] = len(adapters)
	}

	utils.SendSuccess(c, health)
}
