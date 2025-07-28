package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
		// Return empty array instead of error to prevent 500 errors
		utils.SendSuccessWithMeta(c, []types.PMAEntity{}, gin.H{
			"count":          0,
			"include_room":   includeRoom,
			"include_area":   includeArea,
			"available_only": availableOnly,
			"error":          "No entities available",
		})
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
	entities := make([]types.PMAEntity, 0, len(entitiesWithRooms))

	for _, entityWithRoom := range entitiesWithRooms {
		// Extract the entity from the wrapper
		entities = append(entities, entityWithRoom.Entity)
		source := string(entityWithRoom.Entity.GetSource())
		sourceCounts[source]++
	}
	meta["by_source"] = sourceCounts

	h.log.WithField("entity_count", len(entities)).Info("Retrieved entities successfully")
	utils.SendSuccessWithMeta(c, entities, meta)
}

// GetEntity retrieves a specific entity using the unified PMA service
func (h *Handlers) GetEntity(c *gin.Context) {
	// Start debug logging for the handler call
	finish := h.debugUtils.LogCall(c.Request.Context(), "GetEntity", c.Params)
	defer finish()

	entityID := c.Param("id")
	includeRoom := c.Query("include_room") == "true"
	includeArea := c.Query("include_area") == "true"

	h.debugUtils.LogInfo(c.Request.Context(), "GetEntity", "Request parameters", gin.H{
		"entity_id":    entityID,
		"include_room": includeRoom,
		"include_area": includeArea,
	})

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	options := unified.GetEntityOptions{
		IncludeRoom: includeRoom,
		IncludeArea: includeArea,
	}

	entityWithRoom, err := h.unifiedService.GetByID(ctx, entityID, options)
	if err != nil {
		h.debugUtils.LogError(ctx, "GetEntity", err, gin.H{"entity_id": entityID})
		h.log.WithError(err).WithFields(logrus.Fields{
			"entity_id": entityID,
			"method":    "GetByID",
		}).Error("Failed to get entity")
		utils.SendError(c, http.StatusNotFound, fmt.Sprintf("Entity not found: entity ID '%s'", entityID))
		return
	}

	h.debugUtils.LogData(ctx, "GetEntity", "Response", entityWithRoom)

	h.log.WithFields(logrus.Fields{
		"entity_id":     entityID,
		"friendly_name": entityWithRoom.Entity.GetFriendlyName(),
		"state":         entityWithRoom.Entity.GetState(),
	}).Debug("Entity retrieved successfully")

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

// ExecuteEntityAction executes an action on a specific entity
func (h *Handlers) ExecuteEntityAction(c *gin.Context) {
	entityID := c.Param("id")

	h.log.WithField("entity_id", entityID).Info("🚀 ExecuteEntityAction: Starting action execution")

	// Parse the action request
	var actionRequest types.PMAControlAction
	if err := c.ShouldBindJSON(&actionRequest); err != nil {
		h.log.WithError(err).WithField("entity_id", entityID).Error("Failed to parse action request")
		utils.SendError(c, http.StatusBadRequest, "Invalid action request format")
		return
	}

	// Set the entity ID from URL parameter
	actionRequest.EntityID = entityID

	h.log.WithFields(logrus.Fields{
		"entity_id":  entityID,
		"action":     actionRequest.Action,
		"parameters": actionRequest.Parameters,
	}).Info("🔍 ExecuteEntityAction: Parsed action request")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h.log.WithField("entity_id", entityID).Info("⏰ ExecuteEntityAction: About to get entity from unified service")

	// Get the entity to determine its source
	entityWithRoom, err := h.unifiedService.GetByID(ctx, entityID, unified.GetEntityOptions{})
	if err != nil {
		h.log.WithError(err).WithField("entity_id", entityID).Error("Failed to get entity for action execution")
		utils.SendError(c, http.StatusNotFound, fmt.Sprintf("Entity not found: %s", entityID))
		return
	}

	entity := entityWithRoom.Entity
	sourceType := entity.GetSource()

	h.log.WithFields(logrus.Fields{
		"entity_id": entityID,
		"source":    sourceType,
	}).Info("🎯 ExecuteEntityAction: About to get adapter from registry")

	// Get the appropriate adapter for this entity's source
	adapter, err := h.unifiedService.GetRegistryManager().GetAdapterRegistry().GetAdapterBySource(sourceType)
	if err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"entity_id": entityID,
			"source":    sourceType,
		}).Error("Failed to get adapter for entity action")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("No adapter available for source: %s", sourceType))
		return
	}

	h.log.WithFields(logrus.Fields{
		"entity_id": entityID,
		"action":    actionRequest.Action,
		"source":    sourceType,
	}).Info("⚡ ExecuteEntityAction: About to execute action through adapter")

	// Execute the action through the adapter
	result, err := adapter.ExecuteAction(ctx, actionRequest)
	if err != nil {
		h.log.WithError(err).WithFields(logrus.Fields{
			"entity_id": entityID,
			"action":    actionRequest.Action,
			"source":    sourceType,
		}).Error("Failed to execute entity action")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Action execution failed: %v", err))
		return
	}

	h.log.WithFields(logrus.Fields{
		"entity_id": entityID,
		"action":    actionRequest.Action,
		"source":    sourceType,
		"success":   result.Success,
	}).Info("✅ ExecuteEntityAction: Action execution completed")

	// Log the result
	h.log.WithFields(logrus.Fields{
		"entity_id": entityID,
		"action":    actionRequest.Action,
		"success":   result.Success,
		"new_state": result.NewState,
		"duration":  result.Duration,
	}).Info("Entity action executed")

	// Enhanced state synchronization for rock-solid reliability
	if result.Success {
		h.log.WithFields(logrus.Fields{
			"entity_id": entityID,
			"new_state": result.NewState,
		}).Info("🔄 Starting enhanced state synchronization")

		// Immediate state update in unified service
		if result.NewState != "" {
			h.log.WithFields(logrus.Fields{
				"entity_id": entityID,
				"new_state": result.NewState,
				"source":    sourceType,
			}).Info("🔄 Attempting to update entity state in unified service")

			if _, err := h.unifiedService.UpdateEntityState(ctx, entityID, string(result.NewState), sourceType); err != nil {
				h.log.WithError(err).WithFields(logrus.Fields{
					"entity_id": entityID,
					"new_state": result.NewState,
					"source":    sourceType,
				}).Error("❌ CRITICAL: Failed to update entity state in unified service")

				// Try direct registry update as fallback
				h.log.WithField("entity_id", entityID).Info("🔄 Attempting direct registry update as fallback")
				if registryEntity, err := h.unifiedService.GetByID(ctx, entityID, unified.GetEntityOptions{}); err == nil {
					// Update the entity state directly
					switch e := registryEntity.Entity.(type) {
					case *types.PMASwitchEntity:
						e.State = types.PMAEntityState(result.NewState)
						e.LastUpdated = time.Now()
					case *types.PMALightEntity:
						e.State = types.PMAEntityState(result.NewState)
						e.LastUpdated = time.Now()
					case *types.PMASensorEntity:
						e.State = types.PMAEntityState(result.NewState)
						e.LastUpdated = time.Now()
					default:
						if baseEntity, ok := registryEntity.Entity.(*types.PMABaseEntity); ok {
							baseEntity.State = types.PMAEntityState(result.NewState)
							baseEntity.LastUpdated = time.Now()
						}
					}

					// Force update in registry
					if err := h.unifiedService.GetRegistryManager().GetEntityRegistry().UpdateEntity(registryEntity.Entity); err != nil {
						h.log.WithError(err).WithField("entity_id", entityID).Error("❌ Direct registry update also failed")
					} else {
						h.log.WithField("entity_id", entityID).Info("✅ Direct registry update succeeded")
					}
				}
			} else {
				h.log.WithFields(logrus.Fields{
					"entity_id": entityID,
					"new_state": result.NewState,
				}).Info("✅ Entity state updated successfully in unified service")

				// Immediate verification
				if verifyEntity, err := h.unifiedService.GetByID(ctx, entityID, unified.GetEntityOptions{}); err == nil {
					h.log.WithFields(logrus.Fields{
						"entity_id":        entityID,
						"expected_state":   result.NewState,
						"actual_state":     verifyEntity.Entity.GetState(),
						"timestamps_match": verifyEntity.Entity.GetLastUpdated().After(time.Now().Add(-10 * time.Second)),
					}).Info("🔍 State update verification")
				}
			}

			// Immediate WebSocket broadcast for real-time updates
			go func() {
				defer func() {
					if r := recover(); r != nil {
						h.log.WithField("panic", r).Error("Panic during immediate WebSocket broadcast")
					}
				}()

				// Get the current entity state for broadcasting
				if currentEntity, err := h.unifiedService.GetByID(context.Background(), entityID, unified.GetEntityOptions{}); err == nil {
					// Broadcast the complete entity with updated timestamp
					h.wsHub.BroadcastPMAEntityStateChange(
						entityID,
						"", // We don't have old state here, but that's okay
						currentEntity.Entity.GetState(),
						map[string]interface{}{
							"entity":        currentEntity.Entity,
							"action_result": result,
							"source":        "api_action",
							"timestamp":     time.Now().UTC(),
						},
					)
					h.log.WithField("entity_id", entityID).Info("📡 Immediate WebSocket broadcast sent")
				}
			}()
		}

		// Schedule a state validation check for external source synchronization
		go func() {
			defer func() {
				if r := recover(); r != nil {
					h.log.WithField("panic", r).Error("Panic during state validation")
				}
			}()

			// Wait for external sources to potentially update
			time.Sleep(500 * time.Millisecond)

			// Validate state consistency with source
			if adapter, err := h.unifiedService.GetRegistryManager().GetAdapterRegistry().GetAdapterBySource(sourceType); err == nil {
				h.log.WithField("entity_id", entityID).Info("🔍 Performing state validation with source")

				// Trigger a state refresh from the source
				if refresher, ok := adapter.(interface {
					RefreshEntityState(ctx context.Context, entityID string) error
				}); ok {
					refreshCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()

					if err := refresher.RefreshEntityState(refreshCtx, entityID); err != nil {
						h.log.WithError(err).WithField("entity_id", entityID).Warn("Failed to refresh entity state from source")
					} else {
						h.log.WithField("entity_id", entityID).Info("🔄 Entity state refreshed from source")
					}
				}
			}
		}()
	}

	// Return enhanced result with immediate state information
	if result.Success {
		utils.SendSuccess(c, gin.H{
			"success":      result.Success,
			"entity_id":    result.EntityID,
			"action":       result.Action,
			"new_state":    result.NewState,
			"attributes":   result.Attributes,
			"processed_at": result.ProcessedAt,
			"duration":     result.Duration.String(),
		})
	} else {
		// Action failed but adapter returned a result with error details
		utils.SendError(c, http.StatusUnprocessableEntity, fmt.Sprintf("Action failed: %s", result.Error.Message))
	}
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

// CreateOrUpdateEntity creates or updates an entity
func (h *Handlers) CreateOrUpdateEntity(c *gin.Context) {
	var entityData struct {
		ID           string                 `json:"id" binding:"required"`
		Type         types.PMAEntityType    `json:"type" binding:"required"`
		FriendlyName string                 `json:"friendly_name" binding:"required"`
		Icon         string                 `json:"icon,omitempty"`
		State        types.PMAEntityState   `json:"state,omitempty"`
		Attributes   map[string]interface{} `json:"attributes,omitempty"`
		Capabilities []types.PMACapability  `json:"capabilities,omitempty"`
		RoomID       *string                `json:"room_id,omitempty"`
		AreaID       *string                `json:"area_id,omitempty"`
		DeviceID     *string                `json:"device_id,omitempty"`
		Metadata     *types.PMAMetadata     `json:"metadata,omitempty"`
	}

	if err := c.ShouldBindJSON(&entityData); err != nil {
		utils.SendError(c, http.StatusBadRequest, fmt.Sprintf("Invalid request payload: %v", err))
		return
	}

	// Validate entity type
	validTypes := []types.PMAEntityType{
		types.EntityTypeLight,
		types.EntityTypeSwitch,
		types.EntityTypeSensor,
		types.EntityTypeClimate,
		types.EntityTypeCover,
		types.EntityTypeCamera,
		types.EntityTypeLock,
		types.EntityTypeFan,
		types.EntityTypeMediaPlayer,
		types.EntityTypeBinarySensor,
		types.EntityTypeDevice,
		types.EntityTypeGeneric,
	}

	validType := false
	for _, validT := range validTypes {
		if entityData.Type == validT {
			validType = true
			break
		}
	}

	if !validType {
		utils.SendError(c, http.StatusBadRequest, fmt.Sprintf("Invalid entity type: %s", entityData.Type))
		return
	}

	// Check if entity already exists to determine if this is create or update
	existingEntity, err := h.unifiedService.GetRegistryManager().GetEntityRegistry().GetEntity(entityData.ID)
	isUpdate := err == nil && existingEntity != nil

	// Create PMA entity
	entity := &types.PMABaseEntity{
		ID:           entityData.ID,
		Type:         entityData.Type,
		FriendlyName: entityData.FriendlyName,
		Icon:         entityData.Icon,
		State:        entityData.State,
		Attributes:   entityData.Attributes,
		LastUpdated:  time.Now(),
		Capabilities: entityData.Capabilities,
		RoomID:       entityData.RoomID,
		AreaID:       entityData.AreaID,
		DeviceID:     entityData.DeviceID,
		Available:    true, // New entities are available by default
	}

	// Set default state if not provided
	if entity.State == "" {
		if entity.Type == types.EntityTypeSwitch || entity.Type == types.EntityTypeLight {
			entity.State = types.StateOff
		} else {
			entity.State = types.StateUnknown
		}
	}

	// Set default icon if not provided
	if entity.Icon == "" {
		entity.Icon = h.getDefaultIconForEntityType(entity.Type)
	}

	// Handle metadata
	if entityData.Metadata != nil {
		entity.Metadata = entityData.Metadata
	} else {
		entity.Metadata = &types.PMAMetadata{
			Source:         types.SourcePMA, // User-created entities are PMA source
			SourceEntityID: entityData.ID,
			LastSynced:     time.Now(),
			QualityScore:   1.0, // Perfect quality for manually created entities
			IsVirtual:      false,
		}
	}

	// Set default attributes if not provided
	if entity.Attributes == nil {
		entity.Attributes = make(map[string]interface{})
	}

	// Add some default attributes based on entity type
	entity.Attributes["created_by"] = "api"
	entity.Attributes["created_at"] = time.Now()
	if isUpdate {
		entity.Attributes["updated_at"] = time.Now()
	}

	// Register or update the entity
	if isUpdate {
		err = h.unifiedService.GetRegistryManager().GetEntityRegistry().UpdateEntity(entity)
		if err != nil {
			h.log.WithError(err).Error("Failed to update entity")
			utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to update entity: %v", err))
			return
		}

		h.log.WithFields(logrus.Fields{
			"entity_id":   entity.ID,
			"entity_type": entity.Type,
		}).Info("Entity updated successfully")

		utils.SendSuccess(c, gin.H{
			"message":    "Entity updated successfully",
			"entity_id":  entity.ID,
			"operation":  "update",
			"entity":     entity,
			"updated_at": time.Now(),
		})
	} else {
		err = h.unifiedService.GetRegistryManager().GetEntityRegistry().RegisterEntity(entity)
		if err != nil {
			h.log.WithError(err).Error("Failed to create entity")
			utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to create entity: %v", err))
			return
		}

		h.log.WithFields(logrus.Fields{
			"entity_id":   entity.ID,
			"entity_type": entity.Type,
		}).Info("Entity created successfully")

		utils.SendSuccess(c, gin.H{
			"message":    "Entity created successfully",
			"entity_id":  entity.ID,
			"operation":  "create",
			"entity":     entity,
			"created_at": time.Now(),
		})
	}
}

// getDefaultIconForEntityType returns a default icon for an entity type
func (h *Handlers) getDefaultIconForEntityType(entityType types.PMAEntityType) string {
	iconMap := map[types.PMAEntityType]string{
		types.EntityTypeLight:        "mdi:lightbulb",
		types.EntityTypeSwitch:       "mdi:toggle-switch",
		types.EntityTypeSensor:       "mdi:gauge",
		types.EntityTypeClimate:      "mdi:thermostat",
		types.EntityTypeCover:        "mdi:window-shutter",
		types.EntityTypeCamera:       "mdi:camera",
		types.EntityTypeLock:         "mdi:lock",
		types.EntityTypeFan:          "mdi:fan",
		types.EntityTypeMediaPlayer:  "mdi:speaker",
		types.EntityTypeBinarySensor: "mdi:checkbox-marked-circle",
		types.EntityTypeDevice:       "mdi:chip",
		types.EntityTypeGeneric:      "mdi:circle",
	}

	if icon, exists := iconMap[entityType]; exists {
		return icon
	}
	return "mdi:help-circle"
}

// DeleteEntity deletes an entity
func (h *Handlers) DeleteEntity(c *gin.Context) {
	entityID := c.Param("entity_id")
	if entityID == "" {
		utils.SendError(c, http.StatusBadRequest, "Entity ID is required")
		return
	}

	// Check if entity exists
	entity, err := h.unifiedService.GetRegistryManager().GetEntityRegistry().GetEntity(entityID)
	if err != nil {
		utils.SendError(c, http.StatusNotFound, fmt.Sprintf("Entity not found: %s", entityID))
		return
	}

	// Don't allow deletion of entities from external sources (safety measure)
	if entity.GetSource() != types.SourcePMA {
		utils.SendError(c, http.StatusBadRequest, fmt.Sprintf("Cannot delete entity from external source: %s", entity.GetSource()))
		return
	}

	// Delete the entity
	err = h.unifiedService.GetRegistryManager().GetEntityRegistry().UnregisterEntity(entityID)
	if err != nil {
		h.log.WithError(err).Error("Failed to delete entity")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to delete entity: %v", err))
		return
	}

	h.log.WithFields(logrus.Fields{
		"entity_id":   entityID,
		"entity_type": entity.GetType(),
	}).Info("Entity deleted successfully")

	utils.SendSuccess(c, gin.H{
		"message":    "Entity deleted successfully",
		"entity_id":  entityID,
		"operation":  "delete",
		"deleted_at": time.Now(),
	})
}

// DebugSyncEntities triggers a manual entity sync for debugging
func (h *Handlers) DebugSyncEntities(c *gin.Context) {
	source := c.Query("source")
	if source == "" {
		source = "homeassistant"
	}

	h.log.WithField("source", source).Info("Debug sync triggered manually")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Trigger sync
	result, err := h.unifiedService.SyncFromSource(ctx, types.PMASourceType(source))
	if err != nil {
		h.log.WithError(err).Error("Debug sync failed")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Sync failed: %v", err))
		return
	}

	h.log.WithFields(logrus.Fields{
		"source":              result.Source,
		"success":             result.Success,
		"entities_found":      result.EntitiesFound,
		"entities_registered": result.EntitiesRegistered,
		"entities_updated":    result.EntitiesUpdated,
		"duration":            result.Duration,
		"error":               result.Error,
	}).Info("Debug sync completed")

	utils.SendSuccess(c, map[string]interface{}{
		"sync_result": result,
		"message":     "Manual sync completed",
	})
}

// DebugEntityRegistry returns debug information about the entity registry
func (h *Handlers) DebugEntityRegistry(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all entities
	allEntities, err := h.unifiedService.GetAll(ctx, unified.GetAllOptions{})
	if err != nil {
		h.log.WithError(err).Error("Failed to get all entities for debug")
		utils.SendError(c, http.StatusInternalServerError, "Failed to get entity registry info")
		return
	}

	// Group by source
	bySource := make(map[string]int)
	byType := make(map[string]int)
	sampleEntities := []map[string]interface{}{}

	for i, entityWithRoom := range allEntities {
		entity := entityWithRoom.Entity
		source := string(entity.GetSource())
		entityType := string(entity.GetType())

		bySource[source]++
		byType[entityType]++

		// Include first 10 entities as samples
		if i < 10 {
			sampleEntities = append(sampleEntities, map[string]interface{}{
				"id":            entity.GetID(),
				"friendly_name": entity.GetFriendlyName(),
				"type":          entityType,
				"source":        source,
				"state":         entity.GetState(),
				"available":     entity.IsAvailable(),
			})
		}
	}

	debugInfo := map[string]interface{}{
		"total_entities":  len(allEntities),
		"by_source":       bySource,
		"by_type":         byType,
		"sample_entities": sampleEntities,
		"registry_status": "active",
	}

	h.log.WithFields(logrus.Fields{
		"total_entities": len(allEntities),
		"by_source":      bySource,
		"by_type":        byType,
	}).Info("Entity registry debug info requested")

	utils.SendSuccess(c, debugInfo)
}
