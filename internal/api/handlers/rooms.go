package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/rooms"
	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
)

// GetRooms retrieves all rooms
func (h *Handlers) GetRooms(c *gin.Context) {
	includeEntities := c.Query("include_entities") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	roomService := rooms.NewService(h.repos.Room, h.repos.Entity, h.logger)

	roomsWithEntities, err := roomService.GetAll(ctx, includeEntities)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get all rooms")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve rooms")
		return
	}

	utils.SendSuccessWithMeta(c, roomsWithEntities, gin.H{
		"count":            len(roomsWithEntities),
		"include_entities": includeEntities,
	})
}

// GetRoom retrieves a specific room
func (h *Handlers) GetRoom(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	includeEntities := c.Query("include_entities") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	roomService := rooms.NewService(h.repos.Room, h.repos.Entity, h.logger)

	roomWithEntities, err := roomService.GetByID(ctx, roomID, includeEntities)
	if err != nil {
		h.logger.WithError(err).Errorf("Failed to get room: %d", roomID)
		utils.SendError(c, http.StatusNotFound, "Room not found")
		return
	}

	utils.SendSuccess(c, roomWithEntities)
}

// CreateRoom creates a new room
func (h *Handlers) CreateRoom(c *gin.Context) {
	var request struct {
		Name                string  `json:"name" binding:"required"`
		Icon                *string `json:"icon"`
		Description         *string `json:"description"`
		HomeAssistantAreaID *string `json:"home_assistant_area_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	room := &models.Room{
		Name: request.Name,
	}

	if request.Icon != nil {
		room.Icon.String = *request.Icon
		room.Icon.Valid = true
	}

	if request.Description != nil {
		room.Description.String = *request.Description
		room.Description.Valid = true
	}

	if request.HomeAssistantAreaID != nil {
		room.HomeAssistantAreaID.String = *request.HomeAssistantAreaID
		room.HomeAssistantAreaID.Valid = true
	}

	roomService := rooms.NewService(h.repos.Room, h.repos.Entity, h.logger)

	err := roomService.Create(ctx, room)
	if err != nil {
		h.logger.WithError(err).Errorf("Failed to create room: %s", request.Name)
		utils.SendError(c, http.StatusInternalServerError, "Failed to create room")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "Room created successfully",
		"room_id": room.ID,
		"name":    room.Name,
	})
}

// UpdateRoom updates a room
func (h *Handlers) UpdateRoom(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var request struct {
		Name                string  `json:"name" binding:"required"`
		Icon                *string `json:"icon"`
		Description         *string `json:"description"`
		HomeAssistantAreaID *string `json:"home_assistant_area_id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	updates := &models.Room{
		Name: request.Name,
	}

	if request.Icon != nil {
		updates.Icon.String = *request.Icon
		updates.Icon.Valid = true
	}

	if request.Description != nil {
		updates.Description.String = *request.Description
		updates.Description.Valid = true
	}

	if request.HomeAssistantAreaID != nil {
		updates.HomeAssistantAreaID.String = *request.HomeAssistantAreaID
		updates.HomeAssistantAreaID.Valid = true
	}

	roomService := rooms.NewService(h.repos.Room, h.repos.Entity, h.logger)

	err = roomService.Update(ctx, roomID, updates)
	if err != nil {
		h.logger.WithError(err).Errorf("Failed to update room: %d", roomID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to update room")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "Room updated successfully",
		"room_id": roomID,
	})
}

// DeleteRoom deletes a room
func (h *Handlers) DeleteRoom(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid room ID")
		return
	}

	var request struct {
		ReassignToRoomID *int `json:"reassign_to_room_id"`
	}

	// This is optional, so we don't use binding:"required"
	if err := c.ShouldBindJSON(&request); err != nil {
		// If JSON is invalid, we'll proceed without reassignment
		request.ReassignToRoomID = nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	roomService := rooms.NewService(h.repos.Room, h.repos.Entity, h.logger)

	err = roomService.Delete(ctx, roomID, request.ReassignToRoomID)
	if err != nil {
		h.logger.WithError(err).Errorf("Failed to delete room: %d", roomID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to delete room")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message": "Room deleted successfully",
		"room_id": roomID,
	})
}

// GetRoomStats returns room statistics
func (h *Handlers) GetRoomStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	roomService := rooms.NewService(h.repos.Room, h.repos.Entity, h.logger)

	stats, err := roomService.GetStats(ctx)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get room stats")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve room statistics")
		return
	}

	utils.SendSuccess(c, stats)
}

// SyncRoomsWithHA synchronizes rooms with Home Assistant areas
func (h *Handlers) SyncRoomsWithHA(c *gin.Context) {
	var request struct {
		Areas []rooms.HAArea `json:"areas" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	roomService := rooms.NewService(h.repos.Room, h.repos.Entity, h.logger)

	err := roomService.SyncWithHomeAssistant(ctx, request.Areas)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sync rooms with Home Assistant")
		utils.SendError(c, http.StatusInternalServerError, "Failed to synchronize with Home Assistant")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":     "Rooms synchronized with Home Assistant successfully",
		"areas_count": len(request.Areas),
	})
}
