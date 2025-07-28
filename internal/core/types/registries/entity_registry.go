package registries

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// Custom errors for entity registry
var (
	ErrEntityNotFound          = fmt.Errorf("entity not found")
	ErrEntityAlreadyRegistered = fmt.Errorf("entity already registered")
	ErrInvalidEntity           = fmt.Errorf("invalid entity")
	ErrRegistryFull            = fmt.Errorf("entity registry is full")
)

// DefaultEntityRegistry implements the EntityRegistry interface
type DefaultEntityRegistry struct {
	entities         map[string]types.PMAEntity       // entityID -> entity
	entitiesByType   map[types.PMAEntityType][]string // entityType -> []entityID
	entitiesBySource map[types.PMASourceType][]string // sourceType -> []entityID
	entitiesByRoom   map[string][]string              // roomID -> []entityID
	entityTimestamps map[string]time.Time             // entityID -> last access time
	mutex            sync.RWMutex
	logger           *logrus.Logger

	// Memory management
	maxEntities     int           // Maximum number of entities to keep in memory
	cleanupInterval time.Duration // How often to run cleanup
	lastCleanup     time.Time     // Last cleanup time
}

// NewDefaultEntityRegistry creates a new entity registry
func NewDefaultEntityRegistry(logger *logrus.Logger) *DefaultEntityRegistry {
	return &DefaultEntityRegistry{
		entities:         make(map[string]types.PMAEntity),
		entitiesByType:   make(map[types.PMAEntityType][]string),
		entitiesBySource: make(map[types.PMASourceType][]string),
		entitiesByRoom:   make(map[string][]string),
		entityTimestamps: make(map[string]time.Time),
		logger:           logger,
		maxEntities:      500,             // Reduced from 1000 to prevent memory leaks
		cleanupInterval:  1 * time.Minute, // More frequent cleanup
		lastCleanup:      time.Now(),
	}
}

// RegisterEntity registers a new entity in the registry
func (r *DefaultEntityRegistry) RegisterEntity(entity types.PMAEntity) error {
	if entity == nil {
		return ErrInvalidEntity
	}

	entityID := entity.GetID()
	if entityID == "" {
		return fmt.Errorf("%w: entity ID cannot be empty", ErrInvalidEntity)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// Check if we need to cleanup before adding new entities
	// Trigger cleanup more frequently to prevent memory leaks
	if len(r.entities) >= r.maxEntities || time.Since(r.lastCleanup) > r.cleanupInterval {
		r.cleanupOldEntities()
	}

	// Check if entity already exists
	if existingEntity, exists := r.entities[entityID]; exists {
		// If it's the same entity (same source), update it
		if existingEntity.GetSource() == entity.GetSource() {
			return r.updateEntityInternal(entity)
		}
		return fmt.Errorf("%w: entity ID '%s'", ErrEntityAlreadyRegistered, entityID)
	}

	// Check if registry is still full after cleanup
	if len(r.entities) >= r.maxEntities {
		return fmt.Errorf("%w: cannot register entity '%s', registry is full", ErrRegistryFull, entityID)
	}

	// Register the entity
	r.entities[entityID] = entity
	r.entityTimestamps[entityID] = time.Now()

	// Add to type index
	entityType := entity.GetType()
	r.entitiesByType[entityType] = append(r.entitiesByType[entityType], entityID)

	// Add to source index
	sourceType := entity.GetSource()
	r.entitiesBySource[sourceType] = append(r.entitiesBySource[sourceType], entityID)

	// Add to room index if entity has a room
	if roomID := entity.GetRoomID(); roomID != nil && *roomID != "" {
		r.entitiesByRoom[*roomID] = append(r.entitiesByRoom[*roomID], entityID)
	}

	r.logger.WithField("entity_id", entityID).Debug("Entity registered successfully")
	return nil
}

// UnregisterEntity removes an entity from the registry
func (r *DefaultEntityRegistry) UnregisterEntity(entityID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	entity, exists := r.entities[entityID]
	if !exists {
		return fmt.Errorf("%w: entity ID '%s'", ErrEntityNotFound, entityID)
	}

	// Remove from main map
	delete(r.entities, entityID)
	delete(r.entityTimestamps, entityID) // Also delete timestamp

	// Remove from type index
	entityType := entity.GetType()
	r.entitiesByType[entityType] = removeFromSlice(r.entitiesByType[entityType], entityID)
	if len(r.entitiesByType[entityType]) == 0 {
		delete(r.entitiesByType, entityType)
	}

	// Remove from source index
	sourceType := entity.GetSource()
	r.entitiesBySource[sourceType] = removeFromSlice(r.entitiesBySource[sourceType], entityID)
	if len(r.entitiesBySource[sourceType]) == 0 {
		delete(r.entitiesBySource, sourceType)
	}

	// Remove from room index
	if roomID := entity.GetRoomID(); roomID != nil && *roomID != "" {
		r.entitiesByRoom[*roomID] = removeFromSlice(r.entitiesByRoom[*roomID], entityID)
		if len(r.entitiesByRoom[*roomID]) == 0 {
			delete(r.entitiesByRoom, *roomID)
		}
	}

	r.logger.Debugf("Unregistered entity: %s", entityID)

	return nil
}

// GetEntity retrieves an entity by its ID
func (r *DefaultEntityRegistry) GetEntity(entityID string) (types.PMAEntity, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	entity, exists := r.entities[entityID]
	if !exists {
		return nil, fmt.Errorf("%w: entity ID '%s'", ErrEntityNotFound, entityID)
	}

	// Update timestamp
	r.entityTimestamps[entityID] = time.Now()

	return entity, nil
}

// GetEntitiesByType retrieves all entities of a specific type
func (r *DefaultEntityRegistry) GetEntitiesByType(entityType types.PMAEntityType) ([]types.PMAEntity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entityIDs, exists := r.entitiesByType[entityType]
	if !exists {
		return []types.PMAEntity{}, nil
	}

	entities := make([]types.PMAEntity, 0, len(entityIDs))
	for _, entityID := range entityIDs {
		if entity, exists := r.entities[entityID]; exists {
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

// GetEntitiesBySource retrieves all entities from a specific source
func (r *DefaultEntityRegistry) GetEntitiesBySource(source types.PMASourceType) ([]types.PMAEntity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entityIDs, exists := r.entitiesBySource[source]
	if !exists {
		return []types.PMAEntity{}, nil
	}

	entities := make([]types.PMAEntity, 0, len(entityIDs))
	for _, entityID := range entityIDs {
		if entity, exists := r.entities[entityID]; exists {
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

// GetEntitiesByRoom retrieves all entities in a specific room
func (r *DefaultEntityRegistry) GetEntitiesByRoom(roomID string) ([]types.PMAEntity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entityIDs, exists := r.entitiesByRoom[roomID]
	if !exists {
		return []types.PMAEntity{}, nil
	}

	entities := make([]types.PMAEntity, 0, len(entityIDs))
	for _, entityID := range entityIDs {
		if entity, exists := r.entities[entityID]; exists {
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

// GetAllEntities retrieves all registered entities
func (r *DefaultEntityRegistry) GetAllEntities() ([]types.PMAEntity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entities := make([]types.PMAEntity, 0, len(r.entities))
	for _, entity := range r.entities {
		entities = append(entities, entity)
	}

	return entities, nil
}

// UpdateEntity updates an existing entity in the registry
func (r *DefaultEntityRegistry) UpdateEntity(entity types.PMAEntity) error {
	if entity == nil {
		return ErrInvalidEntity
	}

	entityID := entity.GetID()
	if entityID == "" {
		return fmt.Errorf("%w: entity ID cannot be empty", ErrInvalidEntity)
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	return r.updateEntityInternal(entity)
}

// updateEntityInternal updates an entity without acquiring locks (internal use)
func (r *DefaultEntityRegistry) updateEntityInternal(entity types.PMAEntity) error {
	entityID := entity.GetID()

	existingEntity, exists := r.entities[entityID]
	if !exists {
		return fmt.Errorf("%w: entity ID '%s'", ErrEntityNotFound, entityID)
	}

	// Check if room changed and update room index
	oldRoomID := existingEntity.GetRoomID()
	newRoomID := entity.GetRoomID()

	if (oldRoomID == nil && newRoomID != nil) ||
		(oldRoomID != nil && newRoomID == nil) ||
		(oldRoomID != nil && newRoomID != nil && *oldRoomID != *newRoomID) {

		// Remove from old room
		if oldRoomID != nil && *oldRoomID != "" {
			r.entitiesByRoom[*oldRoomID] = removeFromSlice(r.entitiesByRoom[*oldRoomID], entityID)
			if len(r.entitiesByRoom[*oldRoomID]) == 0 {
				delete(r.entitiesByRoom, *oldRoomID)
			}
		}

		// Add to new room
		if newRoomID != nil && *newRoomID != "" {
			r.entitiesByRoom[*newRoomID] = append(r.entitiesByRoom[*newRoomID], entityID)
		}
	}

	// Update the entity
	r.entities[entityID] = entity
	r.entityTimestamps[entityID] = time.Now() // Update timestamp

	r.logger.Debugf("Updated entity: %s", entityID)

	return nil
}

// SearchEntities searches for entities by name or ID
func (r *DefaultEntityRegistry) SearchEntities(query string) ([]types.PMAEntity, error) {
	if query == "" {
		return r.GetAllEntities()
	}

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	query = strings.ToLower(query)
	var matchingEntities []types.PMAEntity

	for _, entity := range r.entities {
		// Check if query matches entity ID or friendly name
		if strings.Contains(strings.ToLower(entity.GetID()), query) ||
			strings.Contains(strings.ToLower(entity.GetFriendlyName()), query) {
			matchingEntities = append(matchingEntities, entity)
		}
	}

	return matchingEntities, nil
}

// GetEntityCount returns the total number of registered entities
func (r *DefaultEntityRegistry) GetEntityCount() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.entities)
}

// GetEntityCountByType returns the count of entities grouped by type
func (r *DefaultEntityRegistry) GetEntityCountByType() map[types.PMAEntityType]int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	counts := make(map[types.PMAEntityType]int)
	for entityType, entityIDs := range r.entitiesByType {
		counts[entityType] = len(entityIDs)
	}

	return counts
}

// GetEntityCountBySource returns the count of entities grouped by source
func (r *DefaultEntityRegistry) GetEntityCountBySource() map[types.PMASourceType]int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	counts := make(map[types.PMASourceType]int)
	for sourceType, entityIDs := range r.entitiesBySource {
		counts[sourceType] = len(entityIDs)
	}

	return counts
}

// GetAvailableEntities returns only available entities
func (r *DefaultEntityRegistry) GetAvailableEntities() ([]types.PMAEntity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var availableEntities []types.PMAEntity
	for _, entity := range r.entities {
		if entity.IsAvailable() {
			availableEntities = append(availableEntities, entity)
		}
	}

	return availableEntities, nil
}

// GetRegistryStats returns statistics about the registry
func (r *DefaultEntityRegistry) GetRegistryStats() map[string]interface{} {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := map[string]interface{}{
		"total_entities":     len(r.entities),
		"max_entities":       r.maxEntities,
		"memory_usage_pct":   float64(len(r.entities)) / float64(r.maxEntities) * 100,
		"last_cleanup":       r.lastCleanup,
		"cleanup_interval":   r.cleanupInterval.String(),
		"entities_by_type":   r.GetEntityCountByType(),
		"entities_by_source": r.GetEntityCountBySource(),
	}

	return stats
}

// cleanupOldEntities removes entities that haven't been accessed for a long time
func (r *DefaultEntityRegistry) cleanupOldEntities() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	now := time.Now()
	entitiesToRemove := []string{}

	for entityID, lastAccess := range r.entityTimestamps {
		if now.Sub(lastAccess) > r.cleanupInterval {
			entitiesToRemove = append(entitiesToRemove, entityID)
		}
	}

	for _, entityID := range entitiesToRemove {
		r.UnregisterEntity(entityID)
	}

	r.lastCleanup = now
	r.logger.Debugf("Cleanup completed. Removed %d entities.", len(entitiesToRemove))
}

// Helper function to remove a string from a slice
func removeFromSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
