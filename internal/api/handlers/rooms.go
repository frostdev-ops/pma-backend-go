package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RoomWithEntities represents a PMA room with its entities
type RoomWithEntities struct {
	*types.PMARoom
	Entities     []types.PMAEntity `json:"entities,omitempty"`
	EntityCount  int               `json:"entity_count"`
	SourceCounts map[string]int    `json:"source_counts,omitempty"`
}

// GetRooms retrieves all rooms using the unified PMA service
func (h *Handlers) GetRooms(c *gin.Context) {
	includeEntities := c.Query("include_entities") == "true"
	source := c.Query("source")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get rooms from all sources or specific source
	rooms, err := h.getAllRoomsFromSources(ctx, source)
	if err != nil {
		h.log.WithError(err).Error("Failed to get all rooms from unified service")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve rooms")
		return
	}

	// Convert to response format with optional entities
	roomsWithEntities := make([]*RoomWithEntities, 0, len(rooms))
	for _, room := range rooms {
		roomWithEntities := &RoomWithEntities{
			PMARoom:     room,
			EntityCount: len(room.EntityIDs),
		}

		if includeEntities {
			// Get entities for this room
			options := unified.GetAllOptions{
				IncludeRoom: false, // Already have room info
				IncludeArea: false,
			}
			entitiesResult, err := h.unifiedService.GetByRoom(ctx, room.ID, options)
			if err != nil {
				h.log.WithError(err).Warnf("Failed to get entities for room %s", room.ID)
			} else {
				// Extract just the entities and count by source
				entities := make([]types.PMAEntity, len(entitiesResult))
				sourceCounts := make(map[string]int)
				for i, entityWithRoom := range entitiesResult {
					entities[i] = entityWithRoom.Entity
					source := string(entityWithRoom.Entity.GetSource())
					sourceCounts[source]++
				}
				roomWithEntities.Entities = entities
				roomWithEntities.SourceCounts = sourceCounts
			}
		}

		roomsWithEntities = append(roomsWithEntities, roomWithEntities)
	}

	// Calculate metadata
	totalEntities := 0
	sourceCounts := make(map[string]int)
	for _, room := range roomsWithEntities {
		totalEntities += room.EntityCount
		roomSource := string(room.GetSource())
		sourceCounts[roomSource]++
	}

	utils.SendSuccessWithMeta(c, roomsWithEntities, gin.H{
		"count":            len(roomsWithEntities),
		"include_entities": includeEntities,
		"total_entities":   totalEntities,
		"by_source":        sourceCounts,
	})
}

// GetRoom retrieves a specific room using the unified PMA service
func (h *Handlers) GetRoom(c *gin.Context) {
	roomID := c.Param("id")
	includeEntities := c.Query("include_entities") == "true"

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get room from registry (this would need to be implemented)
	room, err := h.getRoomByID(ctx, roomID)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to get room: %s", roomID)
		utils.SendError(c, http.StatusNotFound, "Room not found")
		return
	}

	roomWithEntities := &RoomWithEntities{
		PMARoom:     room,
		EntityCount: len(room.EntityIDs),
	}

	if includeEntities {
		// Get entities for this room
		options := unified.GetAllOptions{
			IncludeRoom: false, // Already have room info
			IncludeArea: false,
		}
		entitiesResult, err := h.unifiedService.GetByRoom(ctx, roomID, options)
		if err != nil {
			h.log.WithError(err).Warnf("Failed to get entities for room %s", roomID)
		} else {
			// Extract just the entities and count by source
			entities := make([]types.PMAEntity, len(entitiesResult))
			sourceCounts := make(map[string]int)
			for i, entityWithRoom := range entitiesResult {
				entities[i] = entityWithRoom.Entity
				source := string(entityWithRoom.Entity.GetSource())
				sourceCounts[source]++
			}
			roomWithEntities.Entities = entities
			roomWithEntities.SourceCounts = sourceCounts
		}
	}

	utils.SendSuccess(c, roomWithEntities)
}

// CreateRoom creates a new PMA room (note: may need to be routed through an adapter)
func (h *Handlers) CreateRoom(c *gin.Context) {
	var request struct {
		Name        string `json:"name" binding:"required"`
		Icon        string `json:"icon"`
		Description string `json:"description"`
		Source      string `json:"source"` // Which source to create the room in
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// For now, create a PMA-native room
	// In the future, this could route to the appropriate adapter
	room := h.typeRegistry.CreateRoom(generateRoomID(), request.Name)
	room.Icon = request.Icon
	room.Description = request.Description

	// Store the room (this would need room registry implementation)
	if err := h.storeRoom(ctx, room); err != nil {
		h.log.WithError(err).Errorf("Failed to create room: %s", request.Name)
		utils.SendError(c, http.StatusInternalServerError, "Failed to create room")
		return
	}

	h.log.WithField("room_id", room.ID).WithField("room_name", room.Name).Info("Room created")

	// Broadcast WebSocket event for room creation
	if h.wsHub != nil {
		message := websocket.RoomUpdatedMessage(0, room.Name, "created") // Legacy format
		h.wsHub.BroadcastToAll(message.Type, message.Data)
	}

	utils.SendSuccess(c, gin.H{
		"message": "Room created successfully",
		"room_id": room.ID,
		"name":    room.Name,
	})
}

// UpdateRoom updates a room (may need to route through appropriate adapter)
func (h *Handlers) UpdateRoom(c *gin.Context) {
	roomID := c.Param("id")

	var request struct {
		Name        string `json:"name" binding:"required"`
		Icon        string `json:"icon"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		utils.SendError(c, http.StatusBadRequest, "Invalid request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get existing room
	room, err := h.getRoomByID(ctx, roomID)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to get room for update: %s", roomID)
		utils.SendError(c, http.StatusNotFound, "Room not found")
		return
	}

	// Update room properties
	room.Name = request.Name
	room.Icon = request.Icon
	room.Description = request.Description
	room.UpdatedAt = time.Now()

	// Store updated room
	if err := h.updateRoom(ctx, room); err != nil {
		h.log.WithError(err).Errorf("Failed to update room: %s", roomID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to update room")
		return
	}

	h.log.WithField("room_id", roomID).WithField("room_name", room.Name).Info("Room updated")

	// Broadcast WebSocket event for room update
	if h.wsHub != nil {
		message := websocket.RoomUpdatedMessage(0, room.Name, "updated") // Legacy format
		h.wsHub.BroadcastToAll(message.Type, message.Data)
	}

	utils.SendSuccess(c, gin.H{
		"message": "Room updated successfully",
		"room_id": roomID,
	})
}

// DeleteRoom deletes a room (may need to route through appropriate adapter)
func (h *Handlers) DeleteRoom(c *gin.Context) {
	roomID := c.Param("id")

	var request struct {
		ReassignToRoomID string `json:"reassign_to_room_id"`
	}

	// This is optional, so we don't use binding:"required"
	if err := c.ShouldBindJSON(&request); err != nil {
		// If JSON is invalid, we'll proceed without reassignment
		request.ReassignToRoomID = ""
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get room before deletion for WebSocket message
	room, err := h.getRoomByID(ctx, roomID)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to get room for deletion: %s", roomID)
		utils.SendError(c, http.StatusNotFound, "Room not found")
		return
	}

	// Reassign entities if requested
	if request.ReassignToRoomID != "" {
		if err := h.reassignRoomEntities(ctx, roomID, request.ReassignToRoomID); err != nil {
			h.log.WithError(err).Errorf("Failed to reassign entities from room %s", roomID)
			utils.SendError(c, http.StatusInternalServerError, "Failed to reassign entities")
			return
		}
	}

	// Delete room
	if err := h.deleteRoom(ctx, roomID); err != nil {
		h.log.WithError(err).Errorf("Failed to delete room: %s", roomID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to delete room")
		return
	}

	h.log.WithField("room_id", roomID).WithField("room_name", room.Name).Info("Room deleted")

	// Broadcast WebSocket event for room deletion
	if h.wsHub != nil {
		message := websocket.RoomUpdatedMessage(0, room.Name, "deleted") // Legacy format
		h.wsHub.BroadcastToAll(message.Type, message.Data)
	}

	utils.SendSuccess(c, gin.H{
		"message": "Room deleted successfully",
		"room_id": roomID,
	})
}

// GetRoomStats returns room statistics from the unified system
func (h *Handlers) GetRoomStats(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get all rooms and calculate stats
	rooms, err := h.getAllRoomsFromSources(ctx, "")
	if err != nil {
		h.log.WithError(err).Error("Failed to get rooms for stats")
		utils.SendError(c, http.StatusInternalServerError, "Failed to retrieve room statistics")
		return
	}

	// Calculate statistics
	totalRooms := len(rooms)
	totalEntities := 0
	roomsBySource := make(map[string]int)
	entitiesBySource := make(map[string]int)
	largestRoom := ""
	maxEntities := 0

	for _, room := range rooms {
		entityCount := len(room.EntityIDs)
		totalEntities += entityCount

		source := string(room.GetSource())
		roomsBySource[source]++
		entitiesBySource[source] += entityCount

		if entityCount > maxEntities {
			maxEntities = entityCount
			largestRoom = room.Name
		}
	}

	var averageEntitiesPerRoom float64
	if totalRooms > 0 {
		averageEntitiesPerRoom = float64(totalEntities) / float64(totalRooms)
	}

	stats := gin.H{
		"total_rooms":               totalRooms,
		"total_entities":            totalEntities,
		"average_entities_per_room": averageEntitiesPerRoom,
		"largest_room":              largestRoom,
		"max_entities_in_room":      maxEntities,
		"rooms_by_source":           roomsBySource,
		"entities_by_source":        entitiesBySource,
	}

	utils.SendSuccess(c, stats)
}

// SyncRoomsFromSources synchronizes rooms from all sources
func (h *Handlers) SyncRoomsFromSources(c *gin.Context) {
	sourceStr := c.Query("source")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if sourceStr != "" {
		// Sync rooms from specific source
		source := types.PMASourceType(sourceStr)
		result, err := h.syncRoomsFromSource(ctx, source)
		if err != nil {
			h.log.WithError(err).Errorf("Failed to sync rooms from source: %s", source)
			utils.SendError(c, http.StatusInternalServerError, "Failed to sync rooms")
			return
		}

		utils.SendSuccess(c, result)
	} else {
		// Sync rooms from all sources
		results, err := h.syncRoomsFromAllSources(ctx)
		if err != nil {
			h.log.WithError(err).Error("Failed to sync rooms from all sources")
			utils.SendError(c, http.StatusInternalServerError, "Failed to sync rooms")
			return
		}

		utils.SendSuccess(c, results)
	}
}

// AssignEntityToRoom assigns an entity to a PMA room
func (h *Handlers) AssignEntityToRoom(c *gin.Context) {
	entityID := c.Param("entity_id")
	roomID := c.Param("room_id")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the entity to verify it exists
	_, err := h.entityRegistry.GetEntity(entityID)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to get entity for room assignment: %s", entityID)
		utils.SendError(c, http.StatusNotFound, "Entity not found")
		return
	}

	// Get the room
	room, err := h.getRoomByID(ctx, roomID)
	if err != nil {
		h.log.WithError(err).Errorf("Failed to get room for entity assignment: %s", roomID)
		utils.SendError(c, http.StatusNotFound, "Room not found")
		return
	}

	// Update entity room assignment (this would need proper implementation)
	// For now, we'll update through the entity registry
	if err := h.assignEntityToRoom(ctx, entityID, roomID); err != nil {
		h.log.WithError(err).Errorf("Failed to assign entity %s to room %s", entityID, roomID)
		utils.SendError(c, http.StatusInternalServerError, "Failed to assign entity to room")
		return
	}

	utils.SendSuccess(c, gin.H{
		"message":   "Entity assigned to room successfully",
		"entity_id": entityID,
		"room_id":   roomID,
		"room_name": room.Name,
	})
}

// Helper methods (these would need full implementation)

func (h *Handlers) getAllRoomsFromSources(ctx context.Context, source string) ([]*types.PMARoom, error) {
	// Get rooms from the room service
	return h.roomService.GetAllRooms(ctx)
}

func (h *Handlers) getRoomByID(ctx context.Context, roomID string) (*types.PMARoom, error) {
	// Get room from room service
	return h.roomService.GetRoomByID(ctx, roomID)
}

func (h *Handlers) storeRoom(ctx context.Context, room *types.PMARoom) error {
	// Store room in room service
	return h.roomService.CreateRoom(ctx, room)
}

func (h *Handlers) updateRoom(ctx context.Context, room *types.PMARoom) error {
	// Update room in room service
	return h.roomService.UpdateRoom(ctx, room)
}

func (h *Handlers) deleteRoom(ctx context.Context, roomID string) error {
	// Delete room from room service
	return h.roomService.DeleteRoom(ctx, roomID)
}

func (h *Handlers) reassignRoomEntities(ctx context.Context, fromRoomID, toRoomID string) error {
	// Reassign entities between rooms
	return h.roomService.ReassignEntities(ctx, fromRoomID, toRoomID)
}

func (h *Handlers) assignEntityToRoom(ctx context.Context, entityID, roomID string) error {
	// Assign entity to room
	return h.roomService.AssignEntityToRoom(ctx, entityID, roomID)
}

func (h *Handlers) syncRoomsFromSource(ctx context.Context, source types.PMASourceType) (interface{}, error) {
	// Get all registered adapters
	adapters := h.adapterRegistry.GetAllAdapters()

	// Find adapter matching the source
	for _, adapter := range adapters {
		if adapter.GetSourceType() == source {
			err := h.roomService.SyncRoomsFromSource(ctx, adapter)
			if err != nil {
				return nil, fmt.Errorf("failed to sync rooms from %s: %w", source, err)
			}

			// Get room stats for response
			stats := h.roomService.GetRoomStats()
			return gin.H{
				"message": fmt.Sprintf("Rooms synchronized from %s", source),
				"source":  source,
				"stats":   stats,
			}, nil
		}
	}

	return nil, fmt.Errorf("adapter not found for source: %s", source)
}

func (h *Handlers) syncRoomsFromAllSources(ctx context.Context) (interface{}, error) {
	// Get all registered adapters
	adapters := h.adapterRegistry.GetAllAdapters()

	var errors []string
	syncedCount := 0

	for _, adapter := range adapters {
		err := h.roomService.SyncRoomsFromSource(ctx, adapter)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", adapter.GetSourceType(), err))
		} else {
			syncedCount++
		}
	}

	// Get room stats for response
	stats := h.roomService.GetRoomStats()

	response := gin.H{
		"message":        "Room synchronization completed",
		"sources_total":  len(adapters),
		"sources_synced": syncedCount,
		"stats":          stats,
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	return response, nil
}

func generateRoomID() string {
	// Generate a unique room ID
	// For now, use timestamp-based ID
	return "pma_room_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

// SyncRoomsWithHA synchronizes rooms with Home Assistant areas
func (h *Handlers) SyncRoomsWithHA(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Get Home Assistant adapter
	adapter, err := h.unifiedService.GetRegistryManager().GetAdapterRegistry().GetAdapterBySource(types.SourceHomeAssistant)
	if err != nil {
		h.log.WithError(err).Error("Home Assistant adapter not found")
		utils.SendError(c, http.StatusServiceUnavailable, "Home Assistant adapter not available")
		return
	}

	// Check if adapter is connected
	if !adapter.IsConnected() {
		h.log.Error("Home Assistant adapter not connected")
		utils.SendError(c, http.StatusServiceUnavailable, "Home Assistant adapter not connected")
		return
	}

	startTime := time.Now()
	h.log.Info("Starting Home Assistant room synchronization")

	// Sync rooms from Home Assistant
	haRooms, err := adapter.SyncRooms(ctx)
	if err != nil {
		h.log.WithError(err).Error("Failed to sync rooms from Home Assistant")
		utils.SendError(c, http.StatusInternalServerError, fmt.Sprintf("Failed to sync rooms: %v", err))
		return
	}

	// Process and store the synced rooms
	syncedCount := 0
	updatedCount := 0
	errors := []string{}

	for _, haRoom := range haRooms {
		// Check if room already exists
		existingRoom, err := h.getRoomByID(ctx, haRoom.ID)
		if err != nil {
			// Room doesn't exist, create it
			if err := h.storeRoom(ctx, haRoom); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to create room %s: %v", haRoom.Name, err))
				continue
			}
			syncedCount++
			h.log.WithField("room_name", haRoom.Name).Debug("Created new room from Home Assistant")
		} else {
			// Room exists, update it if needed
			if h.shouldUpdateRoom(existingRoom, haRoom) {
				haRoom.CreatedAt = existingRoom.CreatedAt // Preserve creation time
				haRoom.UpdatedAt = time.Now()
				if err := h.updateRoom(ctx, haRoom); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to update room %s: %v", haRoom.Name, err))
					continue
				}
				updatedCount++
				h.log.WithField("room_name", haRoom.Name).Debug("Updated room from Home Assistant")
			}
		}
	}

	duration := time.Since(startTime)

	// Prepare response
	response := gin.H{
		"success":       true,
		"message":       "Home Assistant room synchronization completed",
		"rooms_found":   len(haRooms),
		"rooms_synced":  syncedCount,
		"rooms_updated": updatedCount,
		"duration":      duration.String(),
		"processed_at":  time.Now(),
	}

	if len(errors) > 0 {
		response["errors"] = errors
		response["error_count"] = len(errors)
		h.log.WithField("error_count", len(errors)).Warn("Room sync completed with errors")
	}

	h.log.WithFields(logrus.Fields{
		"rooms_found":   len(haRooms),
		"rooms_synced":  syncedCount,
		"rooms_updated": updatedCount,
		"duration":      duration,
		"errors":        len(errors),
	}).Info("Home Assistant room synchronization completed")

	// Broadcast WebSocket event for room sync
	if h.wsHub != nil {
		message := websocket.Message{
			Type: "room_sync_completed",
			Data: map[string]interface{}{
				"source":        "homeassistant",
				"rooms_synced":  syncedCount,
				"rooms_updated": updatedCount,
				"duration":      duration.String(),
			},
		}
		h.wsHub.BroadcastToAll(message.Type, message.Data)
	}

	utils.SendSuccess(c, response)
}

// shouldUpdateRoom determines if a room should be updated based on changes
func (h *Handlers) shouldUpdateRoom(existing, new *types.PMARoom) bool {
	// Update if name, description, icon, or entity assignments have changed
	if existing.Name != new.Name ||
		existing.Description != new.Description ||
		existing.Icon != new.Icon {
		return true
	}

	// Check if entity assignments have changed
	if len(existing.EntityIDs) != len(new.EntityIDs) {
		return true
	}

	// Check individual entity IDs
	existingEntities := make(map[string]bool)
	for _, id := range existing.EntityIDs {
		existingEntities[id] = true
	}

	for _, id := range new.EntityIDs {
		if !existingEntities[id] {
			return true
		}
	}

	return false
}
