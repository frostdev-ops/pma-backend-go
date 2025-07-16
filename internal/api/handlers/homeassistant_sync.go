package handlers

import (
	"net/http"

	"github.com/frostdev-ops/pma-backend-go/internal/core/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HomeAssistantSyncHandler handles Home Assistant synchronization API endpoints
type HomeAssistantSyncHandler struct {
	syncService homeassistant.SyncServiceInterface
	logger      *logrus.Logger
}

// NewHomeAssistantSyncHandler creates a new sync handler
func NewHomeAssistantSyncHandler(syncService homeassistant.SyncServiceInterface, logger *logrus.Logger) *HomeAssistantSyncHandler {
	return &HomeAssistantSyncHandler{
		syncService: syncService,
		logger:      logger,
	}
}

// TriggerFullSync triggers a full synchronization with Home Assistant
// @Summary Trigger full sync
// @Description Triggers a complete synchronization of all supported entities from Home Assistant
// @Tags homeassistant
// @Accept json
// @Produce json
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/ha/sync/full [post]
func (h *HomeAssistantSyncHandler) TriggerFullSync(c *gin.Context) {
	if !h.syncService.IsRunning() {
		utils.ErrorResponse(c, http.StatusBadRequest, "Sync service is not running", nil)
		return
	}

	go func() {
		if err := h.syncService.FullSync(c.Request.Context()); err != nil {
			h.logger.WithError(err).Error("Failed to perform full sync")
		}
	}()

	utils.SuccessResponse(c, "Full sync triggered successfully", map[string]interface{}{
		"message": "Full synchronization has been started in the background",
	})
}

// GetSyncStatus returns the current synchronization status
// @Summary Get sync status
// @Description Returns current synchronization status and statistics
// @Tags homeassistant
// @Accept json
// @Produce json
// @Success 200 {object} utils.SuccessResponse{data=homeassistant.SyncStats}
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/ha/sync/status [get]
func (h *HomeAssistantSyncHandler) GetSyncStatus(c *gin.Context) {
	stats := h.syncService.GetSyncStats()

	response := map[string]interface{}{
		"running":   h.syncService.IsRunning(),
		"last_sync": h.syncService.GetLastSync(),
		"stats":     stats,
	}

	utils.SuccessResponse(c, "Sync status retrieved successfully", response)
}

// SyncEntity synchronizes a specific entity
// @Summary Sync specific entity
// @Description Synchronizes a specific entity by ID with Home Assistant
// @Tags homeassistant
// @Accept json
// @Produce json
// @Param id path string true "Entity ID"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/ha/sync/entity/{id} [post]
func (h *HomeAssistantSyncHandler) SyncEntity(c *gin.Context) {
	entityID := c.Param("id")
	if entityID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Entity ID is required", nil)
		return
	}

	if !h.syncService.IsRunning() {
		utils.ErrorResponse(c, http.StatusBadRequest, "Sync service is not running", nil)
		return
	}

	if err := h.syncService.SyncEntity(c.Request.Context(), entityID); err != nil {
		h.logger.WithError(err).Errorf("Failed to sync entity: %s", entityID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to sync entity", map[string]interface{}{
			"entity_id": entityID,
			"error":     err.Error(),
		})
		return
	}

	utils.SuccessResponse(c, "Entity synchronized successfully", map[string]interface{}{
		"entity_id": entityID,
	})
}

// SyncRoom synchronizes all entities in a specific room
// @Summary Sync room entities
// @Description Synchronizes all entities in a specific room with Home Assistant
// @Tags homeassistant
// @Accept json
// @Produce json
// @Param id path string true "Room ID"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/ha/sync/room/{id} [post]
func (h *HomeAssistantSyncHandler) SyncRoom(c *gin.Context) {
	roomID := c.Param("id")
	if roomID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Room ID is required", nil)
		return
	}

	if !h.syncService.IsRunning() {
		utils.ErrorResponse(c, http.StatusBadRequest, "Sync service is not running", nil)
		return
	}

	if err := h.syncService.SyncRoom(c.Request.Context(), roomID); err != nil {
		h.logger.WithError(err).Errorf("Failed to sync room: %s", roomID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to sync room", map[string]interface{}{
			"room_id": roomID,
			"error":   err.Error(),
		})
		return
	}

	utils.SuccessResponse(c, "Room synchronized successfully", map[string]interface{}{
		"room_id": roomID,
	})
}

// CallService calls a Home Assistant service
// @Summary Call HA service
// @Description Calls a Home Assistant service with the provided data
// @Tags homeassistant
// @Accept json
// @Produce json
// @Param domain path string true "Service domain"
// @Param service path string true "Service name"
// @Param request body CallServiceRequest true "Service call data"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/ha/service/{domain}/{service} [post]
func (h *HomeAssistantSyncHandler) CallService(c *gin.Context) {
	domain := c.Param("domain")
	service := c.Param("service")

	if domain == "" || service == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Domain and service are required", nil)
		return
	}

	var request CallServiceRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if !h.syncService.IsRunning() {
		utils.ErrorResponse(c, http.StatusBadRequest, "Sync service is not running", nil)
		return
	}

	if err := h.syncService.CallService(c.Request.Context(), domain, service, request.EntityID, request.Data); err != nil {
		h.logger.WithError(err).Errorf("Failed to call service: %s.%s", domain, service)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to call service", map[string]interface{}{
			"domain":  domain,
			"service": service,
			"error":   err.Error(),
		})
		return
	}

	utils.SuccessResponse(c, "Service called successfully", map[string]interface{}{
		"domain":    domain,
		"service":   service,
		"entity_id": request.EntityID,
	})
}

// UpdateEntityState updates an entity's state in Home Assistant
// @Summary Update entity state
// @Description Updates an entity's state and attributes in Home Assistant
// @Tags homeassistant
// @Accept json
// @Produce json
// @Param id path string true "Entity ID"
// @Param request body UpdateEntityStateRequest true "Entity state update data"
// @Success 200 {object} utils.SuccessResponse
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/ha/entity/{id}/state [put]
func (h *HomeAssistantSyncHandler) UpdateEntityState(c *gin.Context) {
	entityID := c.Param("id")
	if entityID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Entity ID is required", nil)
		return
	}

	var request UpdateEntityStateRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	if !h.syncService.IsRunning() {
		utils.ErrorResponse(c, http.StatusBadRequest, "Sync service is not running", nil)
		return
	}

	if err := h.syncService.UpdateEntityState(c.Request.Context(), entityID, request.State, request.Attributes); err != nil {
		h.logger.WithError(err).Errorf("Failed to update entity state: %s", entityID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to update entity state", map[string]interface{}{
			"entity_id": entityID,
			"error":     err.Error(),
		})
		return
	}

	utils.SuccessResponse(c, "Entity state updated successfully", map[string]interface{}{
		"entity_id": entityID,
		"state":     request.State,
	})
}

// Request DTOs
type CallServiceRequest struct {
	EntityID string                 `json:"entity_id" binding:"required"`
	Data     map[string]interface{} `json:"data"`
}

type UpdateEntityStateRequest struct {
	State      interface{}            `json:"state" binding:"required"`
	Attributes map[string]interface{} `json:"attributes"`
}
