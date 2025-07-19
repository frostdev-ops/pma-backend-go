package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/unified"
	"github.com/frostdev-ops/pma-backend-go/internal/websocket"
	"github.com/frostdev-ops/pma-backend-go/pkg/utils"
	"github.com/gin-gonic/gin"
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
	// This would iterate through all adapters and get their rooms
	// For now, return an empty slice
	// TODO: Implement room registry and room synchronization
	return []*types.PMARoom{}, nil
}

func (h *Handlers) getRoomByID(ctx context.Context, roomID string) (*types.PMARoom, error) {
	// This would get room from room registry
	// For now, return a placeholder
	return &types.PMARoom{
		ID:   roomID,
		Name: "Unknown Room",
	}, nil
}

func (h *Handlers) storeRoom(ctx context.Context, room *types.PMARoom) error {
	// This would store room in room registry
	// TODO: Implement room storage
	return nil
}

func (h *Handlers) updateRoom(ctx context.Context, room *types.PMARoom) error {
	// This would update room in room registry
	// TODO: Implement room update
	return nil
}

func (h *Handlers) deleteRoom(ctx context.Context, roomID string) error {
	// This would delete room from room registry
	// TODO: Implement room deletion
	return nil
}

func (h *Handlers) reassignRoomEntities(ctx context.Context, fromRoomID, toRoomID string) error {
	// This would reassign all entities from one room to another
	// TODO: Implement entity reassignment
	return nil
}

func (h *Handlers) assignEntityToRoom(ctx context.Context, entityID, roomID string) error {
	// This would assign an entity to a room
	// TODO: Implement entity-room assignment
	return nil
}

func (h *Handlers) syncRoomsFromSource(ctx context.Context, source types.PMASourceType) (interface{}, error) {
	// This would sync rooms from a specific adapter
	// TODO: Implement room sync from source
	return gin.H{"message": "Room sync not yet implemented"}, nil
}

func (h *Handlers) syncRoomsFromAllSources(ctx context.Context) (interface{}, error) {
	// This would sync rooms from all adapters
	// TODO: Implement room sync from all sources
	return gin.H{"message": "Room sync not yet implemented"}, nil
}

func generateRoomID() string {
	// Generate a unique room ID
	// For now, use timestamp-based ID
	return "pma_room_" + strconv.FormatInt(time.Now().UnixNano(), 10)
}

// SyncRoomsWithHA synchronizes rooms with Home Assistant areas (stub implementation)
func (h *Handlers) SyncRoomsWithHA(c *gin.Context) {
	// TODO: Implement Home Assistant room synchronization
	utils.SendError(c, http.StatusNotImplemented, "Home Assistant room sync not yet implemented")
}
