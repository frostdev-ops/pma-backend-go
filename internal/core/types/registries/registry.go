package registries

import (
	"fmt"
	"sync"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// RegistryManager manages all registry instances
type RegistryManager struct {
	AdapterRegistry  types.AdapterRegistry
	EntityRegistry   types.EntityRegistry
	ConflictResolver types.ConflictResolver
	PriorityManager  types.SourcePriorityManager
	logger           *logrus.Logger
	mutex            sync.RWMutex
}

// NewRegistryManager creates a new registry manager with all registries initialized
func NewRegistryManager(logger *logrus.Logger) *RegistryManager {
	// Create priority manager first since conflict resolver depends on it
	priorityManager := NewDefaultSourcePriorityManager(logger)

	// Create conflict resolver with priority manager
	conflictResolver := NewDefaultConflictResolver(priorityManager, logger)

	// Create other registries
	adapterRegistry := NewDefaultAdapterRegistry(logger)
	entityRegistry := NewDefaultEntityRegistry(logger)

	return &RegistryManager{
		AdapterRegistry:  adapterRegistry,
		EntityRegistry:   entityRegistry,
		ConflictResolver: conflictResolver,
		PriorityManager:  priorityManager,
		logger:           logger,
	}
}

// GetAdapterRegistry returns the adapter registry
func (rm *RegistryManager) GetAdapterRegistry() types.AdapterRegistry {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return rm.AdapterRegistry
}

// GetEntityRegistry returns the entity registry
func (rm *RegistryManager) GetEntityRegistry() types.EntityRegistry {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return rm.EntityRegistry
}

// GetConflictResolver returns the conflict resolver
func (rm *RegistryManager) GetConflictResolver() types.ConflictResolver {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return rm.ConflictResolver
}

// GetPriorityManager returns the source priority manager
func (rm *RegistryManager) GetPriorityManager() types.SourcePriorityManager {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	return rm.PriorityManager
}

// RegisterEntityWithConflictResolution registers an entity and resolves conflicts if they exist
func (rm *RegistryManager) RegisterEntityWithConflictResolution(entity types.PMAEntity) error {
	entityID := entity.GetID()

	// Check if entity already exists
	existingEntity, err := rm.EntityRegistry.GetEntity(entityID)
	if err == nil {
		// Entity exists, resolve conflict
		entities := []types.PMAEntity{existingEntity, entity}
		resolvedEntity, err := rm.ConflictResolver.ResolveEntityConflict(entities)
		if err != nil {
			return err
		}

		// Update with resolved entity
		return rm.EntityRegistry.UpdateEntity(resolvedEntity)
	}

	// Entity doesn't exist, register normally
	return rm.EntityRegistry.RegisterEntity(entity)
}

// SyncEntitiesFromAdapter synchronizes entities from an adapter and handles conflicts
func (rm *RegistryManager) SyncEntitiesFromAdapter(adapterID string) error {
	adapter, err := rm.AdapterRegistry.GetAdapter(adapterID)
	if err != nil {
		return err
	}

	// Get existing entities from this source
	existingEntities, err := rm.EntityRegistry.GetEntitiesBySource(adapter.GetSourceType())
	if err != nil {
		return err
	}

	// Create a map of existing entities for quick lookup
	existingMap := make(map[string]types.PMAEntity)
	for _, entity := range existingEntities {
		existingMap[entity.GetID()] = entity
	}

	// Sync entities from adapter
	newEntities, err := adapter.SyncEntities(nil)
	if err != nil {
		return err
	}

	// Process each new entity
	for _, newEntity := range newEntities {
		err := rm.RegisterEntityWithConflictResolution(newEntity)
		if err != nil {
			rm.logger.Errorf("Failed to register entity %s: %v", newEntity.GetID(), err)
			continue
		}

		// Remove from existing map (entities not in this sync will be considered stale)
		delete(existingMap, newEntity.GetID())
	}

	// Remove stale entities (entities that existed before but weren't in the sync)
	for entityID := range existingMap {
		err := rm.EntityRegistry.UnregisterEntity(entityID)
		if err != nil {
			rm.logger.Errorf("Failed to unregister stale entity %s: %v", entityID, err)
		} else {
			rm.logger.Debugf("Removed stale entity: %s", entityID)
		}
	}

	rm.logger.Infof("Synchronized %d entities from adapter %s", len(newEntities), adapterID)
	return nil
}

// GetAllRegistryStats returns comprehensive statistics about all registries
func (rm *RegistryManager) GetAllRegistryStats() map[string]interface{} {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	stats := make(map[string]interface{})

	// Get adapter registry stats
	if adapterReg, ok := rm.AdapterRegistry.(*DefaultAdapterRegistry); ok {
		stats["adapters"] = adapterReg.GetRegistryStats()
	}

	// Get entity registry stats
	if entityReg, ok := rm.EntityRegistry.(*DefaultEntityRegistry); ok {
		stats["entities"] = entityReg.GetRegistryStats()
	}

	// Get priority manager stats
	if priorityMgr, ok := rm.PriorityManager.(*DefaultSourcePriorityManager); ok {
		stats["priorities"] = priorityMgr.GetAllPriorities()
	}

	// Get conflict resolver strategy
	if conflictRes, ok := rm.ConflictResolver.(*DefaultConflictResolver); ok {
		stats["conflict_resolution"] = conflictRes.GetConflictResolutionStrategy()
	}

	return stats
}

// ValidateRegistryConsistency checks for consistency issues across registries
func (rm *RegistryManager) ValidateRegistryConsistency() []string {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()

	var issues []string

	// Get all entities
	allEntities, err := rm.EntityRegistry.GetAllEntities()
	if err != nil {
		issues = append(issues, "Failed to get all entities for validation")
		return issues
	}

	// Check if adapters exist for all entity sources
	for _, entity := range allEntities {
		source := entity.GetSource()
		if source == types.SourcePMA {
			continue // Virtual entities don't need adapters
		}

		_, err := rm.AdapterRegistry.GetAdapterBySource(source)
		if err != nil {
			issues = append(issues,
				fmt.Sprintf("Entity %s references source %s but no adapter is registered for this source",
					entity.GetID(), source))
		}
	}

	// Check for duplicate entity IDs across sources (potential conflicts)
	entityIDs := make(map[string][]types.PMASourceType)
	for _, entity := range allEntities {
		entityID := entity.GetID()
		source := entity.GetSource()
		entityIDs[entityID] = append(entityIDs[entityID], source)
	}

	for entityID, sources := range entityIDs {
		if len(sources) > 1 {
			issues = append(issues,
				fmt.Sprintf("Entity ID %s exists in multiple sources: %v (potential conflict)",
					entityID, sources))
		}
	}

	return issues
}

// Factory functions for creating individual registries

// NewAdapterRegistry creates a new default adapter registry
func NewAdapterRegistry(logger *logrus.Logger) types.AdapterRegistry {
	return NewDefaultAdapterRegistry(logger)
}

// NewEntityRegistry creates a new default entity registry
func NewEntityRegistry(logger *logrus.Logger) types.EntityRegistry {
	return NewDefaultEntityRegistry(logger)
}

// NewSourcePriorityManager creates a new default source priority manager
func NewSourcePriorityManager(logger *logrus.Logger) types.SourcePriorityManager {
	return NewDefaultSourcePriorityManager(logger)
}

// NewConflictResolver creates a new default conflict resolver
func NewConflictResolver(priorityManager types.SourcePriorityManager, logger *logrus.Logger) types.ConflictResolver {
	return NewDefaultConflictResolver(priorityManager, logger)
}
