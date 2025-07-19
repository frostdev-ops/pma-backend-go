package registries

import (
	"fmt"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// Custom errors for conflict resolver
var (
	ErrNoEntitiesProvided       = fmt.Errorf("no entities provided for conflict resolution")
	ErrConflictResolutionFailed = fmt.Errorf("conflict resolution failed")
	ErrNoValidEntity            = fmt.Errorf("no valid entity found after conflict resolution")
)

// DefaultConflictResolver implements the ConflictResolver interface
type DefaultConflictResolver struct {
	priorityManager types.SourcePriorityManager
	logger          *logrus.Logger
}

// NewDefaultConflictResolver creates a new conflict resolver
func NewDefaultConflictResolver(priorityManager types.SourcePriorityManager, logger *logrus.Logger) *DefaultConflictResolver {
	return &DefaultConflictResolver{
		priorityManager: priorityManager,
		logger:          logger,
	}
}

// ResolveEntityConflict resolves conflicts between multiple entities representing the same logical entity
func (r *DefaultConflictResolver) ResolveEntityConflict(entities []types.PMAEntity) (types.PMAEntity, error) {
	if len(entities) == 0 {
		return nil, ErrNoEntitiesProvided
	}

	if len(entities) == 1 {
		return entities[0], nil
	}

	r.logger.Debugf("Resolving conflict between %d entities", len(entities))

	// Filter out unavailable entities
	availableEntities := make([]types.PMAEntity, 0)
	for _, entity := range entities {
		if entity.IsAvailable() {
			availableEntities = append(availableEntities, entity)
		}
	}

	// If no available entities, use all entities
	if len(availableEntities) == 0 {
		availableEntities = entities
	}

	// Find the entity with the highest priority source
	var winningEntity types.PMAEntity
	highestPriority := 1000 // Start with a very low priority

	for _, entity := range availableEntities {
		sourcePriority := r.priorityManager.GetSourcePriority(entity.GetSource())

		// Lower number = higher priority
		if sourcePriority < highestPriority {
			highestPriority = sourcePriority
			winningEntity = entity
		} else if sourcePriority == highestPriority && winningEntity != nil {
			// If same priority, prefer the one with higher quality score
			if entity.GetQualityScore() > winningEntity.GetQualityScore() {
				winningEntity = entity
			} else if entity.GetQualityScore() == winningEntity.GetQualityScore() {
				// If same quality, prefer the more recently updated
				if entity.GetLastUpdated().After(winningEntity.GetLastUpdated()) {
					winningEntity = entity
				}
			}
		}
	}

	if winningEntity == nil {
		return nil, ErrNoValidEntity
	}

	r.logger.Debugf("Conflict resolved: selected entity from source %s (priority %d)",
		winningEntity.GetSource(), highestPriority)

	return winningEntity, nil
}

// ResolvePrioritySource determines which source should take priority for a given entity type
func (r *DefaultConflictResolver) ResolvePrioritySource(sources []types.PMASourceType, entityType types.PMAEntityType) types.PMASourceType {
	if len(sources) == 0 {
		return ""
	}

	if len(sources) == 1 {
		return sources[0]
	}

	// Apply entity-type specific logic
	switch entityType {
	case types.EntityTypeCamera:
		// Prefer Ring for cameras
		for _, source := range sources {
			if source == types.SourceRing {
				return source
			}
		}
	case types.EntityTypeSensor:
		// Prefer specialized sensors over generic ones
		priorityOrder := []types.PMASourceType{
			types.SourceShelly,        // Specialized sensor devices
			types.SourceUPS,           // Power sensors
			types.SourceHomeAssistant, // General sensors
			types.SourceNetwork,       // Network sensors
		}
		for _, preferredSource := range priorityOrder {
			for _, availableSource := range sources {
				if availableSource == preferredSource {
					return preferredSource
				}
			}
		}
	}

	// Fall back to general priority
	highestPrioritySource := sources[0]
	highestPriority := r.priorityManager.GetSourcePriority(sources[0])

	for _, source := range sources[1:] {
		priority := r.priorityManager.GetSourcePriority(source)
		if priority < highestPriority { // Lower number = higher priority
			highestPriority = priority
			highestPrioritySource = source
		}
	}

	return highestPrioritySource
}

// MergeEntityAttributes merges attributes from multiple entities, prioritizing by source
func (r *DefaultConflictResolver) MergeEntityAttributes(entities []types.PMAEntity) map[string]interface{} {
	if len(entities) == 0 {
		return make(map[string]interface{})
	}

	if len(entities) == 1 {
		return entities[0].GetAttributes()
	}

	// Sort entities by source priority
	sortedEntities := make([]types.PMAEntity, len(entities))
	copy(sortedEntities, entities)

	// Simple bubble sort by priority (for small slices this is fine)
	for i := 0; i < len(sortedEntities)-1; i++ {
		for j := 0; j < len(sortedEntities)-i-1; j++ {
			priority1 := r.priorityManager.GetSourcePriority(sortedEntities[j].GetSource())
			priority2 := r.priorityManager.GetSourcePriority(sortedEntities[j+1].GetSource())
			if priority1 > priority2 { // Higher priority (lower number) should come first
				sortedEntities[j], sortedEntities[j+1] = sortedEntities[j+1], sortedEntities[j]
			}
		}
	}

	// Merge attributes, with higher priority sources overriding lower priority ones
	mergedAttributes := make(map[string]interface{})

	// Start with lowest priority and work up (so higher priority overrides)
	for i := len(sortedEntities) - 1; i >= 0; i-- {
		entity := sortedEntities[i]
		entityAttrs := entity.GetAttributes()

		for key, value := range entityAttrs {
			// Special handling for certain attribute types
			switch key {
			case "last_updated", "last_changed":
				// Always use the most recent timestamp
				if existingValue, exists := mergedAttributes[key]; exists {
					if existingTime, ok := existingValue.(time.Time); ok {
						if newTime, ok := value.(time.Time); ok && newTime.After(existingTime) {
							mergedAttributes[key] = value
						}
					} else {
						mergedAttributes[key] = value
					}
				} else {
					mergedAttributes[key] = value
				}
			case "quality_score":
				// Use the highest quality score
				if existingValue, exists := mergedAttributes[key]; exists {
					if existingScore, ok := existingValue.(float64); ok {
						if newScore, ok := value.(float64); ok && newScore > existingScore {
							mergedAttributes[key] = value
						}
					} else {
						mergedAttributes[key] = value
					}
				} else {
					mergedAttributes[key] = value
				}
			default:
				// For other attributes, use source priority (higher priority overrides)
				mergedAttributes[key] = value
			}
		}
	}

	r.logger.Debugf("Merged attributes from %d entities into %d final attributes",
		len(entities), len(mergedAttributes))

	return mergedAttributes
}

// CreateVirtualEntity creates a virtual entity that combines data from multiple sources
func (r *DefaultConflictResolver) CreateVirtualEntity(entities []types.PMAEntity) (types.PMAEntity, error) {
	if len(entities) == 0 {
		return nil, ErrNoEntitiesProvided
	}

	if len(entities) == 1 {
		return entities[0], nil
	}

	// Get the primary entity (highest priority source)
	primaryEntity, err := r.ResolveEntityConflict(entities)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to determine primary entity", err)
	}

	// Create a virtual entity based on the primary entity
	virtualEntity := &types.PMABaseEntity{
		ID:           primaryEntity.GetID(),
		Type:         primaryEntity.GetType(),
		FriendlyName: primaryEntity.GetFriendlyName(),
		Icon:         primaryEntity.GetIcon(),
		State:        primaryEntity.GetState(),
		Capabilities: primaryEntity.GetCapabilities(),
		RoomID:       primaryEntity.GetRoomID(),
		AreaID:       primaryEntity.GetAreaID(),
		DeviceID:     primaryEntity.GetDeviceID(),
		Available:    primaryEntity.IsAvailable(),
		LastUpdated:  time.Now(),
	}

	// Merge attributes from all entities
	virtualEntity.Attributes = r.MergeEntityAttributes(entities)

	// Create virtual metadata
	sources := make([]types.PMASourceType, len(entities))
	totalQuality := 0.0
	for i, entity := range entities {
		sources[i] = entity.GetSource()
		totalQuality += entity.GetQualityScore()
	}

	virtualEntity.Metadata = &types.PMAMetadata{
		Source:         types.SourcePMA, // Mark as virtual/PMA-generated
		SourceEntityID: primaryEntity.GetID(),
		LastSynced:     time.Now(),
		QualityScore:   totalQuality / float64(len(entities)), // Average quality
		IsVirtual:      true,
		VirtualSources: sources,
		SourceData: map[string]interface{}{
			"primary_source": primaryEntity.GetSource(),
			"entity_count":   len(entities),
			"created_at":     time.Now(),
		},
	}

	// Use the highest quality score among all entities
	highestQuality := 0.0
	for _, entity := range entities {
		if entity.GetQualityScore() > highestQuality {
			highestQuality = entity.GetQualityScore()
		}
	}
	if highestQuality > virtualEntity.Metadata.QualityScore {
		virtualEntity.Metadata.QualityScore = highestQuality
	}

	r.logger.Infof("Created virtual entity %s from %d sources (primary: %s, quality: %.2f)",
		virtualEntity.ID, len(entities), primaryEntity.GetSource(), virtualEntity.Metadata.QualityScore)

	return virtualEntity, nil
}

// GetConflictResolutionStrategy returns information about how conflicts are resolved
func (r *DefaultConflictResolver) GetConflictResolutionStrategy() map[string]interface{} {
	return map[string]interface{}{
		"strategy":       "source_priority_with_quality_fallback",
		"priority_order": r.priorityManager.GetPriorityOrder(),
		"fallback_rules": []string{
			"prefer_available_entities",
			"prefer_higher_quality_score",
			"prefer_more_recent_updates",
		},
		"entity_specific_rules": map[string]string{
			"camera": "prefer_ring_source",
			"sensor": "prefer_specialized_sensors",
		},
	}
}
