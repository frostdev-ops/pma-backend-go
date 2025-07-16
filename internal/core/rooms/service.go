package rooms

import (
	"context"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// Service handles room business logic
type Service struct {
	roomRepo   repositories.RoomRepository
	entityRepo repositories.EntityRepository
	logger     *logrus.Logger
}

// NewService creates a new room service
func NewService(roomRepo repositories.RoomRepository, entityRepo repositories.EntityRepository, logger *logrus.Logger) *Service {
	return &Service{
		roomRepo:   roomRepo,
		entityRepo: entityRepo,
		logger:     logger,
	}
}

// RoomWithEntities represents a room with its entities
type RoomWithEntities struct {
	*models.Room
	Entities    []*models.Entity `json:"entities,omitempty"`
	EntityCount int              `json:"entity_count"`
}

// RoomStats represents room statistics
type RoomStats struct {
	TotalRooms         int            `json:"total_rooms"`
	RoomsWithEntities  int            `json:"rooms_with_entities"`
	EmptyRooms         int            `json:"empty_rooms"`
	EntityDistribution map[string]int `json:"entity_distribution"` // domain -> count
}

// GetAll retrieves all rooms with optional entity information
func (s *Service) GetAll(ctx context.Context, includeEntities bool) ([]*RoomWithEntities, error) {
	rooms, err := s.roomRepo.GetAll(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get all rooms")
		return nil, fmt.Errorf("failed to get rooms: %w", err)
	}

	result := make([]*RoomWithEntities, len(rooms))
	for i, room := range rooms {
		roomWithEntities := &RoomWithEntities{Room: room}

		// Get entities for this room
		entities, err := s.entityRepo.GetByRoom(ctx, room.ID)
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to get entities for room %d", room.ID)
			roomWithEntities.EntityCount = 0
		} else {
			roomWithEntities.EntityCount = len(entities)
			if includeEntities {
				roomWithEntities.Entities = entities
			}
		}

		result[i] = roomWithEntities
	}

	return result, nil
}

// GetByID retrieves a room by ID with optional entity information
func (s *Service) GetByID(ctx context.Context, roomID int, includeEntities bool) (*RoomWithEntities, error) {
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get room: %d", roomID)
		return nil, fmt.Errorf("failed to get room: %w", err)
	}

	roomWithEntities := &RoomWithEntities{Room: room}

	// Get entities for this room
	entities, err := s.entityRepo.GetByRoom(ctx, roomID)
	if err != nil {
		s.logger.WithError(err).Warnf("Failed to get entities for room %d", roomID)
		roomWithEntities.EntityCount = 0
	} else {
		roomWithEntities.EntityCount = len(entities)
		if includeEntities {
			roomWithEntities.Entities = entities
		}
	}

	return roomWithEntities, nil
}

// Create creates a new room
func (s *Service) Create(ctx context.Context, room *models.Room) error {
	// Check if room with same name already exists
	existing, err := s.roomRepo.GetByName(ctx, room.Name)
	if err == nil && existing != nil {
		return fmt.Errorf("room with name '%s' already exists", room.Name)
	}

	room.CreatedAt = time.Now()
	room.UpdatedAt = time.Now()

	err = s.roomRepo.Create(ctx, room)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to create room: %s", room.Name)
		return fmt.Errorf("failed to create room: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"room_id":   room.ID,
		"room_name": room.Name,
	}).Info("Room created")

	return nil
}

// Update updates a room
func (s *Service) Update(ctx context.Context, roomID int, updates *models.Room) error {
	existing, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("room not found: %w", err)
	}

	// Check if name is being changed to an existing name
	if updates.Name != existing.Name {
		existingWithName, err := s.roomRepo.GetByName(ctx, updates.Name)
		if err == nil && existingWithName != nil && existingWithName.ID != roomID {
			return fmt.Errorf("room with name '%s' already exists", updates.Name)
		}
	}

	// Update fields
	existing.Name = updates.Name
	existing.Icon = updates.Icon
	existing.Description = updates.Description
	existing.HomeAssistantAreaID = updates.HomeAssistantAreaID
	existing.UpdatedAt = time.Now()

	err = s.roomRepo.Update(ctx, existing)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to update room: %d", roomID)
		return fmt.Errorf("failed to update room: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"room_id":   roomID,
		"room_name": existing.Name,
	}).Info("Room updated")

	return nil
}

// Delete deletes a room and optionally reassigns its entities
func (s *Service) Delete(ctx context.Context, roomID int, reassignToRoomID *int) error {
	// Check if room exists
	room, err := s.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return fmt.Errorf("room not found: %w", err)
	}

	// Get entities in this room
	entities, err := s.entityRepo.GetByRoom(ctx, roomID)
	if err != nil {
		s.logger.WithError(err).Warnf("Failed to get entities for room %d during deletion", roomID)
	} else if len(entities) > 0 {
		if reassignToRoomID != nil {
			// Verify target room exists
			_, err := s.roomRepo.GetByID(ctx, *reassignToRoomID)
			if err != nil {
				return fmt.Errorf("target room for reassignment not found: %w", err)
			}

			// Reassign all entities to the new room
			for _, entity := range entities {
				entity.RoomID.Int64 = int64(*reassignToRoomID)
				entity.RoomID.Valid = true
				err = s.entityRepo.Update(ctx, entity)
				if err != nil {
					s.logger.WithError(err).Errorf("Failed to reassign entity %s during room deletion", entity.EntityID)
					return fmt.Errorf("failed to reassign entities: %w", err)
				}
			}

			s.logger.WithFields(logrus.Fields{
				"room_id":        roomID,
				"target_room_id": *reassignToRoomID,
				"entities_moved": len(entities),
			}).Info("Entities reassigned during room deletion")
		} else {
			// Unassign all entities from the room
			for _, entity := range entities {
				entity.RoomID.Valid = false
				err = s.entityRepo.Update(ctx, entity)
				if err != nil {
					s.logger.WithError(err).Errorf("Failed to unassign entity %s during room deletion", entity.EntityID)
					return fmt.Errorf("failed to unassign entities: %w", err)
				}
			}

			s.logger.WithFields(logrus.Fields{
				"room_id":             roomID,
				"entities_unassigned": len(entities),
			}).Info("Entities unassigned during room deletion")
		}
	}

	// Delete the room
	err = s.roomRepo.Delete(ctx, roomID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to delete room: %d", roomID)
		return fmt.Errorf("failed to delete room: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"room_id":   roomID,
		"room_name": room.Name,
	}).Info("Room deleted")

	return nil
}

// GetStats returns room statistics
func (s *Service) GetStats(ctx context.Context) (*RoomStats, error) {
	rooms, err := s.GetAll(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get rooms for stats: %w", err)
	}

	stats := &RoomStats{
		TotalRooms:         len(rooms),
		EntityDistribution: make(map[string]int),
	}

	roomsWithEntities := 0
	for _, room := range rooms {
		if room.EntityCount > 0 {
			roomsWithEntities++
		}

		// Count entities by domain
		for _, entity := range room.Entities {
			stats.EntityDistribution[entity.Domain]++
		}
	}

	stats.RoomsWithEntities = roomsWithEntities
	stats.EmptyRooms = stats.TotalRooms - roomsWithEntities

	return stats, nil
}

// SyncWithHomeAssistant synchronizes room data with Home Assistant areas
func (s *Service) SyncWithHomeAssistant(ctx context.Context, haAreas []HAArea) error {
	s.logger.Info("Starting Home Assistant area synchronization")

	for _, area := range haAreas {
		// Check if room with this HA area ID already exists
		existing, err := s.roomRepo.GetByName(ctx, area.Name)
		if err == nil && existing != nil {
			// Update existing room
			if existing.HomeAssistantAreaID.String != area.ID {
				existing.HomeAssistantAreaID.String = area.ID
				existing.HomeAssistantAreaID.Valid = true
				existing.UpdatedAt = time.Now()

				err = s.roomRepo.Update(ctx, existing)
				if err != nil {
					s.logger.WithError(err).Errorf("Failed to update room with HA area ID: %s", area.Name)
					continue
				}

				s.logger.WithFields(logrus.Fields{
					"room_name":  area.Name,
					"ha_area_id": area.ID,
				}).Info("Room synchronized with HA area")
			}
		} else {
			// Create new room from HA area
			room := &models.Room{
				Name:      area.Name,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			room.HomeAssistantAreaID.String = area.ID
			room.HomeAssistantAreaID.Valid = true

			err = s.roomRepo.Create(ctx, room)
			if err != nil {
				s.logger.WithError(err).Errorf("Failed to create room from HA area: %s", area.Name)
				continue
			}

			s.logger.WithFields(logrus.Fields{
				"room_name":  area.Name,
				"ha_area_id": area.ID,
			}).Info("Room created from HA area")
		}
	}

	s.logger.Info("Home Assistant area synchronization completed")
	return nil
}

// HAArea represents a Home Assistant area
type HAArea struct {
	ID   string `json:"area_id"`
	Name string `json:"name"`
}
