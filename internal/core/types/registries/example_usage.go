package registries

import (
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// ExampleUsage demonstrates how to use the registry infrastructure
func ExampleUsage() {
	logger := logrus.New()

	// 1. Create the registry manager (coordinates all registries)
	registryManager := NewRegistryManager(logger)

	// 2. Create a sample entity
	sampleEntity := &types.PMABaseEntity{
		ID:           "light.living_room",
		Type:         types.EntityTypeLight,
		FriendlyName: "Living Room Light",
		Icon:         "mdi:lightbulb",
		State:        types.StateOn,
		Attributes: map[string]interface{}{
			"brightness": 80,
			"color":      "#FFFFFF",
		},
		LastUpdated:  time.Now(),
		Capabilities: []types.PMACapability{types.CapabilityDimmable, types.CapabilityColorable},
		Available:    true,
		Metadata: &types.PMAMetadata{
			Source:         types.SourceHomeAssistant,
			SourceEntityID: "light.living_room",
			LastSynced:     time.Now(),
			QualityScore:   0.95,
			IsVirtual:      false,
		},
	}

	// 3. Register entity with automatic conflict resolution
	err := registryManager.RegisterEntityWithConflictResolution(sampleEntity)
	if err != nil {
		logger.Errorf("Failed to register entity: %v", err)
		return
	}

	// 4. Query entities by different criteria

	// Get all entities
	allEntities, err := registryManager.GetEntityRegistry().GetAllEntities()
	if err != nil {
		logger.Errorf("Failed to get all entities: %v", err)
		return
	}
	logger.Infof("Total entities: %d", len(allEntities))

	// Get entities by type
	lightEntities, err := registryManager.GetEntityRegistry().GetEntitiesByType(types.EntityTypeLight)
	if err != nil {
		logger.Errorf("Failed to get light entities: %v", err)
		return
	}
	logger.Infof("Light entities: %d", len(lightEntities))

	// Get entities by source
	haEntities, err := registryManager.GetEntityRegistry().GetEntitiesBySource(types.SourceHomeAssistant)
	if err != nil {
		logger.Errorf("Failed to get HA entities: %v", err)
		return
	}
	logger.Infof("HomeAssistant entities: %d", len(haEntities))

	// Search entities
	searchResults, err := registryManager.GetEntityRegistry().SearchEntities("living")
	if err != nil {
		logger.Errorf("Failed to search entities: %v", err)
		return
	}
	logger.Infof("Search results for 'living': %d", len(searchResults))

	// 5. Work with source priorities
	priorityManager := registryManager.GetPriorityManager()

	// Get current priority order
	priorityOrder := priorityManager.GetPriorityOrder()
	logger.Infof("Priority order: %v", priorityOrder)

	// Check if one source should override another
	shouldOverride := priorityManager.ShouldOverride(types.SourceRing, types.SourceHomeAssistant)
	logger.Infof("Should HA override Ring? %v", shouldOverride)

	// 6. Demonstrate conflict resolution
	conflictResolver := registryManager.GetConflictResolver()

	// Create a conflicting entity from a different source
	conflictingEntity := &types.PMABaseEntity{
		ID:           "light.living_room", // Same ID as above
		Type:         types.EntityTypeLight,
		FriendlyName: "Living Room Light (Ring)",
		Icon:         "mdi:lightbulb",
		State:        types.StateOff,
		Attributes: map[string]interface{}{
			"brightness": 0,
			"motion":     false,
		},
		LastUpdated:  time.Now(),
		Capabilities: []types.PMACapability{types.CapabilityDimmable, types.CapabilityMotion},
		Available:    true,
		Metadata: &types.PMAMetadata{
			Source:         types.SourceRing,
			SourceEntityID: "light.living_room",
			LastSynced:     time.Now(),
			QualityScore:   0.85,
			IsVirtual:      false,
		},
	}

	// Resolve conflict between the two entities
	entities := []types.PMAEntity{sampleEntity, conflictingEntity}
	resolvedEntity, err := conflictResolver.ResolveEntityConflict(entities)
	if err != nil {
		logger.Errorf("Failed to resolve conflict: %v", err)
		return
	}
	logger.Infof("Conflict resolved: selected entity from source %s", resolvedEntity.GetSource())

	// Merge attributes from both entities
	mergedAttributes := conflictResolver.MergeEntityAttributes(entities)
	logger.Infof("Merged attributes: %v", mergedAttributes)

	// Create a virtual entity combining both sources
	virtualEntity, err := conflictResolver.CreateVirtualEntity(entities)
	if err != nil {
		logger.Errorf("Failed to create virtual entity: %v", err)
		return
	}
	logger.Infof("Created virtual entity with quality score: %.2f", virtualEntity.GetQualityScore())

	// 7. Get comprehensive statistics
	stats := registryManager.GetAllRegistryStats()
	logger.Infof("Registry statistics: %+v", stats)

	// 8. Validate registry consistency
	issues := registryManager.ValidateRegistryConsistency()
	if len(issues) > 0 {
		logger.Warnf("Found %d consistency issues:", len(issues))
		for _, issue := range issues {
			logger.Warn(issue)
		}
	} else {
		logger.Info("Registry consistency check passed")
	}

	logger.Info("Registry usage example completed successfully")
}

// ExampleAdapterIntegration shows how adapters would integrate with registries
func ExampleAdapterIntegration() {
	logger := logrus.New()
	registryManager := NewRegistryManager(logger)

	// Note: This is pseudo-code showing the pattern, not actual working code
	// since we don't have real adapters implemented yet

	/*
		// 1. Create and register adapters
		haAdapter := homeassistant.NewAdapter(haConfig)
		ringAdapter := ring.NewAdapter(ringConfig)
		shellyAdapter := shelly.NewAdapter(shellyConfig)

		adapterRegistry := registryManager.GetAdapterRegistry()
		adapterRegistry.RegisterAdapter(haAdapter)
		adapterRegistry.RegisterAdapter(ringAdapter)
		adapterRegistry.RegisterAdapter(shellyAdapter)

		// 2. Sync entities from all connected adapters
		connectedAdapters := adapterRegistry.GetConnectedAdapters()
		for _, adapter := range connectedAdapters {
			err := registryManager.SyncEntitiesFromAdapter(adapter.GetID())
			if err != nil {
				logger.Errorf("Sync failed for adapter %s: %v", adapter.GetID(), err)
				continue
			}
			logger.Infof("Successfully synced entities from %s", adapter.GetID())
		}

		// 3. Monitor adapter health
		for _, adapter := range adapterRegistry.GetAllAdapters() {
			health := adapter.GetHealth()
			if !health.IsHealthy {
				logger.Warnf("Adapter %s is unhealthy: %v", adapter.GetID(), health.Issues)
			}

			metrics, err := adapterRegistry.GetAdapterMetrics(adapter.GetID())
			if err == nil {
				logger.Infof("Adapter %s manages %d entities",
					adapter.GetID(), metrics.EntitiesManaged)
			}
		}
	*/

	// For now, just show that we can get the registry stats
	stats := registryManager.GetAllRegistryStats()
	logger.Infof("Registry manager initialized with stats: %+v", stats)
	logger.Info("Adapter integration example outlined")
}
