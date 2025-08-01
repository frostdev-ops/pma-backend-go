package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/area"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// Area Management Endpoints

// GetAreas retrieves all areas with optional hierarchy
func (h *Handlers) GetAreas(c *gin.Context) {
	includeInactive := c.Query("include_inactive") == "true"
	buildHierarchy := c.Query("hierarchy") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	result, err := areaService.GetAllAreas(ctx, includeInactive, buildHierarchy)
	if err != nil {
		h.log.WithError(err).Error("Failed to get areas")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve areas")
		return
	}

	utils.SendSuccessWithMeta(c, result, gin.H{
		"include_inactive": includeInactive,
		"hierarchy":        buildHierarchy,
	})
}

// CreateArea creates a new area
func (h *Handlers) CreateArea(c *gin.Context) {
	var req models.CreateAreaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	createdArea, err := areaService.CreateArea(ctx, &req)
	if err != nil {
		h.log.WithError(err).Error("Failed to create area")
		utils.SendError(c, http.StatusInternalServerError, "Failed to create area: "+err.Error())
		return
	}

	utils.SendSuccess(c, createdArea)
}

// GetArea retrieves a specific area
func (h *Handlers) GetArea(c *gin.Context) {
	areaIDStr := c.Param("id")
	areaID, err := strconv.Atoi(areaIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid area ID")
		return
	}

	includeChildren := c.Query("include_children") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	areaData, err := areaService.GetArea(ctx, areaID, includeChildren)
	if err != nil {
		h.log.WithError(err).Error("Failed to get area")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve area")
		return
	}

	if areaData == nil {
		utils.SendError(c, http.StatusNotFound, "Area not found")
		return
	}

	utils.SendSuccess(c, areaData)
}

// UpdateArea updates an existing area
func (h *Handlers) UpdateArea(c *gin.Context) {
	areaIDStr := c.Param("id")
	areaID, err := strconv.Atoi(areaIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid area ID")
		return
	}

	var req models.UpdateAreaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	updatedArea, err := areaService.UpdateArea(ctx, areaID, &req)
	if err != nil {
		h.log.WithError(err).Error("Failed to update area")
		utils.SendError(c, http.StatusInternalServerError, "Failed to update area: "+err.Error())
		return
	}

	utils.SendSuccess(c, updatedArea)
}

// DeleteArea deletes an area
func (h *Handlers) DeleteArea(c *gin.Context) {
	areaIDStr := c.Param("id")
	areaID, err := strconv.Atoi(areaIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid area ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	if err := areaService.DeleteArea(ctx, areaID); err != nil {
		h.log.WithError(err).Error("Failed to delete area")
		utils.SendError(c, http.StatusInternalServerError, "Failed to delete area: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{"message": "Area deleted successfully"})
}

// Area Mapping Endpoints

// GetAreaMappings retrieves all area mappings
func (h *Handlers) GetAreaMappings(c *gin.Context) {
	externalSystem := c.Query("external_system")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	mappings, err := areaService.GetAreaMappings(ctx, externalSystem)
	if err != nil {
		h.log.WithError(err).Error("Failed to get area mappings")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve area mappings")
		return
	}

	utils.SendSuccessWithMeta(c, mappings, gin.H{
		"count":           len(mappings),
		"external_system": externalSystem,
	})
}

// CreateAreaMapping creates a new area mapping
func (h *Handlers) CreateAreaMapping(c *gin.Context) {
	var req models.CreateAreaMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Set default external system if not provided
	if req.ExternalSystem == "" {
		req.ExternalSystem = models.ExternalSystemHomeAssistant
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	mapping, err := areaService.CreateAreaMapping(ctx, &req)
	if err != nil {
		h.log.WithError(err).Error("Failed to create area mapping")
		utils.SendError(c, http.StatusInternalServerError, "Failed to create area mapping: "+err.Error())
		return
	}

	utils.SendSuccess(c, mapping)
}

// UpdateAreaMapping updates an existing area mapping
func (h *Handlers) UpdateAreaMapping(c *gin.Context) {
	mappingIDStr := c.Param("id")
	mappingID, err := strconv.Atoi(mappingIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid mapping ID")
		return
	}

	var req models.UpdateAreaMappingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	mapping, err := areaService.UpdateAreaMapping(ctx, mappingID, &req)
	if err != nil {
		h.log.WithError(err).Error("Failed to update area mapping")
		utils.SendError(c, http.StatusInternalServerError, "Failed to update area mapping: "+err.Error())
		return
	}

	utils.SendSuccess(c, mapping)
}

// DeleteAreaMapping deletes an area mapping
func (h *Handlers) DeleteAreaMapping(c *gin.Context) {
	mappingIDStr := c.Param("id")
	mappingID, err := strconv.Atoi(mappingIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid mapping ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	if err := areaService.DeleteAreaMapping(ctx, mappingID); err != nil {
		h.log.WithError(err).Error("Failed to delete area mapping")
		utils.SendError(c, http.StatusInternalServerError, "Failed to delete area mapping: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{"message": "Area mapping deleted successfully"})
}

// Enhanced Room Management

// TriggerAreaSync triggers synchronization with external systems
func (h *Handlers) TriggerAreaSync(c *gin.Context) {
	var req models.SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Set defaults
	if req.SyncType == "" {
		req.SyncType = models.SyncTypeManual
	}
	if req.ExternalSystem == "" {
		req.ExternalSystem = models.ExternalSystemHomeAssistant
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	syncLog, err := areaService.SyncWithExternalSystem(ctx, &req)
	if err != nil {
		h.log.WithError(err).Error("Failed to trigger area sync")
		utils.SendError(c, http.StatusInternalServerError, "Failed to trigger synchronization: "+err.Error())
		return
	}

	utils.SendSuccess(c, syncLog)
}

// GetAreaSyncStatus retrieves synchronization status
func (h *Handlers) GetAreaSyncStatus(c *gin.Context) {
	externalSystem := c.Query("external_system")
	if externalSystem == "" {
		externalSystem = models.ExternalSystemHomeAssistant
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	syncStatus, err := areaService.GetSyncStatus(ctx, externalSystem)
	if err != nil {
		h.log.WithError(err).Error("Failed to get sync status")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve sync status")
		return
	}

	if syncStatus == nil {
		utils.SendSuccess(c, gin.H{
			"status":          "no_sync",
			"external_system": externalSystem,
		})
		return
	}

	utils.SendSuccess(c, syncStatus)
}

// GetAreaSyncHistory retrieves synchronization history
func (h *Handlers) GetAreaSyncHistory(c *gin.Context) {
	externalSystem := c.Query("external_system")
	if externalSystem == "" {
		externalSystem = models.ExternalSystemHomeAssistant
	}

	limitStr := c.Query("limit")
	limit := 10
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	history, err := areaService.GetSyncHistory(ctx, externalSystem, limit)
	if err != nil {
		h.log.WithError(err).Error("Failed to get sync history")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve sync history")
		return
	}

	utils.SendSuccessWithMeta(c, history, gin.H{
		"count":           len(history),
		"external_system": externalSystem,
		"limit":           limit,
	})
}

// System Status and Analytics

// GetAreaStatus retrieves overall area system status
func (h *Handlers) GetAreaStatus(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	status, err := areaService.GetAreaStatus(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get area status")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve area status")
		return
	}

	utils.SendSuccess(c, status)
}

// GetAreaAnalytics retrieves area analytics
func (h *Handlers) GetAreaAnalytics(c *gin.Context) {
	var req models.AreaAnalyticsRequest

	// Parse query parameters
	if areaIDsStr := c.Query("area_ids"); areaIDsStr != "" {
		// Parse comma-separated area IDs
		// This is a simplified implementation
		if areaID, err := strconv.Atoi(areaIDsStr); err == nil {
			req.AreaIDs = []int{areaID}
		}
	}

	if metricsStr := c.Query("metrics"); metricsStr != "" {
		// Parse comma-separated metrics
		req.Metrics = []string{metricsStr}
	}

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if startDate, err := time.Parse(time.RFC3339, startDateStr); err == nil {
			req.StartDate = &startDate
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if endDate, err := time.Parse(time.RFC3339, endDateStr); err == nil {
			req.EndDate = &endDate
		}
	}

	req.TimePeriod = c.Query("time_period")
	req.Grouping = c.Query("grouping")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	analytics, err := areaService.GetAreaAnalytics(ctx, &req)
	if err != nil {
		h.log.WithError(err).Error("Failed to get area analytics")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve area analytics")
		return
	}

	utils.SendSuccessWithMeta(c, analytics, gin.H{
		"count":       len(analytics),
		"time_period": req.TimePeriod,
		"grouping":    req.Grouping,
	})
}

// GetAreaAnalyticsSummary retrieves analytics summary
func (h *Handlers) GetAreaAnalyticsSummary(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	req := &models.AreaAnalyticsRequest{}
	analytics, err := areaService.GetAreaAnalytics(ctx, req)
	if err != nil {
		h.log.WithError(err).Error("Failed to get area analytics summary")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve analytics summary")
		return
	}

	// Generate summary statistics
	summary := gin.H{
		"total_areas":    len(analytics),
		"analytics_data": analytics,
		"generated_at":   time.Now(),
	}

	utils.SendSuccess(c, summary)
}

// Settings Management

// GetAreaSettings retrieves area settings
func (h *Handlers) GetAreaSettings(c *gin.Context) {
	var areaID *int
	if areaIDStr := c.Query("area_id"); areaIDStr != "" {
		if id, err := strconv.Atoi(areaIDStr); err == nil {
			areaID = &id
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	settings, err := areaService.GetSettings(ctx, areaID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get area settings")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve area settings")
		return
	}

	utils.SendSuccess(c, settings)
}

// UpdateAreaSettings updates area settings
func (h *Handlers) UpdateAreaSettings(c *gin.Context) {
	var areaID *int
	if areaIDStr := c.Query("area_id"); areaIDStr != "" {
		if id, err := strconv.Atoi(areaIDStr); err == nil {
			areaID = &id
		}
	}

	var settings models.AreaSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	settings.AreaID = areaID

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	if err := areaService.UpdateSettings(ctx, areaID, &settings); err != nil {
		h.log.WithError(err).Error("Failed to update area settings")
		utils.SendError(c, http.StatusInternalServerError, "Failed to update area settings: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{"message": "Area settings updated successfully"})
}

// Room-Area Assignment Endpoints

// GetRoomAreaAssignments retrieves area assignments for a room
func (h *Handlers) GetRoomAreaAssignments(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assignments, err := h.repos.Area.GetRoomAreaAssignments(ctx, roomID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get room area assignments")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve room area assignments")
		return
	}

	utils.SendSuccessWithMeta(c, assignments, gin.H{
		"room_id": roomID,
		"count":   len(assignments),
	})
}

// GetAreaRoomAssignments retrieves room assignments for an area
func (h *Handlers) GetAreaRoomAssignments(c *gin.Context) {
	areaIDStr := c.Param("id")
	areaID, err := strconv.Atoi(areaIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid area ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	assignments, err := h.repos.Area.GetAreaRoomAssignments(ctx, areaID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get area room assignments")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve area room assignments")
		return
	}

	utils.SendSuccessWithMeta(c, assignments, gin.H{
		"area_id": areaID,
		"count":   len(assignments),
	})
}

// Entity Management for Areas

// GetAreaEntities retrieves entities in an area
func (h *Handlers) GetAreaEntities(c *gin.Context) {
	areaIDStr := c.Param("id")
	areaID, err := strconv.Atoi(areaIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid area ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	entities, err := areaService.GetAreaEntities(ctx, areaID)
	if err != nil {
		h.log.WithError(err).Error("Failed to get area entities")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve area entities")
		return
	}

	utils.SendSuccessWithMeta(c, entities, gin.H{
		"area_id": areaID,
		"count":   len(entities),
	})
}

// AssignEntitiesToArea assigns entities to an area
func (h *Handlers) AssignEntitiesToArea(c *gin.Context) {
	areaIDStr := c.Param("id")
	areaID, err := strconv.Atoi(areaIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid area ID")
		return
	}

	var req struct {
		EntityIDs []string `json:"entity_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	result, err := areaService.AssignEntitiesToArea(ctx, areaID, req.EntityIDs)
	if err != nil {
		h.log.WithError(err).Error("Failed to assign entities to area")
		utils.SendError(c, http.StatusInternalServerError, "Failed to assign entities to area: "+err.Error())
		return
	}

	utils.SendSuccessWithMeta(c, result, gin.H{
		"area_id":        areaID,
		"assigned_count": len(req.EntityIDs),
	})
}

// RemoveEntityFromArea removes an entity from an area
func (h *Handlers) RemoveEntityFromArea(c *gin.Context) {
	areaIDStr := c.Param("id")
	areaID, err := strconv.Atoi(areaIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid area ID")
		return
	}

	entityID := c.Param("entity_id")
	if entityID == "" {
		utils.SendError(c, http.StatusBadRequest, "Entity ID is required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	err = areaService.RemoveEntityFromArea(ctx, areaID, entityID)
	if err != nil {
		h.log.WithError(err).Error("Failed to remove entity from area")
		utils.SendError(c, http.StatusInternalServerError, "Failed to remove entity from area: "+err.Error())
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Entity removed from area successfully",
		"area_id":   areaID,
		"entity_id": entityID,
	})
}
