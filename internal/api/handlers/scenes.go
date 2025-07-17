package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// GetScenes returns all available Home Assistant scenes
func (h *Handlers) GetScenes(c *gin.Context) {
	ctx := c.Request.Context()

	// Get Home Assistant client from context or create new one
	haClient, err := homeassistant.NewClient(h.cfg, h.repos.Config, h.log)
	if err != nil {
		h.log.WithError(err).Error("Failed to create Home Assistant client")
		utils.SendError(c, http.StatusInternalServerError, "Failed to connect to Home Assistant")
		return
	}

	// Initialize client
	if err := haClient.Initialize(ctx); err != nil {
		h.log.WithError(err).Error("Failed to initialize Home Assistant client")
		utils.SendError(c, http.StatusInternalServerError, "Failed to initialize Home Assistant connection")
		return
	}

	// Get all entity states and filter for scenes
	states, err := haClient.GetStates(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get states from Home Assistant")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve scenes")
		return
	}

	// Filter for scene entities
	var scenes []interface{}
	for _, state := range states {
		if strings.HasPrefix(state.EntityID, "scene.") {
			scenes = append(scenes, map[string]interface{}{
				"entity_id":    state.EntityID,
				"state":        state.State,
				"attributes":   state.Attributes,
				"last_changed": state.LastChanged,
				"last_updated": state.LastUpdated,
			})
		}
	}

	utils.SendSuccess(c, gin.H{
		"scenes": scenes,
		"count":  len(scenes),
	})
}

// GetScene returns a specific Home Assistant scene by ID
func (h *Handlers) GetScene(c *gin.Context) {
	sceneID := c.Param("id")
	if sceneID == "" {
		utils.SendError(c, http.StatusBadRequest, "Scene ID is required")
		return
	}

	ctx := c.Request.Context()

	// Get Home Assistant client
	haClient, err := homeassistant.NewClient(h.cfg, h.repos.Config, h.log)
	if err != nil {
		h.log.WithError(err).Error("Failed to create Home Assistant client")
		utils.SendError(c, http.StatusInternalServerError, "Failed to connect to Home Assistant")
		return
	}

	// Initialize client
	if err := haClient.Initialize(ctx); err != nil {
		h.log.WithError(err).Error("Failed to initialize Home Assistant client")
		utils.SendError(c, http.StatusInternalServerError, "Failed to initialize Home Assistant connection")
		return
	}

	// Get specific scene state
	scene, err := haClient.GetState(ctx, sceneID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get scene from Home Assistant")
		utils.SendError(c, http.StatusNotFound, "Scene not found")
		return
	}

	utils.SendSuccess(c, scene)
}

// ActivateScene activates a Home Assistant scene
func (h *Handlers) ActivateScene(c *gin.Context) {
	sceneID := c.Param("id")
	if sceneID == "" {
		utils.SendError(c, http.StatusBadRequest, "Scene ID is required")
		return
	}

	ctx := c.Request.Context()

	// Get Home Assistant client
	haClient, err := homeassistant.NewClient(h.cfg, h.repos.Config, h.log)
	if err != nil {
		h.log.WithError(err).Error("Failed to create Home Assistant client")
		utils.SendError(c, http.StatusInternalServerError, "Failed to connect to Home Assistant")
		return
	}

	// Initialize client
	if err := haClient.Initialize(ctx); err != nil {
		h.log.WithError(err).Error("Failed to initialize Home Assistant client")
		utils.SendError(c, http.StatusInternalServerError, "Failed to initialize Home Assistant connection")
		return
	}

	// Activate scene using scene.turn_on service call
	serviceData := map[string]interface{}{
		"entity_id": sceneID,
	}

	if err := haClient.CallService(ctx, "scene", "turn_on", serviceData); err != nil {
		h.log.WithError(err).Error("Failed to activate scene")
		utils.SendError(c, http.StatusInternalServerError, "Failed to activate scene")
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
	})
}
