package rooms

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// RoomService manages rooms and their entity assignments
type RoomService struct {
	rooms  map[string]*types.PMARoom
	logger *logrus.Logger
	mutex  sync.RWMutex
}

// NewRoomService creates a new room service
func NewRoomService(logger *logrus.Logger) *RoomService {
	return &RoomService{
		rooms:  make(map[string]*types.PMARoom),
		logger: logger,
	}
}

// GetAllRooms returns all rooms
func (s *RoomService) GetAllRooms(ctx context.Context) ([]*types.PMARoom, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	rooms := make([]*types.PMARoom, 0, len(s.rooms))
	for _, room := range s.rooms {
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// GetRoomByID returns a room by ID
func (s *RoomService) GetRoomByID(ctx context.Context, roomID string) (*types.PMARoom, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	room, exists := s.rooms[roomID]
	if !exists {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}

	return room, nil
}

// CreateRoom creates a new room
func (s *RoomService) CreateRoom(ctx context.Context, room *types.PMARoom) error {
	if room.ID == "" {
		return fmt.Errorf("room ID is required")
	}
	if room.Name == "" {
		return fmt.Errorf("room name is required")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if room already exists
	if _, exists := s.rooms[room.ID]; exists {
		return fmt.Errorf("room already exists: %s", room.ID)
	}

	// Set creation time
	room.CreatedAt = time.Now()
	room.UpdatedAt = time.Now()

	// Initialize entity list if nil
	if room.EntityIDs == nil {
		room.EntityIDs = make([]string, 0)
	}

	s.rooms[room.ID] = room

	s.logger.WithFields(logrus.Fields{
		"room_id":   room.ID,
		"room_name": room.Name,
	}).Info("Room created")

	return nil
}

// UpdateRoom updates an existing room
func (s *RoomService) UpdateRoom(ctx context.Context, room *types.PMARoom) error {
	if room.ID == "" {
		return fmt.Errorf("room ID is required")
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if room exists
	existingRoom, exists := s.rooms[room.ID]
	if !exists {
		return fmt.Errorf("room not found: %s", room.ID)
	}

	// Preserve creation time and update timestamp
	room.CreatedAt = existingRoom.CreatedAt
	room.UpdatedAt = time.Now()

	s.rooms[room.ID] = room

	s.logger.WithFields(logrus.Fields{
		"room_id":   room.ID,
		"room_name": room.Name,
	}).Info("Room updated")

	return nil
}

// DeleteRoom deletes a room
func (s *RoomService) DeleteRoom(ctx context.Context, roomID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if room exists
	if _, exists := s.rooms[roomID]; !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	delete(s.rooms, roomID)

	s.logger.WithField("room_id", roomID).Info("Room deleted")
	return nil
}

// AssignEntityToRoom assigns an entity to a room
func (s *RoomService) AssignEntityToRoom(ctx context.Context, entityID, roomID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if room exists
	room, exists := s.rooms[roomID]
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	// Check if entity is already in the room
	for _, existingEntityID := range room.EntityIDs {
		if existingEntityID == entityID {
			return nil // Already assigned
		}
	}

	// Add entity to room
	room.EntityIDs = append(room.EntityIDs, entityID)
	room.UpdatedAt = time.Now()

	s.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"room_id":   roomID,
	}).Info("Entity assigned to room")

	return nil
}

// UnassignEntityFromRoom removes an entity from a room
func (s *RoomService) UnassignEntityFromRoom(ctx context.Context, entityID, roomID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if room exists
	room, exists := s.rooms[roomID]
	if !exists {
		return fmt.Errorf("room not found: %s", roomID)
	}

	// Find and remove entity
	for i, existingEntityID := range room.EntityIDs {
		if existingEntityID == entityID {
			room.EntityIDs = append(room.EntityIDs[:i], room.EntityIDs[i+1:]...)
			room.UpdatedAt = time.Now()

			s.logger.WithFields(logrus.Fields{
				"entity_id": entityID,
				"room_id":   roomID,
			}).Info("Entity unassigned from room")

			return nil
		}
	}

	return fmt.Errorf("entity not found in room: %s", entityID)
}

// GetEntitiesInRoom returns all entities in a room
func (s *RoomService) GetEntitiesInRoom(ctx context.Context, roomID string) ([]string, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	room, exists := s.rooms[roomID]
	if !exists {
		return nil, fmt.Errorf("room not found: %s", roomID)
	}

	// Return a copy to prevent external modification
	entities := make([]string, len(room.EntityIDs))
	copy(entities, room.EntityIDs)

	return entities, nil
}

// ReassignEntities moves all entities from one room to another
func (s *RoomService) ReassignEntities(ctx context.Context, fromRoomID, toRoomID string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if both rooms exist
	fromRoom, exists := s.rooms[fromRoomID]
	if !exists {
		return fmt.Errorf("source room not found: %s", fromRoomID)
	}

	toRoom, exists := s.rooms[toRoomID]
	if !exists {
		return fmt.Errorf("destination room not found: %s", toRoomID)
	}

	// Move entities
	entityCount := len(fromRoom.EntityIDs)
	toRoom.EntityIDs = append(toRoom.EntityIDs, fromRoom.EntityIDs...)
	fromRoom.EntityIDs = make([]string, 0)

	// Update timestamps
	fromRoom.UpdatedAt = time.Now()
	toRoom.UpdatedAt = time.Now()

	s.logger.WithFields(logrus.Fields{
		"from_room_id": fromRoomID,
		"to_room_id":   toRoomID,
		"entity_count": entityCount,
	}).Info("Entities reassigned between rooms")

	return nil
}

// SyncRoomsFromSource synchronizes rooms from an adapter source
func (s *RoomService) SyncRoomsFromSource(ctx context.Context, adapter types.PMAAdapter) error {
	// Get rooms from adapter
	rooms, err := adapter.SyncRooms(ctx)
	if err != nil {
		return fmt.Errorf("failed to sync rooms from adapter %s: %w", adapter.GetID(), err)
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	syncCount := 0
	for _, room := range rooms {
		// Check if room already exists
		existingRoom, exists := s.rooms[room.ID]
		if exists {
			// Update existing room
			room.CreatedAt = existingRoom.CreatedAt
			room.UpdatedAt = time.Now()
		} else {
			// New room
			room.CreatedAt = time.Now()
			room.UpdatedAt = time.Now()
			syncCount++
		}

		s.rooms[room.ID] = room
	}

	s.logger.WithFields(logrus.Fields{
		"adapter_id":      adapter.GetID(),
		"rooms_synced":    len(rooms),
		"new_rooms_added": syncCount,
	}).Info("Rooms synchronized from adapter")

	return nil
}

// GetRoomStats returns statistics about rooms
func (s *RoomService) GetRoomStats() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	totalEntities := 0
	for _, room := range s.rooms {
		totalEntities += len(room.EntityIDs)
	}

	return map[string]interface{}{
		"total_rooms":    len(s.rooms),
		"total_entities": totalEntities,
		"avg_entities_per_room": func() float64 {
			if len(s.rooms) == 0 {
				return 0
			}
			return float64(totalEntities) / float64(len(s.rooms))
		}(),
	}
}
