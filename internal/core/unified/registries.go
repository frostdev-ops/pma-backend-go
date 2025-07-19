package unified

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// ConcreteAdapterRegistry implements the AdapterRegistry interface
type ConcreteAdapterRegistry struct {
	adapters map[string]types.PMAAdapter
	bySource map[types.PMASourceType]types.PMAAdapter
	logger   *logrus.Logger
	mutex    sync.RWMutex
}

// NewConcreteAdapterRegistry creates a new adapter registry
func NewConcreteAdapterRegistry(logger *logrus.Logger) *ConcreteAdapterRegistry {
	return &ConcreteAdapterRegistry{
		adapters: make(map[string]types.PMAAdapter),
		bySource: make(map[types.PMASourceType]types.PMAAdapter),
		logger:   logger,
	}
}

func (r *ConcreteAdapterRegistry) RegisterAdapter(adapter types.PMAAdapter) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.adapters[adapter.GetID()]; exists {
		return fmt.Errorf("adapter with ID %s already registered", adapter.GetID())
	}

	if _, exists := r.bySource[adapter.GetSourceType()]; exists {
		return fmt.Errorf("adapter for source %s already registered", adapter.GetSourceType())
	}

	r.adapters[adapter.GetID()] = adapter
	r.bySource[adapter.GetSourceType()] = adapter

	r.logger.WithFields(logrus.Fields{
		"adapter_id": adapter.GetID(),
		"source":     adapter.GetSourceType(),
		"name":       adapter.GetName(),
	}).Info("Adapter registered")

	return nil
}

func (r *ConcreteAdapterRegistry) UnregisterAdapter(adapterID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	adapter, exists := r.adapters[adapterID]
	if !exists {
		return fmt.Errorf("adapter %s not found", adapterID)
	}

	delete(r.adapters, adapterID)
	delete(r.bySource, adapter.GetSourceType())

	r.logger.WithField("adapter_id", adapterID).Info("Adapter unregistered")
	return nil
}

func (r *ConcreteAdapterRegistry) GetAdapter(adapterID string) (types.PMAAdapter, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	adapter, exists := r.adapters[adapterID]
	if !exists {
		return nil, fmt.Errorf("adapter %s not found", adapterID)
	}

	return adapter, nil
}

func (r *ConcreteAdapterRegistry) GetAdapterBySource(sourceType types.PMASourceType) (types.PMAAdapter, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	adapter, exists := r.bySource[sourceType]
	if !exists {
		return nil, fmt.Errorf("no adapter found for source %s", sourceType)
	}

	return adapter, nil
}

func (r *ConcreteAdapterRegistry) GetAllAdapters() []types.PMAAdapter {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	adapters := make([]types.PMAAdapter, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		adapters = append(adapters, adapter)
	}

	return adapters
}

func (r *ConcreteAdapterRegistry) GetConnectedAdapters() []types.PMAAdapter {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var connected []types.PMAAdapter
	for _, adapter := range r.adapters {
		if adapter.IsConnected() {
			connected = append(connected, adapter)
		}
	}

	return connected
}

func (r *ConcreteAdapterRegistry) GetAdapterMetrics(adapterID string) (*types.AdapterMetrics, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	adapter, exists := r.adapters[adapterID]
	if !exists {
		return nil, fmt.Errorf("adapter %s not found", adapterID)
	}

	return adapter.GetMetrics(), nil
}

// ConcreteEntityRegistry implements the EntityRegistry interface
type ConcreteEntityRegistry struct {
	entities map[string]types.PMAEntity
	byType   map[types.PMAEntityType][]string
	bySource map[types.PMASourceType][]string
	byRoom   map[string][]string
	logger   *logrus.Logger
	mutex    sync.RWMutex
}

// NewConcreteEntityRegistry creates a new entity registry
func NewConcreteEntityRegistry(logger *logrus.Logger) *ConcreteEntityRegistry {
	return &ConcreteEntityRegistry{
		entities: make(map[string]types.PMAEntity),
		byType:   make(map[types.PMAEntityType][]string),
		bySource: make(map[types.PMASourceType][]string),
		byRoom:   make(map[string][]string),
		logger:   logger,
	}
}

func (r *ConcreteEntityRegistry) RegisterEntity(entity types.PMAEntity) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	entityID := entity.GetID()
	if _, exists := r.entities[entityID]; exists {
		return fmt.Errorf("entity with ID %s already registered", entityID)
	}

	r.entities[entityID] = entity

	// Update indexes
	r.addToTypeIndex(entity.GetType(), entityID)
	r.addToSourceIndex(entity.GetSource(), entityID)
	if roomID := entity.GetRoomID(); roomID != nil {
		r.addToRoomIndex(*roomID, entityID)
	}

	r.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"type":      entity.GetType(),
		"source":    entity.GetSource(),
	}).Debug("Entity registered")

	return nil
}

func (r *ConcreteEntityRegistry) UnregisterEntity(entityID string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	entity, exists := r.entities[entityID]
	if !exists {
		return fmt.Errorf("entity %s not found", entityID)
	}

	// Remove from indexes
	r.removeFromTypeIndex(entity.GetType(), entityID)
	r.removeFromSourceIndex(entity.GetSource(), entityID)
	if roomID := entity.GetRoomID(); roomID != nil {
		r.removeFromRoomIndex(*roomID, entityID)
	}

	delete(r.entities, entityID)

	r.logger.WithField("entity_id", entityID).Debug("Entity unregistered")
	return nil
}

func (r *ConcreteEntityRegistry) GetEntity(entityID string) (types.PMAEntity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entity, exists := r.entities[entityID]
	if !exists {
		return nil, fmt.Errorf("entity %s not found", entityID)
	}

	return entity, nil
}

func (r *ConcreteEntityRegistry) GetEntitiesByType(entityType types.PMAEntityType) ([]types.PMAEntity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entityIDs, exists := r.byType[entityType]
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

func (r *ConcreteEntityRegistry) GetEntitiesBySource(source types.PMASourceType) ([]types.PMAEntity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entityIDs, exists := r.bySource[source]
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

func (r *ConcreteEntityRegistry) GetEntitiesByRoom(roomID string) ([]types.PMAEntity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entityIDs, exists := r.byRoom[roomID]
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

func (r *ConcreteEntityRegistry) GetAllEntities() ([]types.PMAEntity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	entities := make([]types.PMAEntity, 0, len(r.entities))
	for _, entity := range r.entities {
		entities = append(entities, entity)
	}

	return entities, nil
}

func (r *ConcreteEntityRegistry) UpdateEntity(entity types.PMAEntity) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	entityID := entity.GetID()
	existingEntity, exists := r.entities[entityID]
	if !exists {
		return fmt.Errorf("entity %s not found", entityID)
	}

	// Update indexes if entity properties changed
	if existingEntity.GetType() != entity.GetType() {
		r.removeFromTypeIndex(existingEntity.GetType(), entityID)
		r.addToTypeIndex(entity.GetType(), entityID)
	}

	if existingEntity.GetSource() != entity.GetSource() {
		r.removeFromSourceIndex(existingEntity.GetSource(), entityID)
		r.addToSourceIndex(entity.GetSource(), entityID)
	}

	// Handle room changes
	oldRoomID := existingEntity.GetRoomID()
	newRoomID := entity.GetRoomID()
	if (oldRoomID == nil) != (newRoomID == nil) || (oldRoomID != nil && newRoomID != nil && *oldRoomID != *newRoomID) {
		if oldRoomID != nil {
			r.removeFromRoomIndex(*oldRoomID, entityID)
		}
		if newRoomID != nil {
			r.addToRoomIndex(*newRoomID, entityID)
		}
	}

	r.entities[entityID] = entity

	r.logger.WithField("entity_id", entityID).Debug("Entity updated")
	return nil
}

func (r *ConcreteEntityRegistry) SearchEntities(query string) ([]types.PMAEntity, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return r.GetAllEntities()
	}

	var matches []types.PMAEntity
	for _, entity := range r.entities {
		// Search in entity ID, friendly name, and type
		if r.entityMatches(entity, query) {
			matches = append(matches, entity)
		}
	}

	return matches, nil
}

// Helper methods for index management

func (r *ConcreteEntityRegistry) addToTypeIndex(entityType types.PMAEntityType, entityID string) {
	if r.byType[entityType] == nil {
		r.byType[entityType] = make([]string, 0)
	}
	r.byType[entityType] = append(r.byType[entityType], entityID)
}

func (r *ConcreteEntityRegistry) removeFromTypeIndex(entityType types.PMAEntityType, entityID string) {
	if list, exists := r.byType[entityType]; exists {
		for i, id := range list {
			if id == entityID {
				r.byType[entityType] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}
}

func (r *ConcreteEntityRegistry) addToSourceIndex(source types.PMASourceType, entityID string) {
	if r.bySource[source] == nil {
		r.bySource[source] = make([]string, 0)
	}
	r.bySource[source] = append(r.bySource[source], entityID)
}

func (r *ConcreteEntityRegistry) removeFromSourceIndex(source types.PMASourceType, entityID string) {
	if list, exists := r.bySource[source]; exists {
		for i, id := range list {
			if id == entityID {
				r.bySource[source] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}
}

func (r *ConcreteEntityRegistry) addToRoomIndex(roomID string, entityID string) {
	if r.byRoom[roomID] == nil {
		r.byRoom[roomID] = make([]string, 0)
	}
	r.byRoom[roomID] = append(r.byRoom[roomID], entityID)
}

func (r *ConcreteEntityRegistry) removeFromRoomIndex(roomID string, entityID string) {
	if list, exists := r.byRoom[roomID]; exists {
		for i, id := range list {
			if id == entityID {
				r.byRoom[roomID] = append(list[:i], list[i+1:]...)
				break
			}
		}
	}
}

func (r *ConcreteEntityRegistry) entityMatches(entity types.PMAEntity, query string) bool {
	// Search in entity ID
	if strings.Contains(strings.ToLower(entity.GetID()), query) {
		return true
	}

	// Search in friendly name
	if strings.Contains(strings.ToLower(entity.GetFriendlyName()), query) {
		return true
	}

	// Search in entity type
	if strings.Contains(strings.ToLower(string(entity.GetType())), query) {
		return true
	}

	// Search in attributes
	for key, value := range entity.GetAttributes() {
		if strings.Contains(strings.ToLower(key), query) {
			return true
		}
		if valueStr, ok := value.(string); ok && strings.Contains(strings.ToLower(valueStr), query) {
			return true
		}
	}

	return false
}

// ConcreteConflictResolver implements the ConflictResolver interface
type ConcreteConflictResolver struct {
	priorityManager types.SourcePriorityManager
	logger          *logrus.Logger
}

// NewConcreteConflictResolver creates a new conflict resolver
func NewConcreteConflictResolver(priorityManager types.SourcePriorityManager, logger *logrus.Logger) *ConcreteConflictResolver {
	return &ConcreteConflictResolver{
		priorityManager: priorityManager,
		logger:          logger,
	}
}

func (r *ConcreteConflictResolver) ResolveEntityConflict(entities []types.PMAEntity) (types.PMAEntity, error) {
	if len(entities) == 0 {
		return nil, fmt.Errorf("no entities provided for conflict resolution")
	}

	if len(entities) == 1 {
		return entities[0], nil
	}

	// Sort entities by priority and quality
	sort.Slice(entities, func(i, j int) bool {
		// First, sort by source priority
		priorityI := r.priorityManager.GetSourcePriority(entities[i].GetSource())
		priorityJ := r.priorityManager.GetSourcePriority(entities[j].GetSource())
		if priorityI != priorityJ {
			return priorityI > priorityJ // Higher priority first
		}

		// Then by quality score
		qualityI := entities[i].GetQualityScore()
		qualityJ := entities[j].GetQualityScore()
		if qualityI != qualityJ {
			return qualityI > qualityJ // Higher quality first
		}

		// Finally by last updated timestamp
		return entities[i].GetLastUpdated().After(entities[j].GetLastUpdated())
	})

	winningEntity := entities[0]

	r.logger.WithFields(logrus.Fields{
		"winning_source":  winningEntity.GetSource(),
		"entity_id":       winningEntity.GetID(),
		"total_conflicts": len(entities),
	}).Debug("Resolved entity conflict")

	return winningEntity, nil
}

func (r *ConcreteConflictResolver) ResolvePrioritySource(sources []types.PMASourceType, entityType types.PMAEntityType) types.PMASourceType {
	if len(sources) == 0 {
		return types.SourcePMA
	}

	if len(sources) == 1 {
		return sources[0]
	}

	// Find the source with the highest priority
	highestPriority := -1
	var winningSource types.PMASourceType

	for _, source := range sources {
		priority := r.priorityManager.GetSourcePriority(source)
		if priority > highestPriority {
			highestPriority = priority
			winningSource = source
		}
	}

	return winningSource
}

func (r *ConcreteConflictResolver) MergeEntityAttributes(entities []types.PMAEntity) map[string]interface{} {
	merged := make(map[string]interface{})

	// Merge attributes from all entities, with higher priority sources overriding
	for _, entity := range entities {
		for key, value := range entity.GetAttributes() {
			merged[key] = value
		}
	}

	return merged
}

func (r *ConcreteConflictResolver) CreateVirtualEntity(entities []types.PMAEntity) (types.PMAEntity, error) {
	if len(entities) == 0 {
		return nil, fmt.Errorf("no entities provided for virtual entity creation")
	}

	// Use the highest priority entity as the base
	baseEntity, err := r.ResolveEntityConflict(entities)
	if err != nil {
		return nil, err
	}

	// Create a virtual entity by merging attributes
	mergedAttributes := r.MergeEntityAttributes(entities)

	// Create metadata for virtual entity
	sources := make([]types.PMASourceType, 0, len(entities))
	for _, entity := range entities {
		sources = append(sources, entity.GetSource())
	}

	metadata := &types.PMAMetadata{
		Source:         types.SourcePMA,
		SourceEntityID: baseEntity.GetID(),
		LastSynced:     time.Now(),
		QualityScore:   baseEntity.GetQualityScore(),
		IsVirtual:      true,
		VirtualSources: sources,
	}

	// Create new virtual entity based on the base entity type
	// This is a simplified implementation - a full implementation would need
	// to create the appropriate concrete type
	virtualEntity := &types.PMABaseEntity{
		ID:           baseEntity.GetID(),
		Type:         baseEntity.GetType(),
		FriendlyName: baseEntity.GetFriendlyName(),
		Icon:         baseEntity.GetIcon(),
		State:        baseEntity.GetState(),
		Attributes:   mergedAttributes,
		LastUpdated:  time.Now(),
		Capabilities: baseEntity.GetCapabilities(),
		RoomID:       baseEntity.GetRoomID(),
		AreaID:       baseEntity.GetAreaID(),
		DeviceID:     baseEntity.GetDeviceID(),
		Metadata:     metadata,
		Available:    baseEntity.IsAvailable(),
	}

	return virtualEntity, nil
}

// ConcreteSourcePriorityManager implements the SourcePriorityManager interface
type ConcreteSourcePriorityManager struct {
	priorities map[types.PMASourceType]int
	mutex      sync.RWMutex
}

// NewConcreteSourcePriorityManager creates a new source priority manager
func NewConcreteSourcePriorityManager() *ConcreteSourcePriorityManager {
	manager := &ConcreteSourcePriorityManager{
		priorities: make(map[types.PMASourceType]int),
	}

	// Set default priorities
	manager.priorities[types.SourceHomeAssistant] = 100
	manager.priorities[types.SourceRing] = 80
	manager.priorities[types.SourceShelly] = 70
	manager.priorities[types.SourceUPS] = 60
	manager.priorities[types.SourceNetwork] = 50
	manager.priorities[types.SourcePMA] = 10

	return manager
}

func (m *ConcreteSourcePriorityManager) SetSourcePriority(source types.PMASourceType, priority int) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.priorities[source] = priority
	return nil
}

func (m *ConcreteSourcePriorityManager) GetSourcePriority(source types.PMASourceType) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if priority, exists := m.priorities[source]; exists {
		return priority
	}

	return 0 // Default priority for unknown sources
}

func (m *ConcreteSourcePriorityManager) GetPriorityOrder() []types.PMASourceType {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	type sourcePriority struct {
		source   types.PMASourceType
		priority int
	}

	sources := make([]sourcePriority, 0, len(m.priorities))
	for source, priority := range m.priorities {
		sources = append(sources, sourcePriority{source, priority})
	}

	sort.Slice(sources, func(i, j int) bool {
		return sources[i].priority > sources[j].priority
	})

	result := make([]types.PMASourceType, 0, len(sources))
	for _, sp := range sources {
		result = append(result, sp.source)
	}

	return result
}

func (m *ConcreteSourcePriorityManager) ShouldOverride(currentSource, newSource types.PMASourceType) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	currentPriority := m.GetSourcePriority(currentSource)
	newPriority := m.GetSourcePriority(newSource)

	return newPriority > currentPriority
}
