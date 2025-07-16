package entities

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/database/models"
	"github.com/frostdev-ops/pma-backend-go/internal/database/repositories"
	"github.com/sirupsen/logrus"
)

// Service handles entity business logic
type Service struct {
	entityRepo repositories.EntityRepository
	roomRepo   repositories.RoomRepository
	logger     *logrus.Logger
}

// NewService creates a new entity service
func NewService(entityRepo repositories.EntityRepository, roomRepo repositories.RoomRepository, logger *logrus.Logger) *Service {
	return &Service{
		entityRepo: entityRepo,
		roomRepo:   roomRepo,
		logger:     logger,
	}
}

// EntityWithRoom represents an entity with room information
type EntityWithRoom struct {
	*models.Entity
	Room *models.Room `json:"room,omitempty"`
}

// GetAll retrieves all entities with optional room information
func (s *Service) GetAll(ctx context.Context, includeRoom bool) ([]*EntityWithRoom, error) {
	entities, err := s.entityRepo.GetAll(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get all entities")
		return nil, fmt.Errorf("failed to get entities: %w", err)
	}

	result := make([]*EntityWithRoom, len(entities))
	for i, entity := range entities {
		entityWithRoom := &EntityWithRoom{Entity: entity}

		if includeRoom && entity.RoomID.Valid {
			room, err := s.roomRepo.GetByID(ctx, int(entity.RoomID.Int64))
			if err != nil {
				s.logger.WithError(err).Warnf("Failed to get room for entity %s", entity.EntityID)
			} else {
				entityWithRoom.Room = room
			}
		}

		result[i] = entityWithRoom
	}

	return result, nil
}

// GetByID retrieves an entity by ID with optional room information
func (s *Service) GetByID(ctx context.Context, entityID string, includeRoom bool) (*EntityWithRoom, error) {
	entity, err := s.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get entity: %s", entityID)
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	entityWithRoom := &EntityWithRoom{Entity: entity}

	if includeRoom && entity.RoomID.Valid {
		room, err := s.roomRepo.GetByID(ctx, int(entity.RoomID.Int64))
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to get room for entity %s", entityID)
		} else {
			entityWithRoom.Room = room
		}
	}

	return entityWithRoom, nil
}

// GetByDomain retrieves entities by domain
func (s *Service) GetByDomain(ctx context.Context, domain string) ([]*models.Entity, error) {
	// This would require adding GetByDomain to the repository interface
	// For now, get all and filter
	entities, err := s.entityRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get entities: %w", err)
	}

	var filtered []*models.Entity
	for _, entity := range entities {
		if entity.Domain == domain {
			filtered = append(filtered, entity)
		}
	}

	return filtered, nil
}

// UpdateState updates entity state and attributes
func (s *Service) UpdateState(ctx context.Context, entityID string, state string, attributes map[string]interface{}) error {
	entity, err := s.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		return fmt.Errorf("entity not found: %w", err)
	}

	// Update state
	entity.State.String = state
	entity.State.Valid = true
	entity.LastUpdated = time.Now()

	// Update attributes if provided
	if attributes != nil {
		attributesJSON, err := json.Marshal(attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal attributes: %w", err)
		}
		entity.Attributes = attributesJSON
	}

	err = s.entityRepo.Update(ctx, entity)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to update entity state: %s", entityID)
		return fmt.Errorf("failed to update entity: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"new_state": state,
	}).Info("Entity state updated")

	return nil
}

// AssignToRoom assigns an entity to a room
func (s *Service) AssignToRoom(ctx context.Context, entityID string, roomID *int) error {
	entity, err := s.entityRepo.GetByID(ctx, entityID)
	if err != nil {
		return fmt.Errorf("entity not found: %w", err)
	}

	if roomID != nil {
		// Verify room exists
		_, err := s.roomRepo.GetByID(ctx, *roomID)
		if err != nil {
			return fmt.Errorf("room not found: %w", err)
		}
		entity.RoomID.Int64 = int64(*roomID)
		entity.RoomID.Valid = true
	} else {
		entity.RoomID.Valid = false
	}

	err = s.entityRepo.Update(ctx, entity)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to assign entity to room: %s", entityID)
		return fmt.Errorf("failed to update entity: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"room_id":   roomID,
	}).Info("Entity assigned to room")

	return nil
}

// CreateOrUpdate creates a new entity or updates existing one
func (s *Service) CreateOrUpdate(ctx context.Context, entity *models.Entity) error {
	existing, err := s.entityRepo.GetByID(ctx, entity.EntityID)
	if err != nil {
		// Entity doesn't exist, create it
		entity.LastUpdated = time.Now()
		err = s.entityRepo.Create(ctx, entity)
		if err != nil {
			s.logger.WithError(err).Errorf("Failed to create entity: %s", entity.EntityID)
			return fmt.Errorf("failed to create entity: %w", err)
		}

		s.logger.WithField("entity_id", entity.EntityID).Info("Entity created")
		return nil
	}

	// Entity exists, update it
	existing.FriendlyName = entity.FriendlyName
	existing.State = entity.State
	existing.Attributes = entity.Attributes
	existing.LastUpdated = time.Now()
	// Keep existing room assignment unless specified
	if entity.RoomID.Valid {
		existing.RoomID = entity.RoomID
	}

	err = s.entityRepo.Update(ctx, existing)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to update entity: %s", entity.EntityID)
		return fmt.Errorf("failed to update entity: %w", err)
	}

	s.logger.WithField("entity_id", entity.EntityID).Info("Entity updated")
	return nil
}

// Delete removes an entity
func (s *Service) Delete(ctx context.Context, entityID string) error {
	err := s.entityRepo.Delete(ctx, entityID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to delete entity: %s", entityID)
		return fmt.Errorf("failed to delete entity: %w", err)
	}

	s.logger.WithField("entity_id", entityID).Info("Entity deleted")
	return nil
}
