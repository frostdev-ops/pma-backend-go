package unified

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/internal/adapters/network"
	"github.com/frostdev-ops/pma-backend-go/internal/adapters/ring"
	"github.com/frostdev-ops/pma-backend-go/internal/adapters/shelly"
	"github.com/frostdev-ops/pma-backend-go/internal/adapters/ups"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types/registries"
	"github.com/sirupsen/logrus"
)

// RoomServiceInterface defines the interface for room service operations
type RoomServiceInterface interface {
	GetRoomByID(ctx context.Context, roomID string) (*types.PMARoom, error)
}

// EventEmitter defines the interface for broadcasting real-time updates
type EventEmitter interface {
	BroadcastPMAEntityStateChange(entityID string, oldState, newState interface{}, entity interface{})
	BroadcastPMAEntityAdded(entity interface{})
	BroadcastPMAEntityRemoved(entityID string, source interface{})
	BroadcastPMASyncStatus(source string, status string, details map[string]interface{})
	BroadcastPMAAdapterStatus(adapterID, adapterName, source, status string, health interface{}, metrics interface{})
}

// UnifiedEntityService manages all entities through the PMA type system
type UnifiedEntityService struct {
	typeRegistry    *types.PMATypeRegistry
	registryManager *registries.RegistryManager
	logger          *logrus.Logger
	mutex           sync.RWMutex
	roomService     RoomServiceInterface
	eventEmitter    EventEmitter

	// Caching
	entityCache     map[string]types.PMAEntity
	cacheExpiry     time.Duration
	lastCacheUpdate time.Time

	// Sync scheduler
	syncTicker   *time.Ticker
	syncStopChan chan bool
	syncRunning  bool
	config       *config.Config
}

// NewUnifiedEntityService creates a new unified entity service
func NewUnifiedEntityService(
	typeRegistry *types.PMATypeRegistry,
	config *config.Config,
	logger *logrus.Logger,
) *UnifiedEntityService {
	// Create the registry manager
	registryManager := registries.NewRegistryManager(logger)

	service := &UnifiedEntityService{
		typeRegistry:    typeRegistry,
		registryManager: registryManager,
		logger:          logger,
		entityCache:     make(map[string]types.PMAEntity),
		cacheExpiry:     5 * time.Minute,
		config:          config,
		syncStopChan:    make(chan bool, 1),
	}

	return service
}

// RegisterAdapter registers a new adapter with the registry manager
func (s *UnifiedEntityService) RegisterAdapter(adapter types.PMAAdapter) error {
	return s.registryManager.GetAdapterRegistry().RegisterAdapter(adapter)
}

// GetRegistryManager returns the registry manager
func (s *UnifiedEntityService) GetRegistryManager() *registries.RegistryManager {
	return s.registryManager
}

// SetRoomService sets the room service for room lookups
func (s *UnifiedEntityService) SetRoomService(roomService RoomServiceInterface) {
	s.roomService = roomService
}

// SetEventEmitter sets the event emitter for real-time broadcasting
func (s *UnifiedEntityService) SetEventEmitter(eventEmitter EventEmitter) {
	s.eventEmitter = eventEmitter
	s.logger.Info("Event emitter configured for real-time WebSocket updates")
}

// InitializeAdapters initializes all configured adapters
func (s *UnifiedEntityService) InitializeAdapters(config *config.Config) error {
	var errors []error

	// Initialize HomeAssistant adapter
	if config.HomeAssistant.URL != "" && config.HomeAssistant.Token != "" {
		haAdapter := homeassistant.NewHomeAssistantAdapter(config, s.logger)
		if err := s.RegisterAdapter(haAdapter); err != nil {
			errors = append(errors, fmt.Errorf("failed to register HA adapter: %w", err))
		} else {
			s.logger.Info("HomeAssistant adapter registered successfully")
		}
	}

	// Initialize Ring adapter
	if config.Devices.Ring.Enabled && config.Devices.Ring.Email != "" && config.Devices.Ring.Password != "" {
		ringConfig := ring.RingAdapterConfig{
			Credentials: ring.RingCredentials{
				Email:    config.Devices.Ring.Email,
				Password: config.Devices.Ring.Password,
			},
			AutoReconnect: config.Devices.Ring.AutoReconnect,
		}
		ringAdapter := ring.NewRingAdapter(ringConfig, config, s.logger)
		if err := s.RegisterAdapter(ringAdapter); err != nil {
			errors = append(errors, fmt.Errorf("failed to register Ring adapter: %w", err))
		} else {
			s.logger.Info("Ring adapter registered successfully")
		}
	}

	// Initialize Shelly adapter
	if config.Devices.Shelly.Enabled {
		pollInterval, err := time.ParseDuration(config.Devices.Shelly.PollInterval)
		if err != nil {
			pollInterval = 30 * time.Second // Default fallback
		}

		// Convert device configs to IP addresses
		var deviceIPs []string
		for _, device := range config.Devices.Shelly.Devices {
			if device.Enabled {
				deviceIPs = append(deviceIPs, device.IP)
			}
		}

		shellyConfig := shelly.ShellyAdapterConfig{
			Username:      config.Devices.Shelly.Username,
			Password:      config.Devices.Shelly.Password,
			PollInterval:  pollInterval,
			AutoReconnect: config.Devices.Shelly.AutoReconnect,
			Devices:       deviceIPs,
		}
		shellyAdapter := shelly.NewShellyAdapter(shellyConfig, s.logger)
		if err := s.RegisterAdapter(shellyAdapter); err != nil {
			errors = append(errors, fmt.Errorf("failed to register Shelly adapter: %w", err))
		} else {
			s.logger.Info("Shelly adapter registered successfully")
		}
	}

	// Initialize UPS adapter
	if config.Devices.UPS.Enabled && config.Devices.UPS.NUTHost != "" {
		pollInterval, err := time.ParseDuration(config.Devices.UPS.PollInterval)
		if err != nil {
			pollInterval = 30 * time.Second // Default fallback
		}

		// Convert UPS names from string to slice
		var upsNames []string
		if config.Devices.UPS.UPSName != "" {
			upsNames = append(upsNames, config.Devices.UPS.UPSName)
		}

		upsConfig := ups.UPSAdapterConfig{
			Host:         config.Devices.UPS.NUTHost,
			Port:         config.Devices.UPS.NUTPort,
			Username:     config.Devices.UPS.Username,
			Password:     config.Devices.UPS.Password,
			UPSNames:     upsNames,
			PollInterval: pollInterval,
		}
		upsAdapter := ups.NewUPSAdapter(upsConfig, s.logger)
		if err := s.RegisterAdapter(upsAdapter); err != nil {
			errors = append(errors, fmt.Errorf("failed to register UPS adapter: %w", err))
		} else {
			s.logger.Info("UPS adapter registered successfully")
		}
	}

	// Initialize Network adapter
	if config.Devices.Network.Enabled && config.Router.BaseURL != "" {
		scanInterval, err := time.ParseDuration(config.Devices.Network.ScanInterval)
		if err != nil {
			scanInterval = 5 * time.Minute // Default fallback
		}

		networkConfig := network.NetworkAdapterConfig{
			RouterURL:    config.Router.BaseURL,
			AuthToken:    config.Router.AuthToken,
			PollInterval: scanInterval,
		}
		networkAdapter := network.NewNetworkAdapter(networkConfig, s.logger)
		if err := s.RegisterAdapter(networkAdapter); err != nil {
			errors = append(errors, fmt.Errorf("failed to register Network adapter: %w", err))
		} else {
			s.logger.Info("Network adapter registered successfully")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("adapter initialization had %d errors: %v", len(errors), errors)
	}

	return nil
}

// GetAll retrieves all entities with optional filtering
func (s *UnifiedEntityService) GetAll(ctx context.Context, options GetAllOptions) ([]*EntityWithRoom, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Get all entities from registry
	entities, err := s.registryManager.GetEntityRegistry().GetAllEntities()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get all entities from registry")
		return nil, fmt.Errorf("failed to get entities: %w", err)
	}

	// Filter entities based on options
	filteredEntities := s.filterEntities(entities, options)

	// Convert to EntityWithRoom format
	result := make([]*EntityWithRoom, 0, len(filteredEntities))
	for _, entity := range filteredEntities {
		entityWithRoom := &EntityWithRoom{
			Entity: entity,
		}

		// Add room information if requested
		if options.IncludeRoom && entity.GetRoomID() != nil {
			room, err := s.roomService.GetRoomByID(ctx, *entity.GetRoomID())
			if err != nil {
				s.logger.WithError(err).Warnf("Failed to get room for entity %s", entity.GetID())
			} else {
				entityWithRoom.Room = room
			}
		}

		// Add area information if requested
		if options.IncludeArea && entity.GetAreaID() != nil {
			area, err := s.getAreaByID(ctx, *entity.GetAreaID())
			if err != nil {
				s.logger.WithError(err).Warnf("Failed to get area for entity %s", entity.GetID())
			} else {
				entityWithRoom.Area = area
			}
		}

		result = append(result, entityWithRoom)
	}

	return result, nil
}

// GetByID retrieves a specific entity by ID
func (s *UnifiedEntityService) GetByID(ctx context.Context, entityID string, options GetEntityOptions) (*EntityWithRoom, error) {
	entity, err := s.registryManager.GetEntityRegistry().GetEntity(entityID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get entity: %s", entityID)
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}

	entityWithRoom := &EntityWithRoom{
		Entity: entity,
	}

	// Add room information if requested
	if options.IncludeRoom && entity.GetRoomID() != nil {
		room, err := s.roomService.GetRoomByID(ctx, *entity.GetRoomID())
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to get room for entity %s", entityID)
		} else {
			entityWithRoom.Room = room
		}
	}

	// Add area information if requested
	if options.IncludeArea && entity.GetAreaID() != nil {
		area, err := s.getAreaByID(ctx, *entity.GetAreaID())
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to get area for entity %s", entityID)
		} else {
			entityWithRoom.Area = area
		}
	}

	return entityWithRoom, nil
}

// GetByType retrieves entities by type
func (s *UnifiedEntityService) GetByType(ctx context.Context, entityType types.PMAEntityType, options GetAllOptions) ([]*EntityWithRoom, error) {
	entities, err := s.registryManager.GetEntityRegistry().GetEntitiesByType(entityType)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get entities by type: %s", entityType)
		return nil, fmt.Errorf("failed to get entities by type: %w", err)
	}

	// Apply additional filtering
	filteredEntities := s.filterEntities(entities, options)

	// Convert to EntityWithRoom format
	result := make([]*EntityWithRoom, 0, len(filteredEntities))
	for _, entity := range filteredEntities {
		entityWithRoom := &EntityWithRoom{
			Entity: entity,
		}

		if options.IncludeRoom && entity.GetRoomID() != nil {
			room, err := s.roomService.GetRoomByID(ctx, *entity.GetRoomID())
			if err != nil {
				s.logger.WithError(err).Warnf("Failed to get room for entity %s", entity.GetID())
			} else {
				entityWithRoom.Room = room
			}
		}

		result = append(result, entityWithRoom)
	}

	return result, nil
}

// GetBySource retrieves entities from a specific source
func (s *UnifiedEntityService) GetBySource(ctx context.Context, source types.PMASourceType, options GetAllOptions) ([]*EntityWithRoom, error) {
	entities, err := s.registryManager.GetEntityRegistry().GetEntitiesBySource(source)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get entities by source: %s", source)
		return nil, fmt.Errorf("failed to get entities by source: %w", err)
	}

	// Apply additional filtering
	filteredEntities := s.filterEntities(entities, options)

	// Convert to EntityWithRoom format
	result := make([]*EntityWithRoom, 0, len(filteredEntities))
	for _, entity := range filteredEntities {
		entityWithRoom := &EntityWithRoom{
			Entity: entity,
		}

		if options.IncludeRoom && entity.GetRoomID() != nil {
			room, err := s.roomService.GetRoomByID(ctx, *entity.GetRoomID())
			if err != nil {
				s.logger.WithError(err).Warnf("Failed to get room for entity %s", entity.GetID())
			} else {
				entityWithRoom.Room = room
			}
		}

		result = append(result, entityWithRoom)
	}

	return result, nil
}

// GetByRoom retrieves entities in a specific room
func (s *UnifiedEntityService) GetByRoom(ctx context.Context, roomID string, options GetAllOptions) ([]*EntityWithRoom, error) {
	entities, err := s.registryManager.GetEntityRegistry().GetEntitiesByRoom(roomID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get entities by room: %s", roomID)
		return nil, fmt.Errorf("failed to get entities by room: %w", err)
	}

	// Apply additional filtering
	filteredEntities := s.filterEntities(entities, options)

	// Convert to EntityWithRoom format
	result := make([]*EntityWithRoom, 0, len(filteredEntities))
	for _, entity := range filteredEntities {
		entityWithRoom := &EntityWithRoom{
			Entity: entity,
		}

		if options.IncludeRoom {
			room, err := s.roomService.GetRoomByID(ctx, roomID)
			if err != nil {
				s.logger.WithError(err).Warnf("Failed to get room: %s", roomID)
			} else {
				entityWithRoom.Room = room
			}
		}

		result = append(result, entityWithRoom)
	}

	return result, nil
}

// Search searches entities based on a query string
func (s *UnifiedEntityService) Search(ctx context.Context, query string, options GetAllOptions) ([]*EntityWithRoom, error) {
	entities, err := s.registryManager.GetEntityRegistry().SearchEntities(query)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to search entities with query: %s", query)
		return nil, fmt.Errorf("failed to search entities: %w", err)
	}

	// Apply additional filtering
	filteredEntities := s.filterEntities(entities, options)

	// Convert to EntityWithRoom format
	result := make([]*EntityWithRoom, 0, len(filteredEntities))
	for _, entity := range filteredEntities {
		entityWithRoom := &EntityWithRoom{
			Entity: entity,
		}

		if options.IncludeRoom && entity.GetRoomID() != nil {
			room, err := s.roomService.GetRoomByID(ctx, *entity.GetRoomID())
			if err != nil {
				s.logger.WithError(err).Warnf("Failed to get room for entity %s", entity.GetID())
			} else {
				entityWithRoom.Room = room
			}
		}

		result = append(result, entityWithRoom)
	}

	return result, nil
}

// ExecuteAction executes a control action on an entity
func (s *UnifiedEntityService) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	// Validate the action
	if err := s.validateAction(action); err != nil {
		return &types.PMAControlResult{
			Success:     false,
			EntityID:    action.EntityID,
			Action:      action.Action,
			ProcessedAt: time.Now(),
			Error: &types.PMAError{
				Code:    "VALIDATION_FAILED",
				Message: err.Error(),
				Source:  "unified_service",
			},
		}, nil
	}

	// Get the entity to determine the appropriate adapter
	entity, err := s.registryManager.GetEntityRegistry().GetEntity(action.EntityID)
	if err != nil {
		return &types.PMAControlResult{
			Success:     false,
			EntityID:    action.EntityID,
			Action:      action.Action,
			ProcessedAt: time.Now(),
			Error: &types.PMAError{
				Code:    "ENTITY_NOT_FOUND",
				Message: fmt.Sprintf("Entity not found: %s", action.EntityID),
				Source:  "unified_service",
			},
		}, nil
	}

	// Get the appropriate adapter for this entity's source
	adapter, err := s.registryManager.GetAdapterRegistry().GetAdapterBySource(entity.GetSource())
	if err != nil {
		return &types.PMAControlResult{
			Success:     false,
			EntityID:    action.EntityID,
			Action:      action.Action,
			ProcessedAt: time.Now(),
			Error: &types.PMAError{
				Code:    "ADAPTER_NOT_FOUND",
				Message: fmt.Sprintf("No adapter found for source: %s", entity.GetSource()),
				Source:  "unified_service",
			},
		}, nil
	}

	// Execute the action through the appropriate adapter
	result, err := adapter.ExecuteAction(ctx, action)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to execute action %s on entity %s", action.Action, action.EntityID)
		return &types.PMAControlResult{
			Success:     false,
			EntityID:    action.EntityID,
			Action:      action.Action,
			ProcessedAt: time.Now(),
			Error: &types.PMAError{
				Code:    "EXECUTION_ERROR",
				Message: err.Error(),
				Source:  string(entity.GetSource()),
			},
		}, nil
	}

	// Update the entity in the registry if the action was successful
	if result.Success {
		// Store the old state for broadcasting
		oldEntity := entity

		// Broadcast the action execution to WebSocket clients
		if s.eventEmitter != nil {
			s.eventEmitter.BroadcastPMAEntityStateChange(
				action.EntityID,
				oldEntity.GetState(),
				result.NewState, // Assuming the result contains the new state
				entity,
			)
		}

		// Refresh the entity from the source to get the latest state
		go func() {
			time.Sleep(1 * time.Second) // Give the device time to update
			s.refreshEntity(context.Background(), action.EntityID)
		}()
	}

	return result, nil
}

// SyncFromSource synchronizes entities from a specific source
func (s *UnifiedEntityService) SyncFromSource(ctx context.Context, source types.PMASourceType) (*SyncResult, error) {
	adapter, err := s.registryManager.GetAdapterRegistry().GetAdapterBySource(source)
	if err != nil {
		return nil, fmt.Errorf("no adapter found for source %s: %w", source, err)
	}

	startTime := time.Now()

	// Broadcast sync start
	if s.eventEmitter != nil {
		s.eventEmitter.BroadcastPMASyncStatus(string(source), "syncing", map[string]interface{}{
			"started_at": startTime,
		})
	}

	// Sync entities
	entities, err := adapter.SyncEntities(ctx)
	if err != nil {
		// Broadcast sync error
		if s.eventEmitter != nil {
			s.eventEmitter.BroadcastPMASyncStatus(string(source), "error", map[string]interface{}{
				"error":    err.Error(),
				"duration": time.Since(startTime),
			})
		}

		return &SyncResult{
			Source:      source,
			Success:     false,
			Error:       err.Error(),
			Duration:    time.Since(startTime),
			ProcessedAt: time.Now(),
		}, err
	}

	// Register/update entities in the registry
	registeredCount := 0
	updatedCount := 0
	var errors []string

	for _, entity := range entities {
		// Check if entity already exists
		existingEntity, err := s.registryManager.GetEntityRegistry().GetEntity(entity.GetID())
		if err != nil {
			// Entity doesn't exist, register it
			if err := s.registryManager.GetEntityRegistry().RegisterEntity(entity); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to register entity %s: %v", entity.GetID(), err))
				continue
			}
			registeredCount++

			// Broadcast new entity added
			if s.eventEmitter != nil {
				s.eventEmitter.BroadcastPMAEntityAdded(entity)
			}
		} else {
			// Entity exists, check for conflicts and update
			if s.shouldUpdateEntity(existingEntity, entity) {
				// Store old state for broadcasting
				oldState := existingEntity.GetState()

				if err := s.registryManager.GetEntityRegistry().UpdateEntity(entity); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to update entity %s: %v", entity.GetID(), err))
					continue
				}
				updatedCount++

				// Broadcast entity state change
				if s.eventEmitter != nil && oldState != entity.GetState() {
					s.eventEmitter.BroadcastPMAEntityStateChange(
						entity.GetID(),
						oldState,
						entity.GetState(),
						entity,
					)
				}
			}
		}
	}

	result := &SyncResult{
		Source:             source,
		Success:            len(errors) == 0,
		EntitiesFound:      len(entities),
		EntitiesRegistered: registeredCount,
		EntitiesUpdated:    updatedCount,
		Duration:           time.Since(startTime),
		ProcessedAt:        time.Now(),
	}

	if len(errors) > 0 {
		result.Error = fmt.Sprintf("Sync completed with %d errors: %s", len(errors), strings.Join(errors, "; "))
	}

	// Broadcast sync completion
	if s.eventEmitter != nil {
		status := "completed"
		if len(errors) > 0 {
			status = "completed_with_errors"
		}

		s.eventEmitter.BroadcastPMASyncStatus(string(source), status, map[string]interface{}{
			"entities_found":      len(entities),
			"entities_registered": registeredCount,
			"entities_updated":    updatedCount,
			"duration":            time.Since(startTime),
			"error_count":         len(errors),
		})
	}

	s.logger.WithFields(logrus.Fields{
		"source":              source,
		"entities_found":      len(entities),
		"entities_registered": registeredCount,
		"entities_updated":    updatedCount,
		"duration":            time.Since(startTime),
		"errors":              len(errors),
	}).Info("Entity sync completed")

	return result, nil
}

// SyncFromAllSources synchronizes entities from all available sources
func (s *UnifiedEntityService) SyncFromAllSources(ctx context.Context) ([]*SyncResult, error) {
	adapters := s.registryManager.GetAdapterRegistry().GetConnectedAdapters()

	if len(adapters) == 0 {
		return nil, fmt.Errorf("no connected adapters found")
	}

	// Log adapter count
	s.logger.WithField("adapter_count", len(adapters)).Info("Starting sync from all sources")

	var results []*SyncResult
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for _, adapter := range adapters {
		wg.Add(1)
		go func(adapter types.PMAAdapter) {
			defer wg.Done()

			result, err := s.SyncFromSource(ctx, adapter.GetSourceType())
			if err != nil {
				result = &SyncResult{
					Source:      adapter.GetSourceType(),
					Success:     false,
					Error:       err.Error(),
					ProcessedAt: time.Now(),
				}
			}

			mutex.Lock()
			results = append(results, result)
			mutex.Unlock()
		}(adapter)
	}

	wg.Wait()
	return results, nil
}

// StartPeriodicSync starts the periodic synchronization scheduler
func (s *UnifiedEntityService) StartPeriodicSync() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.syncRunning {
		return fmt.Errorf("periodic sync is already running")
	}

	if !s.config.HomeAssistant.Sync.Enabled {
		s.logger.Info("Periodic sync disabled in configuration")
		return nil
	}

	// Parse sync interval
	syncInterval, err := time.ParseDuration(s.config.HomeAssistant.Sync.FullSyncInterval)
	if err != nil {
		s.logger.WithError(err).Warn("Invalid sync interval, using default 1 hour")
		syncInterval = 1 * time.Hour
	}

	s.logger.WithField("interval", syncInterval).Info("Starting periodic entity sync scheduler")

	// Create ticker for periodic sync
	s.syncTicker = time.NewTicker(syncInterval)
	s.syncRunning = true

	// Start sync goroutine
	go s.periodicSyncLoop()

	return nil
}

// StopPeriodicSync stops the periodic synchronization scheduler
func (s *UnifiedEntityService) StopPeriodicSync() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.syncRunning {
		return
	}

	s.logger.Info("Stopping periodic entity sync scheduler")

	if s.syncTicker != nil {
		s.syncTicker.Stop()
		s.syncTicker = nil
	}

	s.syncStopChan <- true
	s.syncRunning = false
}

// periodicSyncLoop runs the periodic synchronization
func (s *UnifiedEntityService) periodicSyncLoop() {
	for {
		select {
		case <-s.syncStopChan:
			s.logger.Info("Periodic sync loop stopped")
			return
		case <-s.syncTicker.C:
			s.performPeriodicSync()
		}
	}
}

// performPeriodicSync executes a full sync from all sources
func (s *UnifiedEntityService) performPeriodicSync() {
	s.logger.Info("Starting periodic sync from all sources")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	results, err := s.SyncFromAllSources(ctx)
	if err != nil {
		s.logger.WithError(err).Error("Periodic sync failed")
		return
	}

	// Log results
	totalFound := 0
	totalRegistered := 0
	totalUpdated := 0
	successCount := 0

	for _, result := range results {
		totalFound += result.EntitiesFound
		totalRegistered += result.EntitiesRegistered
		totalUpdated += result.EntitiesUpdated
		if result.Success {
			successCount++
		}
	}

	s.logger.WithFields(logrus.Fields{
		"total_sources":             len(results),
		"successful_sources":        successCount,
		"total_entities_found":      totalFound,
		"total_entities_registered": totalRegistered,
		"total_entities_updated":    totalUpdated,
	}).Info("Periodic sync completed")
}

// Helper methods

func (s *UnifiedEntityService) filterEntities(entities []types.PMAEntity, options GetAllOptions) []types.PMAEntity {
	if options.Domain == "" && !options.AvailableOnly && len(options.Capabilities) == 0 {
		return entities
	}

	var filtered []types.PMAEntity
	for _, entity := range entities {
		// Filter by domain (entity type)
		if options.Domain != "" && entity.GetType() != types.PMAEntityType(options.Domain) {
			continue
		}

		// Filter by availability
		if options.AvailableOnly && !entity.IsAvailable() {
			continue
		}

		// Filter by capabilities
		if len(options.Capabilities) > 0 {
			hasAllCapabilities := true
			for _, requiredCap := range options.Capabilities {
				if !entity.HasCapability(requiredCap) {
					hasAllCapabilities = false
					break
				}
			}
			if !hasAllCapabilities {
				continue
			}
		}

		filtered = append(filtered, entity)
	}

	return filtered
}

func (s *UnifiedEntityService) validateAction(action types.PMAControlAction) error {
	if action.EntityID == "" {
		return fmt.Errorf("entity ID is required")
	}
	if action.Action == "" {
		return fmt.Errorf("action is required")
	}

	// Get the entity to validate action compatibility
	entity, err := s.registryManager.GetEntityRegistry().GetEntity(action.EntityID)
	if err != nil {
		return fmt.Errorf("entity not found: %s", action.EntityID)
	}

	// Check if entity supports the action
	availableActions := entity.GetAvailableActions()
	actionSupported := false
	for _, availableAction := range availableActions {
		if availableAction == action.Action {
			actionSupported = true
			break
		}
	}

	if !actionSupported {
		return fmt.Errorf("action %s not supported by entity %s", action.Action, action.EntityID)
	}

	return nil
}

func (s *UnifiedEntityService) shouldUpdateEntity(existing, new types.PMAEntity) bool {
	// Update if the new entity has a more recent timestamp
	return new.GetLastUpdated().After(existing.GetLastUpdated())
}

func (s *UnifiedEntityService) refreshEntity(ctx context.Context, entityID string) {
	// Get the current entity state before refresh
	oldEntity, err := s.registryManager.GetEntityRegistry().GetEntity(entityID)
	if err != nil {
		s.logger.WithError(err).Warnf("Failed to get entity for refresh: %s", entityID)
		return
	}

	adapter, err := s.registryManager.GetAdapterRegistry().GetAdapterBySource(oldEntity.GetSource())
	if err != nil {
		s.logger.WithError(err).Warnf("Failed to get adapter for entity refresh: %s", entityID)
		return
	}

	// Store old state for comparison
	oldState := oldEntity.GetState()

	// Sync entities from the source (this will update the registry)
	_, err = adapter.SyncEntities(ctx)
	if err != nil {
		s.logger.WithError(err).Warnf("Failed to sync entities for refresh")
		return
	}

	// Get the updated entity and check if state changed
	newEntity, err := s.registryManager.GetEntityRegistry().GetEntity(entityID)
	if err != nil {
		s.logger.WithError(err).Warnf("Failed to get refreshed entity: %s", entityID)
		return
	}

	// Broadcast state change if the state actually changed
	newState := newEntity.GetState()
	if s.eventEmitter != nil && oldState != newState {
		s.eventEmitter.BroadcastPMAEntityStateChange(
			entityID,
			oldState,
			newState,
			newEntity,
		)

		s.logger.WithFields(logrus.Fields{
			"entity_id": entityID,
			"old_state": oldState,
			"new_state": newState,
		}).Debug("Broadcasted entity state change")
	}
}

func (s *UnifiedEntityService) getRoomByID(ctx context.Context, roomID string) (*types.PMARoom, error) {
	if s.roomService == nil {
		return &types.PMARoom{
			ID:   roomID,
			Name: "Unknown Room",
		}, nil
	}

	// Use the room service to get actual room data
	room, err := s.roomService.GetRoomByID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	// The room service should return a *types.PMARoom already
	return room, nil
}

func (s *UnifiedEntityService) getAreaByID(ctx context.Context, areaID string) (*types.PMAArea, error) {
	// This would need to be implemented with an area service
	// For now, return a placeholder
	return &types.PMAArea{
		ID:   areaID,
		Name: "Unknown Area",
	}, nil
}

// Types for service options and responses

// GetAllOptions defines options for retrieving all entities
type GetAllOptions struct {
	Domain        string                `json:"domain,omitempty"`
	IncludeRoom   bool                  `json:"include_room,omitempty"`
	IncludeArea   bool                  `json:"include_area,omitempty"`
	AvailableOnly bool                  `json:"available_only,omitempty"`
	Capabilities  []types.PMACapability `json:"capabilities,omitempty"`
}

// GetEntityOptions defines options for retrieving a single entity
type GetEntityOptions struct {
	IncludeRoom bool `json:"include_room,omitempty"`
	IncludeArea bool `json:"include_area,omitempty"`
}

// EntityWithRoom represents an entity with optional room and area information
type EntityWithRoom struct {
	Entity types.PMAEntity `json:"entity"`
	Room   *types.PMARoom  `json:"room,omitempty"`
	Area   *types.PMAArea  `json:"area,omitempty"`
}

// SyncResult represents the result of a synchronization operation
type SyncResult struct {
	Source             types.PMASourceType `json:"source"`
	Success            bool                `json:"success"`
	EntitiesFound      int                 `json:"entities_found"`
	EntitiesRegistered int                 `json:"entities_registered"`
	EntitiesUpdated    int                 `json:"entities_updated"`
	Error              string              `json:"error,omitempty"`
	Duration           time.Duration       `json:"duration"`
	ProcessedAt        time.Time           `json:"processed_at"`
}
