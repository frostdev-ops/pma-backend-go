package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OpenCover opens a cover fully
// @Summary Open cover
// @Description Opens a cover to 100% position
// @Tags cover
// @Accept json
// @Produce json
// @Param id path string true "Entity ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/v1/entities/{id}/cover/open [post]
func (h *Handlers) OpenCover(c *gin.Context) {
	entityID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create PMA control action
	action := types.PMAControlAction{
		EntityID: entityID,
		Action:   "open",
		Context: &types.PMAContext{
			ID:          uuid.New().String(),
			Source:      "api",
			Timestamp:   time.Now(),
			Description: "Open cover via API",
		},
	}

	// Execute through unified service
	result, err := h.unifiedService.ExecuteAction(ctx, action)
	if err != nil {
		h.log.WithError(err).Error("Failed to open cover")
		utils.SendError(c, http.StatusInternalServerError, "Failed to open cover")
		return
	}

	if !result.Success {
		if result.Error != nil && result.Error.Code == "ENTITY_NOT_FOUND" {
			utils.SendError(c, http.StatusNotFound, "Cover not found")
			return
		}
		utils.SendError(c, http.StatusBadRequest, result.Error.Message)
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Cover opened successfully",
		"entity_id": entityID,
		"result":    result,
	})
}

// CloseCover closes a cover fully
// @Summary Close cover
// @Description Closes a cover to 0% position
// @Tags cover
// @Accept json
// @Produce json
// @Param id path string true "Entity ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/v1/entities/{id}/cover/close [post]
func (h *Handlers) CloseCover(c *gin.Context) {
	entityID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create PMA control action
	action := types.PMAControlAction{
		EntityID: entityID,
		Action:   "close",
		Context: &types.PMAContext{
			ID:          uuid.New().String(),
			Source:      "api",
			Timestamp:   time.Now(),
			Description: "Close cover via API",
		},
	}

	// Execute through unified service
	result, err := h.unifiedService.ExecuteAction(ctx, action)
	if err != nil {
		h.log.WithError(err).Error("Failed to close cover")
		utils.SendError(c, http.StatusInternalServerError, "Failed to close cover")
		return
	}

	if !result.Success {
		if result.Error != nil && result.Error.Code == "ENTITY_NOT_FOUND" {
			utils.SendError(c, http.StatusNotFound, "Cover not found")
			return
		}
		utils.SendError(c, http.StatusBadRequest, result.Error.Message)
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Cover closed successfully",
		"entity_id": entityID,
		"result":    result,
	})
}

// StopCover stops cover movement
// @Summary Stop cover
// @Description Stops current cover movement
// @Tags cover
// @Accept json
// @Produce json
// @Param id path string true "Entity ID"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/v1/entities/{id}/cover/stop [post]
func (h *Handlers) StopCover(c *gin.Context) {
	entityID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create PMA control action
	action := types.PMAControlAction{
		EntityID: entityID,
		Action:   "stop",
		Context: &types.PMAContext{
			ID:          uuid.New().String(),
			Source:      "api",
			Timestamp:   time.Now(),
			Description: "Stop cover via API",
		},
	}

	// Execute through unified service
	result, err := h.unifiedService.ExecuteAction(ctx, action)
	if err != nil {
		h.log.WithError(err).Error("Failed to stop cover")
		utils.SendError(c, http.StatusInternalServerError, "Failed to stop cover")
		return
	}

	if !result.Success {
		if result.Error != nil && result.Error.Code == "ENTITY_NOT_FOUND" {
			utils.SendError(c, http.StatusNotFound, "Cover not found")
			return
		}
		utils.SendError(c, http.StatusBadRequest, result.Error.Message)
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Cover stopped successfully",
		"entity_id": entityID,
		"result":    result,
	})
}

// SetCoverPosition sets a specific position for the cover
// @Summary Set cover position
// @Description Sets cover position to a specific value (0-100)
// @Tags cover
// @Accept json
// @Produce json
// @Param id path string true "Entity ID"
// @Param request body SetCoverPositionRequest true "Position data"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/v1/entities/{id}/cover/position [put]
func (h *Handlers) SetCoverPosition(c *gin.Context) {
	entityID := c.Param("id")

	var request SetCoverPositionRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create PMA control action with position parameter
	action := types.PMAControlAction{
		EntityID: entityID,
		Action:   "set_position",
		Parameters: map[string]interface{}{
			"position": request.Position,
		},
		Context: &types.PMAContext{
			ID:          uuid.New().String(),
			Source:      "api",
			Timestamp:   time.Now(),
			Description: "Set cover position via API",
		},
	}

	// Execute through unified service
	result, err := h.unifiedService.ExecuteAction(ctx, action)
	if err != nil {
		h.log.WithError(err).Error("Failed to set cover position")
		utils.SendError(c, http.StatusInternalServerError, "Failed to set cover position")
		return
	}

	if !result.Success {
		if result.Error != nil && result.Error.Code == "ENTITY_NOT_FOUND" {
			utils.SendError(c, http.StatusNotFound, "Cover not found")
			return
		}
		utils.SendError(c, http.StatusBadRequest, result.Error.Message)
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Cover position set successfully",
		"entity_id": entityID,
		"position":  request.Position,
		"result":    result,
	})
}

// SetCoverTilt sets the tilt position for venetian blinds
// @Summary Set cover tilt
// @Description Sets cover tilt position for venetian blinds (0-100)
// @Tags cover
// @Accept json
// @Produce json
// @Param id path string true "Entity ID"
// @Param request body SetCoverTiltRequest true "Tilt position data"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/v1/entities/{id}/cover/tilt [put]
func (h *Handlers) SetCoverTilt(c *gin.Context) {
	entityID := c.Param("id")

	var request SetCoverTiltRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create PMA control action with tilt parameter
	action := types.PMAControlAction{
		EntityID: entityID,
		Action:   "set_tilt",
		Parameters: map[string]interface{}{
			"tilt_position": request.TiltPosition,
		},
		Context: &types.PMAContext{
			ID:          uuid.New().String(),
			Source:      "api",
			Timestamp:   time.Now(),
			Description: "Set cover tilt via API",
		},
	}

	// Execute through unified service
	result, err := h.unifiedService.ExecuteAction(ctx, action)
	if err != nil {
		h.log.WithError(err).Error("Failed to set cover tilt")
		utils.SendError(c, http.StatusInternalServerError, "Failed to set cover tilt")
		return
	}

	if !result.Success {
		if result.Error != nil && result.Error.Code == "ENTITY_NOT_FOUND" {
			utils.SendError(c, http.StatusNotFound, "Cover not found")
			return
		}
		utils.SendError(c, http.StatusBadRequest, result.Error.Message)
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":       "Cover tilt set successfully",
		"entity_id":     entityID,
		"tilt_position": request.TiltPosition,
		"result":        result,
	})
}

// GetCoverStatus retrieves the current status of a cover
// @Summary Get cover status
// @Description Retrieves current cover status including position, tilt, and capabilities
// @Tags cover
// @Accept json
// @Produce json
// @Param id path string true "Entity ID"
// @Success 200 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/v1/entities/{id}/cover/status [get]
func (h *Handlers) GetCoverStatus(c *gin.Context) {
	entityID := c.Param("id")

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	// Get cover entity through unified service
	options := unified.GetEntityOptions{}
	coverWithRoom, err := h.unifiedService.GetByID(ctx, entityID, options)
	if err != nil {
		h.log.WithError(err).Error("Failed to get cover entity")
		utils.SendError(c, http.StatusNotFound, "Cover not found")
		return
	}

	cover := coverWithRoom.Entity

	// Prepare status response using PMA entity data
	status := gin.H{
		"entity_id":     cover.GetID(),
		"friendly_name": cover.GetFriendlyName(),
		"state":         cover.GetState(),
		"attributes":    cover.GetAttributes(),
		"capabilities":  cover.GetCapabilities(),
		"available":     cover.IsAvailable(),
		"last_updated":  cover.GetLastUpdated(),
	}

	utils.SendSuccess(c, gin.H{
		"entity_id": entityID,
		"status":    status,
	})
}

// OperateCoversInGroup operates multiple covers simultaneously
// @Summary Group cover operation
// @Description Operate multiple covers simultaneously
// @Tags cover
// @Accept json
// @Produce json
// @Param request body GroupCoverOperationRequest true "Group operation data"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/v1/covers/group-operation [post]
func (h *Handlers) OperateCoversInGroup(c *gin.Context) {
	var request GroupCoverOperationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	if len(request.EntityIDs) == 0 {
		utils.SendError(c, http.StatusBadRequest, "No entity IDs provided")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	// Execute action on each cover entity
	results := make([]gin.H, 0, len(request.EntityIDs))
	successCount := 0

	for _, entityID := range request.EntityIDs {
		// Prepare parameters based on operation
		parameters := make(map[string]interface{})
		if request.Position != nil {
			parameters["position"] = *request.Position
		}
		if request.TiltPosition != nil {
			parameters["tilt_position"] = *request.TiltPosition
		}

		// Create PMA control action
		action := types.PMAControlAction{
			EntityID:   entityID,
			Action:     request.Operation,
			Parameters: parameters,
			Context: &types.PMAContext{
				ID:          uuid.New().String(),
				Source:      "api",
				Timestamp:   time.Now(),
				Description: "Group cover operation via API",
			},
		}

		// Execute through unified service
		result, err := h.unifiedService.ExecuteAction(ctx, action)
		if err != nil {
			h.log.WithError(err).Errorf("Failed to execute group operation on cover %s", entityID)
			results = append(results, gin.H{
				"entity_id": entityID,
				"success":   false,
				"error":     err.Error(),
			})
			continue
		}

		if result.Success {
			successCount++
		}

		results = append(results, gin.H{
			"entity_id": entityID,
			"success":   result.Success,
			"result":    result,
		})
	}

	utils.SendSuccess(c, gin.H{
		"message":      "Group operation completed",
		"operation":    request.Operation,
		"total_covers": len(request.EntityIDs),
		"successful":   successCount,
		"failed":       len(request.EntityIDs) - successCount,
		"results":      results,
	})
}

// Request types for cover operations

// SetCoverPositionRequest represents a request to set cover position
type SetCoverPositionRequest struct {
	Position int `json:"position" binding:"required,min=0,max=100"`
}

// SetCoverTiltRequest represents a request to set cover tilt position
type SetCoverTiltRequest struct {
	TiltPosition int `json:"tilt_position" binding:"required,min=0,max=100"`
}

// GroupCoverOperationRequest represents a request for group cover operations
type GroupCoverOperationRequest struct {
	EntityIDs    []string `json:"entity_ids" binding:"required"`
	Operation    string   `json:"operation" binding:"required,oneof=open close stop set_position set_tilt"`
	Position     *int     `json:"position,omitempty" binding:"omitempty,min=0,max=100"`
	TiltPosition *int     `json:"tilt_position,omitempty" binding:"omitempty,min=0,max=100"`
}

// SetCoverPreset sets a cover to a preset position
// @Summary Set cover preset
// @Description Sets cover to a predefined preset position
// @Tags cover
// @Accept json
// @Produce json
// @Param id path string true "Entity ID"
// @Param request body SetCoverPresetRequest true "Preset data"
// @Success 200 {object} utils.APIResponse
// @Failure 400 {object} utils.APIResponse
// @Failure 404 {object} utils.APIResponse
// @Failure 500 {object} utils.APIResponse
// @Router /api/v1/entities/{id}/cover/preset [put]
func (h *Handlers) SetCoverPreset(c *gin.Context) {
	entityID := c.Param("id")

	var request SetCoverPresetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Map preset names to positions
	presetPositions := map[string]int{
		"closed":    0,
		"privacy":   25,
		"half_open": 50,
		"ventilate": 75,
		"open":      100,
	}

	position, exists := presetPositions[request.Preset]
	if !exists {
		utils.SendError(c, http.StatusBadRequest, "Invalid preset. Valid presets: closed, privacy, half_open, ventilate, open")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	// Create PMA control action with preset position
	action := types.PMAControlAction{
		EntityID: entityID,
		Action:   "set_position",
		Parameters: map[string]interface{}{
			"position": position,
		},
		Context: &types.PMAContext{
			ID:          uuid.New().String(),
			Source:      "api",
			Timestamp:   time.Now(),
			Description: "Set cover preset via API",
		},
	}

	// Execute through unified service
	result, err := h.unifiedService.ExecuteAction(ctx, action)
	if err != nil {
		h.log.WithError(err).Error("Failed to set cover preset")
		utils.SendError(c, http.StatusInternalServerError, "Failed to set cover preset")
		return
	}

	if !result.Success {
		if result.Error != nil && result.Error.Code == "ENTITY_NOT_FOUND" {
			utils.SendError(c, http.StatusNotFound, "Cover not found")
			return
		}
		utils.SendError(c, http.StatusBadRequest, result.Error.Message)
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Cover preset set successfully",
		"entity_id": entityID,
		"preset":    request.Preset,
		"position":  position,
		"result":    result,
	})
}

// SetCoverPresetRequest represents a request to set a cover preset
type SetCoverPresetRequest struct {
	Preset string `json:"preset" binding:"required,oneof=closed privacy half_open ventilate open"`
}
