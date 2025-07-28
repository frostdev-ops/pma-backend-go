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

// Bulk Operations for Area → Room → Entity hierarchy

// GetAreaWithFullHierarchy retrieves a complete area with all rooms and entities
func (h *Handlers) GetAreaWithFullHierarchy(c *gin.Context) {
	areaIDStr := c.Param("id")
	areaID, err := strconv.Atoi(areaIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid area ID")
		return
	}

	includeEntities := c.Query("include_entities") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	areaData, err := areaService.GetAreaWithFullHierarchy(ctx, areaID, includeEntities)
	if err != nil {
		h.log.WithError(err).Error("Failed to get area with full hierarchy")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve area hierarchy")
		return
	}

	if areaData == nil {
		utils.SendError(c, http.StatusNotFound, "Area not found")
		return
	}

	utils.SendSuccess(c, areaData)
}

// GetAreaSummaries retrieves dashboard-friendly summaries of all areas
func (h *Handlers) GetAreaSummaries(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	summaries, err := h.repos.Area.GetAreaSummaries(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get area summaries")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve area summaries")
		return
	}

	utils.SendSuccess(c, summaries)
}

// ExecuteBulkAreaAction performs bulk actions on all entities within an area
func (h *Handlers) ExecuteBulkAreaAction(c *gin.Context) {
	var req models.BulkAreaAction
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	areaService := area.NewService(h.repos.Area, h.repos.Room, h.repos.Entity, h.unifiedService, h.log, h.cfg)

	result, err := areaService.ExecuteBulkAction(ctx, &req)
	if err != nil {
		h.log.WithError(err).Error("Failed to execute bulk area action")
		utils.SendError(c, http.StatusInternalServerError, "Failed to execute bulk action: "+err.Error())
		return
	}

	utils.SendSuccess(c, result)
}

// GetAreaEntitiesForAction retrieves entities that would be affected by a bulk action
func (h *Handlers) GetAreaEntitiesForAction(c *gin.Context) {
	areaIDStr := c.Param("id")
	areaID, err := strconv.Atoi(areaIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid area ID")
		return
	}

	var filters models.BulkActionFilters
	if err := c.ShouldBindJSON(&filters); err != nil {
		// If no body provided, use empty filters (all entities)
		filters = models.BulkActionFilters{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	entities, err := h.repos.Area.GetAreaEntitiesForBulkAction(ctx, areaID, filters)
	if err != nil {
		h.log.WithError(err).Error("Failed to get area entities for action")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve entities")
		return
	}

	utils.SendSuccessWithMeta(c, entities, gin.H{
		"area_id":      areaID,
		"entity_count": len(entities),
		"filters":      filters,
	})
}

// AssignRoomToArea assigns a room to an area using the simplified hierarchy
func (h *Handlers) AssignRoomToArea(c *gin.Context) {
	areaIDStr := c.Param("id")
	areaID, err := strconv.Atoi(areaIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid area ID")
		return
	}

	roomIDStr := c.Param("room_id")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h.repos.Room.AssignToArea(ctx, roomID, &areaID)
	if err != nil {
		h.log.WithError(err).Error("Failed to assign room to area")
		utils.SendError(c, http.StatusInternalServerError, "Failed to assign room to area")
		return
	}

	utils.SendSuccessWithMeta(c, nil, gin.H{
		"message": "Room successfully assigned to area",
		"area_id": areaID,
		"room_id": roomID,
	})
}

// RemoveRoomFromArea removes a room from an area (sets area_id to NULL)
func (h *Handlers) RemoveRoomFromArea(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = h.repos.Room.AssignToArea(ctx, roomID, nil)
	if err != nil {
		h.log.WithError(err).Error("Failed to remove room from area")
		utils.SendError(c, http.StatusInternalServerError, "Failed to remove room from area")
		return
	}

	utils.SendSuccessWithMeta(c, nil, gin.H{
		"message": "Room successfully removed from area",
		"room_id": roomID,
	})
}

// GetUnassignedRooms retrieves all rooms that are not assigned to any area
func (h *Handlers) GetUnassignedRooms(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rooms, err := h.repos.Room.GetUnassignedRooms(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to get unassigned rooms")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve unassigned rooms")
		return
	}

	utils.SendSuccessWithMeta(c, rooms, gin.H{
		"count": len(rooms),
	})
}

// GetAreaRooms retrieves all rooms within a specific area
func (h *Handlers) GetAreaRooms(c *gin.Context) {
	areaIDStr := c.Param("id")
	areaID, err := strconv.Atoi(areaIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid area ID")
		return
	}

	includeEntities := c.Query("include_entities") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if includeEntities {
		rooms, err := h.repos.Room.GetRoomsWithEntities(ctx, &areaID)
		if err != nil {
			h.log.WithError(err).Error("Failed to get area rooms with entities")
			utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve rooms with entities")
			return
		}

		utils.SendSuccessWithMeta(c, rooms, gin.H{
			"area_id":          areaID,
			"include_entities": true,
			"room_count":       len(rooms),
		})
	} else {
		rooms, err := h.repos.Room.GetByAreaID(ctx, areaID)
		if err != nil {
			h.log.WithError(err).Error("Failed to get area rooms")
			utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve rooms")
			return
		}

		utils.SendSuccessWithMeta(c, rooms, gin.H{
			"area_id":    areaID,
			"room_count": len(rooms),
		})
	}
}
