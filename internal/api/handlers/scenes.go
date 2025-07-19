package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetScenes returns all available scenes using the unified PMA service
func (h *Handlers) GetScenes(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get scenes through unified service
	// Scenes are PMA entities with type "scene"
	options := unified.GetAllOptions{
		Domain: "scene", // Filter for scene entities only
	}

	scenesWithRooms, err := h.unifiedService.GetAll(ctx, options)
	if err != nil {
		h.log.WithError(err).Error("Failed to get scenes from unified service")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve scenes")
		return
	}

	// Extract just the entities (scenes)
	scenes := make([]types.PMAEntity, len(scenesWithRooms))
	for i, swr := range scenesWithRooms {
		scenes[i] = swr.Entity
	}

	utils.SendSuccess(c, gin.H{
		"scenes": scenes,
		"count":  len(scenes),
	})
}

// GetScene returns a specific scene by ID using the unified PMA service
func (h *Handlers) GetScene(c *gin.Context) {
	sceneID := c.Param("id")
	if sceneID == "" {
		utils.SendError(c, http.StatusBadRequest, "Scene ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Get specific scene through unified service
	options := unified.GetEntityOptions{}

	sceneWithRoom, err := h.unifiedService.GetByID(ctx, sceneID, options)
	if err != nil {
		h.log.WithError(err).Error("Failed to get scene from unified service")
		utils.SendError(c, http.StatusNotFound, "Scene not found")
		return
	}

	utils.SendSuccess(c, sceneWithRoom.Entity)
}

// ActivateScene activates a scene using the unified PMA service
func (h *Handlers) ActivateScene(c *gin.Context) {
	sceneID := c.Param("id")
	if sceneID == "" {
		utils.SendError(c, http.StatusBadRequest, "Scene ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create PMA control action for scene activation
	action := types.PMAControlAction{
		EntityID: sceneID,
		Action:   "turn_on",
		Context: &types.PMAContext{
			ID:          uuid.New().String(),
			Source:      "api",
			Timestamp:   time.Now(),
			Description: "Activate scene via API",
		},
	}

	// Execute through unified service
	result, err := h.unifiedService.ExecuteAction(ctx, action)
	if err != nil {
		h.log.WithError(err).Error("Failed to activate scene")
		utils.SendError(c, http.StatusInternalServerError, "Failed to activate scene")
		return
	}

	if !result.Success {
		h.log.Errorf("Scene activation failed: %s", result.Error.Message)
		utils.SendError(c, http.StatusBadRequest, result.Error.Message)
		return
	}

	// Broadcast scene activation via WebSocket
	if h.wsHub != nil {
		data := map[string]interface{}{
			"scene_id":  sceneID,
			"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
		}
		go h.wsHub.BroadcastToAll("scene_activated", data)
	}

	utils.SendSuccess(c, gin.H{
		"message":  "Scene activated successfully",
		"scene_id": sceneID,
		"result":   result,
	})
}
