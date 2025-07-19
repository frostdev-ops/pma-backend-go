package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// GetEntities retrieves all entities using the unified PMA service
func (h *Handlers) GetEntities(c *gin.Context) {
	includeRoom := c.Query("include_room") == "true"
	includeArea := c.Query("include_area") == "true"
	domain := c.Query("domain")
	availableOnly := c.Query("available_only") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Prepare options for unified service
	options := unified.GetAllOptions{
		Domain:        domain,
		IncludeRoom:   includeRoom,
		IncludeArea:   includeArea,
		AvailableOnly: availableOnly,
	}

	// Parse capabilities filter if provided
	if capsQuery := c.Query("capabilities"); capsQuery != "" {
		var capabilities []types.PMACapability
		if err := json.Unmarshal([]byte(capsQuery), &capabilities); err == nil {
			options.Capabilities = capabilities
		}
	}

	// Get entities from unified service
	entitiesWithRooms, err := h.unifiedService.GetAll(ctx, options)
	if err != nil {
		h.log.WithError(err).Error("Failed to get all entities from unified service")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve entities")
		return
	}

	// Prepare metadata for response
	meta := gin.H{
		"count":          len(entitiesWithRooms),
		"include_room":   includeRoom,
		"include_area":   includeArea,
		"available_only": availableOnly,
	}

	if domain != "" {
		meta["domain"] = domain
	}

	// Count by source for debugging
	sourceCounts := make(map[string]int)
	for _, entityWithRoom := range entitiesWithRooms {
		source := string(entityWithRoom.Entity.GetSource())
		sourceCounts[source]++
	}
	meta["by_source"] = sourceCounts

	utils.SendSuccessWithMeta(c, entitiesWithRooms, meta)
}

// GetEntity retrieves a specific entity using the unified PMA service
func (h *Handlers) GetEntity(c *gin.Context) {
	entityID := c.Param("id")
	includeRoom := c.Query("include_room") == "true"
	includeArea := c.Query("include_area") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	options := unified.GetEntityOptions{
		IncludeRoom: includeRoom,
		IncludeArea: includeArea,
	}

	entityWithRoom, err := h.unifiedService.GetByID(ctx, entityID, options)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to get entity: %s", entityID)
		utils.SendError(c, http.StatusNotFound, "Entity not found")
		return
	}

	utils.SendSuccess(c, entityWithRoom)
}

// GetEntitiesByType retrieves entities by type using the unified PMA service
func (h *Handlers) GetEntitiesByType(c *gin.Context) {
	entityTypeStr := c.Param("type")
	includeRoom := c.Query("include_room") == "true"
	includeArea := c.Query("include_area") == "true"
	availableOnly := c.Query("available_only") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert string to PMA entity type
	entityType := types.PMAEntityType(entityTypeStr)

	options := unified.GetAllOptions{
		IncludeRoom:   includeRoom,
		IncludeArea:   includeArea,
		AvailableOnly: availableOnly,
	}

	entitiesWithRooms, err := h.unifiedService.GetByType(ctx, entityType, options)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to get entities by type: %s", entityType)
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve entities")
		return
	}

	meta := gin.H{
		"count":          len(entitiesWithRooms),
		"type":           entityType,
		"include_room":   includeRoom,
		"include_area":   includeArea,
		"available_only": availableOnly,
	}

	utils.SendSuccessWithMeta(c, entitiesWithRooms, meta)
}

// GetEntitiesBySource retrieves entities from a specific source
func (h *Handlers) GetEntitiesBySource(c *gin.Context) {
	sourceStr := c.Param("source")
	includeRoom := c.Query("include_room") == "true"
	includeArea := c.Query("include_area") == "true"
	availableOnly := c.Query("available_only") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Convert string to PMA source type
	source := types.PMASourceType(sourceStr)

	options := unified.GetAllOptions{
		IncludeRoom:   includeRoom,
		IncludeArea:   includeArea,
		AvailableOnly: availableOnly,
	}

	entitiesWithRooms, err := h.unifiedService.GetBySource(ctx, source, options)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to get entities by source: %s", source)
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve entities")
		return
	}

	meta := gin.H{
		"count":          len(entitiesWithRooms),
		"source":         source,
		"include_room":   includeRoom,
		"include_area":   includeArea,
		"available_only": availableOnly,
	}

	utils.SendSuccessWithMeta(c, entitiesWithRooms, meta)
}

// GetEntitiesByRoom retrieves entities in a specific room
func (h *Handlers) GetEntitiesByRoom(c *gin.Context) {
	roomID := c.Param("room_id")
	includeRoom := c.Query("include_room") == "true"
	includeArea := c.Query("include_area") == "true"
	availableOnly := c.Query("available_only") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	options := unified.GetAllOptions{
		IncludeRoom:   includeRoom,
		IncludeArea:   includeArea,
		AvailableOnly: availableOnly,
	}

	entitiesWithRooms, err := h.unifiedService.GetByRoom(ctx, roomID, options)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to get entities by room: %s", roomID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve entities")
		return
	}

	meta := gin.H{
		"count":          len(entitiesWithRooms),
		"room_id":        roomID,
		"include_room":   includeRoom,
		"include_area":   includeArea,
		"available_only": availableOnly,
	}

	utils.SendSuccessWithMeta(c, entitiesWithRooms, meta)
}

// SearchEntities searches entities using the unified PMA service
func (h *Handlers) SearchEntities(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		utils.SendError(c, http.StatusBadRequest, "Search query is required")
		return
	}

	includeRoom := c.Query("include_room") == "true"
	includeArea := c.Query("include_area") == "true"
	availableOnly := c.Query("available_only") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	options := unified.GetAllOptions{
		IncludeRoom:   includeRoom,
		IncludeArea:   includeArea,
		AvailableOnly: availableOnly,
	}

	entitiesWithRooms, err := h.unifiedService.Search(ctx, query, options)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to search entities with query: %s", query)
		utils.SendError(c, http.StatusInternalServerError, "Failed to search entities")
		return
	}

	meta := gin.H{
		"count":          len(entitiesWithRooms),
		"query":          query,
		"include_room":   includeRoom,
		"include_area":   includeArea,
		"available_only": availableOnly,
	}

	utils.SendSuccessWithMeta(c, entitiesWithRooms, meta)
}

// ExecuteEntityAction executes a control action on an entity through the unified system
func (h *Handlers) ExecuteEntityAction(c *gin.Context) {
	entityID := c.Param("id")

	var request struct {
		Action     string                 `json:"action" binding:"required"`
		Parameters map[string]interface{} `json:"parameters"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create PMA control action
	action := types.PMAControlAction{
		Action:     request.Action,
		Parameters: request.Parameters,
		EntityID:   entityID,
		Context: &types.PMAContext{
			Source:      "api",
			Timestamp:   time.Now(),
			Description: "Manual action via API",
		},
	}

	// Execute action through unified service
	result, err := h.unifiedService.ExecuteAction(ctx, action)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to execute action %s on entity %s", request.Action, entityID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to execute action")
		return
	}

	// Return the PMA control result
	utils.SendSuccess(c, result)
}

// UpdateEntityState updates entity state (deprecated - use ExecuteEntityAction instead)
func (h *Handlers) UpdateEntityState(c *gin.Context) {
	entityID := c.Param("id")

	var request struct {
		State      string                 `json:"state" binding:"required"`
		Attributes map[string]interface{} `json:"attributes"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Map state update to appropriate action
	var action string
	switch request.State {
	case "on", "true":
		action = "turn_on"
	case "off", "false":
		action = "turn_off"
	default:
		action = "set_state"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create PMA control action
	controlAction := types.PMAControlAction{
		Action:     action,
		Parameters: request.Attributes,
		EntityID:   entityID,
		Context: &types.PMAContext{
			Source:      "api",
			Timestamp:   time.Now(),
			Description: "State update via legacy API",
		},
	}

	// Execute action through unified service
	result, err := h.unifiedService.ExecuteAction(ctx, controlAction)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to update entity state: %s", entityID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to update entity state")
		return
	}

	if !result.Success {
		utils.SendError(c, http.StatusInternalServerError, "Entity state update failed")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":      "Entity state updated successfully",
		"entity_id":    entityID,
		"new_state":    result.NewState,
		"processed_at": result.ProcessedAt,
	})
}

// GetEntityTypes returns all supported entity types from the unified system
func (h *Handlers) GetEntityTypes(c *gin.Context) {
	supportedTypes := h.typeRegistry.GetSupportedEntityTypes()

	typesInfo := make([]gin.H, len(supportedTypes))
	for i, entityType := range supportedTypes {
		typeInfo, err := h.typeRegistry.GetEntityTypeInfo(entityType)
		if err != nil {
			h.log.WithError(err).Warnf("Failed to get type info for %s", entityType)
			typesInfo[i] = gin.H{
				"type": entityType,
				"name": string(entityType),
			}
			continue
		}

		typesInfo[i] = gin.H{
			"type":         typeInfo.Type,
			"name":         typeInfo.Name,
			"description":  typeInfo.Description,
			"capabilities": typeInfo.Capabilities,
			"actions":      typeInfo.Actions,
		}
	}

	utils.SendSuccessWithMeta(c, typesInfo, gin.H{
		"count": len(typesInfo),
	})
}

// GetEntityCapabilities returns all supported capabilities from the unified system
func (h *Handlers) GetEntityCapabilities(c *gin.Context) {
	capabilities := []gin.H{
		{"capability": types.CapabilityDimmable, "description": "Entity supports dimming/brightness control"},
		{"capability": types.CapabilityColorable, "description": "Entity supports color changes"},
		{"capability": types.CapabilityTemperature, "description": "Entity provides temperature readings or control"},
		{"capability": types.CapabilityHumidity, "description": "Entity provides humidity readings"},
		{"capability": types.CapabilityPosition, "description": "Entity supports position control (covers, etc.)"},
		{"capability": types.CapabilityVolume, "description": "Entity supports volume control"},
		{"capability": types.CapabilityBrightness, "description": "Entity supports brightness control"},
		{"capability": types.CapabilityMotion, "description": "Entity detects motion"},
		{"capability": types.CapabilityRecording, "description": "Entity supports recording functionality"},
		{"capability": types.CapabilityStreaming, "description": "Entity supports streaming functionality"},
		{"capability": types.CapabilityNotification, "description": "Entity supports notifications"},
		{"capability": types.CapabilityBattery, "description": "Entity reports battery status"},
		{"capability": types.CapabilityConnectivity, "description": "Entity reports connectivity status"},
	}

	utils.SendSuccessWithMeta(c, capabilities, gin.H{
		"count": len(capabilities),
	})
}

// SyncEntities triggers synchronization from all sources
func (h *Handlers) SyncEntities(c *gin.Context) {
	sourceStr := c.Query("source")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if sourceStr != "" {
		// Sync from specific source
		source := types.PMASourceType(sourceStr)
		result, err := h.unifiedService.SyncFromSource(ctx, source)
		if err != nil {
			h.log.WithError(err).Errorf("Failed to sync from source: %s", source)
			utils.SendError(c, http.StatusInternalServerError, "Failed to sync entities")
			return
		}

		utils.SendSuccess(c, result)
	} else {
		// Sync from all sources
		results, err := h.unifiedService.SyncFromAllSources(ctx)
		if err != nil {
			h.log.WithError(err).Error("Failed to sync from all sources")
			utils.SendError(c, http.StatusInternalServerError, "Failed to sync entities")
			return
		}

		// Calculate summary statistics
		totalFound := 0
		totalRegistered := 0
		totalUpdated := 0
		successCount := 0

		for _, result := range results {
			totalFound += result.EntitiesFound
			totalRegistered += result.EntitiesRegistered
			totalUpdated += result.EntitiesUpdated
			if result.Success {
				successCount++
			}
		}

		utils.SendSuccessWithMeta(c, results, gin.H{
			"total_sources":             len(results),
			"successful_sources":        successCount,
			"total_entities_found":      totalFound,
			"total_entities_registered": totalRegistered,
			"total_entities_updated":    totalUpdated,
		})
	}
}

// GetSyncStatus returns the current synchronization status
func (h *Handlers) GetSyncStatus(c *gin.Context) {
	adapters := h.adapterRegistry.GetAllAdapters()

	adapterStatus := make([]gin.H, len(adapters))
	for i, adapter := range adapters {
		health := adapter.GetHealth()
		metrics := adapter.GetMetrics()

		adapterStatus[i] = gin.H{
			"id":                adapter.GetID(),
			"name":              adapter.GetName(),
			"source":            adapter.GetSourceType(),
			"connected":         adapter.IsConnected(),
			"is_healthy":        health.IsHealthy,
			"last_health_check": health.LastHealthCheck,
			"response_time":     health.ResponseTime,
			"entities_managed":  metrics.EntitiesManaged,
			"rooms_managed":     metrics.RoomsManaged,
			"last_sync":         metrics.LastSync,
			"sync_errors":       metrics.SyncErrors,
		}
	}

	// Get registry stats
	registryStats := h.typeRegistry.GetRegistryStats()

	utils.SendSuccessWithMeta(c, adapterStatus, gin.H{
		"adapters_count":     len(adapters),
		"connected_adapters": len(h.adapterRegistry.GetConnectedAdapters()),
		"registry_stats":     registryStats,
	})
}

// CreateOrUpdateEntity creates or updates an entity (stub implementation)
func (h *Handlers) CreateOrUpdateEntity(c *gin.Context) {
	// TODO: Implement entity creation/update functionality
	utils.SendError(c, http.StatusNotImplemented, "Entity creation/update not yet implemented")
}

// DeleteEntity deletes an entity (stub implementation)
func (h *Handlers) DeleteEntity(c *gin.Context) {
	// TODO: Implement entity deletion functionality
	utils.SendError(c, http.StatusNotImplemented, "Entity deletion not yet implemented")
}
