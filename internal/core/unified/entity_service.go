package unified

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/adapters/homeassistant"
	"github.com/frostdev-ops/pma-backend-go/internal/adapters/network"
	"github.com/frostdev-ops/pma-backend-go/internal/adapters/ring"
	"github.com/frostdev-ops/pma-backend-go/internal/adapters/shelly"
	"github.com/frostdev-ops/pma-backend-go/internal/adapters/ups"
	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/cache"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types/registries"
	"github.com/frostdev-ops/pma-backend-go/pkg/debug"
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

	// Redis-based caching
	redisCache      *cache.RedisEntityCache
	cacheExpiry     time.Duration
	lastCacheUpdate time.Time

	// Sync scheduler
	syncTicker   *time.Ticker
	syncStopChan chan bool
	syncRunning  bool
	config       *config.Config

	// Memory leak prevention
	syncSemaphore  chan struct{} // Limit concurrent sync operations
	maxSyncWorkers int           // Maximum concurrent sync workers
	syncTimeout    time.Duration // Timeout for sync operations
}

// NewUnifiedEntityService creates a new unified entity service
func NewUnifiedEntityService(
	typeRegistry *types.PMATypeRegistry,
	config *config.Config,
	logger *logrus.Logger,
) *UnifiedEntityService {
	// Create the registry manager
	registryManager := registries.NewRegistryManager(logger)

	// Initialize Redis cache if enabled
	var redisCache *cache.RedisEntityCache
	if config.Redis.Enabled {
		var err error
		redisCache, err = cache.NewRedisEntityCache(config, logger)
		if err != nil {
			logger.WithError(err).Error("Failed to initialize Redis entity cache, falling back to in-memory cache")
			redisCache = nil
		}
	} else {
		logger.Info("Redis caching disabled, using in-memory fallback")
	}

	service := &UnifiedEntityService{
		typeRegistry:    typeRegistry,
		registryManager: registryManager,
		logger:          logger,
		redisCache:      redisCache,
		cacheExpiry:     5 * time.Minute,
		config:          config,
		syncStopChan:    make(chan bool, 1),
		syncSemaphore:   make(chan struct{}, 3), // Limit to 3 concurrent syncs
		maxSyncWorkers:  3,                      // Maximum 3 concurrent sync workers
		syncTimeout:     10 * time.Minute,
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

			// Set up event handler to forward HA state changes to unified service
			haAdapter.SetEventHandler(func(entityID, oldState, newState string) {
				s.logger.WithFields(logrus.Fields{
					"entity_id": entityID,
					"old_state": oldState,
					"new_state": newState,
				}).Debug("Received HA state change, updating unified service")

				// Create a background context for the update
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				// Update the entity state in the unified service.
				// This will update the in-memory state AND broadcast the change via WebSocket.
				_, err := s.UpdateEntityState(ctx, entityID, newState, types.SourceHomeAssistant)
				if err != nil {
					s.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to update entity state in unified service")
				}
			})

			// CRITICAL FIX: Connect the adapter synchronously during startup to ensure it's ready for sync
			s.logger.Info("Connecting Home Assistant adapter synchronously during startup...")

			// Create a context with timeout for the connection attempt
			connectCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			if err := haAdapter.Connect(connectCtx); err != nil {
				s.logger.WithError(err).Error("Failed to connect Home Assistant adapter during startup")
				errors = append(errors, fmt.Errorf("failed to connect HA adapter: %w", err))
			} else {
				s.logger.Info("Home Assistant adapter connected successfully during startup")
			}
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

	// Initialize Shelly adapter with enhanced discovery
	if config.Devices.Shelly.Enabled {
		discoveryInterval, err := time.ParseDuration(config.Devices.Shelly.DiscoveryInterval)
		if err != nil {
			discoveryInterval = 5 * time.Minute
		}

		discoveryTimeout, err := time.ParseDuration(config.Devices.Shelly.DiscoveryTimeout)
		if err != nil {
			discoveryTimeout = 30 * time.Second
		}

		pollInterval, err := time.ParseDuration(config.Devices.Shelly.PollInterval)
		if err != nil {
			pollInterval = 30 * time.Second
		}

		healthCheckInterval, err := time.ParseDuration(config.Devices.Shelly.HealthCheckInterval)
		if err != nil {
			healthCheckInterval = 60 * time.Second
		}

		retryBackoff, err := time.ParseDuration(config.Devices.Shelly.RetryBackoff)
		if err != nil {
			retryBackoff = 10 * time.Second
		}

		shellyConfig := shelly.ShellyAdapterConfig{
			Enabled:                config.Devices.Shelly.Enabled,
			DiscoveryInterval:      discoveryInterval,
			DiscoveryTimeout:       discoveryTimeout,
			NetworkScanEnabled:     config.Devices.Shelly.NetworkScanEnabled,
			NetworkScanRanges:      config.Devices.Shelly.NetworkScanRanges,
			AutoWiFiSetup:          config.Devices.Shelly.AutoWiFiSetup,
			DefaultUsername:        config.Devices.Shelly.DefaultUsername,
			DefaultPassword:        config.Devices.Shelly.DefaultPassword,
			PollInterval:           pollInterval,
			MaxDevices:             config.Devices.Shelly.MaxDevices,
			HealthCheckInterval:    healthCheckInterval,
			RetryAttempts:          config.Devices.Shelly.RetryAttempts,
			RetryBackoff:           retryBackoff,
			EnableGen1Support:      config.Devices.Shelly.EnableGen1Support,
			EnableGen2Support:      config.Devices.Shelly.EnableGen2Support,
			DiscoveryBroadcastAddr: config.Devices.Shelly.DiscoveryBroadcastAddr,

			// Auto-detection configuration
			AutoDetectSubnets:         config.Devices.Shelly.AutoDetectSubnets,
			AutoDetectInterfaceFilter: config.Devices.Shelly.AutoDetectInterfaceFilter,
			ExcludeLoopback:           config.Devices.Shelly.ExcludeLoopback,
			ExcludeDockerInterfaces:   config.Devices.Shelly.ExcludeDockerInterfaces,
			MinSubnetSize:             config.Devices.Shelly.MinSubnetSize,
		}
		// Create debug logger for Shelly adapter
		debugConfig := &debug.DebugConfig{
			Enabled:     true,
			Level:       "debug",
			Console:     true,
			FileEnabled: false,
		}
		debugLogger, err := debug.NewDebugLogger(debugConfig)
		if err != nil {
			s.logger.WithError(err).Error("Failed to create debug logger for Shelly adapter")
			debugLogger = nil
		}

		shellyAdapter := shelly.NewShellyAdapter(shellyConfig, s.logger, debugLogger)
		if err := s.RegisterAdapter(shellyAdapter); err != nil {
			errors = append(errors, fmt.Errorf("failed to register Shelly adapter: %w", err))
		} else {
			s.logger.Info("Enhanced Shelly adapter registered successfully")
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
	s.logger.Debug("🔍 GetAll method starting...")

	// Add timeout protection
	done := make(chan []*EntityWithRoom, 1)
	errChan := make(chan error, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.WithField("panic", r).Error("GetAll method panic recovered")
				errChan <- fmt.Errorf("GetAll panic: %v", r)
			}
		}()

		s.logger.Debug("🔒 Attempting to acquire read lock...")
		s.mutex.RLock()
		s.logger.Debug("✅ Read lock acquired")
		defer func() {
			s.mutex.RUnlock()
			s.logger.Debug("🔓 Read lock released")
		}()

		// Get all entities from registry
		s.logger.Debug("📋 Getting entities from registry...")
		entities, err := s.registryManager.GetEntityRegistry().GetAllEntities()
		if err != nil {
			s.logger.WithError(err).Error("Failed to get all entities from registry")
			// Return empty array instead of error to prevent 500 errors
			s.logger.Info("📭 Registry error - returning empty array as fallback")
			done <- []*EntityWithRoom{}
			return
		}

		s.logger.WithField("entity_count", len(entities)).Debug("📊 Retrieved entities from registry")

		// If no entities found, return empty array instead of error
		if len(entities) == 0 {
			s.logger.Info("📭 No entities found in registry - returning empty array")
			done <- []*EntityWithRoom{}
			return
		}

		// Filter entities based on options
		s.logger.Debug("🔍 Filtering entities...")
		filteredEntities := s.filterEntities(entities, options)
		s.logger.WithField("filtered_count", len(filteredEntities)).Debug("✅ Entities filtered")

		// Convert to EntityWithRoom format (without room/area info to avoid additional service calls)
		result := make([]*EntityWithRoom, 0, len(filteredEntities))
		for _, entity := range filteredEntities {
			entityWithRoom := &EntityWithRoom{
				Entity: entity,
			}
			result = append(result, entityWithRoom)
		}

		s.logger.WithField("result_count", len(result)).Debug("🎯 GetAll completed successfully")
		done <- result
	}()

	// Wait for result or timeout
	select {
	case result := <-done:
		return result, nil
	case err := <-errChan:
		// Return empty array instead of error to prevent 500 errors
		s.logger.WithError(err).Error("GetAll error - returning empty array as fallback")
		return []*EntityWithRoom{}, nil
	case <-ctx.Done():
		s.logger.Error("🚨 GetAll method timed out - possible deadlock detected")
		// Return empty array instead of error to prevent 500 errors
		return []*EntityWithRoom{}, nil
	}
}

// GetByID retrieves a specific entity by ID
func (s *UnifiedEntityService) GetByID(ctx context.Context, entityID string, options GetEntityOptions) (*EntityWithRoom, error) {
	s.logger.WithFields(logrus.Fields{
		"entity_id":    entityID,
		"include_room": options.IncludeRoom,
		"include_area": options.IncludeArea,
	}).Debug("GetByID request received in unified service")

	// First, try to get entity from Redis cache
	var entity types.PMAEntity
	var err error

	if s.redisCache != nil {
		s.logger.WithField("entity_id", entityID).Debug("🔍 Checking Redis cache first")
		entity, err = s.redisCache.GetEntity(ctx, entityID)
		if err == nil {
			s.logger.WithField("entity_id", entityID).Info("✅ Entity found in Redis cache")
		} else {
			s.logger.WithField("entity_id", entityID).Debug("❌ Entity not found in Redis cache, falling back to registry")
		}
	}

	// If not found in Redis, fall back to entity registry
	if entity == nil {
		s.logger.WithField("entity_id", entityID).Debug("🔄 Getting entity from registry")
		entity, err = s.registryManager.GetEntityRegistry().GetEntity(entityID)
		if err != nil {
			s.logger.WithError(err).WithFields(logrus.Fields{
				"entity_id": entityID,
				"method":    "GetEntity from registry",
			}).Error("Failed to get entity from registry")

			// Let's also check what entities are actually in the registry
			if allEntities, getAllErr := s.GetAll(ctx, GetAllOptions{}); getAllErr == nil {
				s.logger.WithFields(logrus.Fields{
					"entity_id":      entityID,
					"total_entities": len(allEntities),
				}).Debug("Registry status during failed GetByID")

				// Check if we can find a similar entity ID
				for _, entityWithRoom := range allEntities {
					existingID := entityWithRoom.Entity.GetID()
					if strings.Contains(existingID, entityID) || strings.Contains(entityID, existingID) {
						s.logger.WithFields(logrus.Fields{
							"requested_id":  entityID,
							"similar_id":    existingID,
							"friendly_name": entityWithRoom.Entity.GetFriendlyName(),
						}).Info("Found similar entity ID in registry")
					}
				}
			}

			return nil, fmt.Errorf("failed to get entity: %w", err)
		}
		s.logger.WithField("entity_id", entityID).Debug("✅ Entity found in registry")
	}

	s.logger.WithFields(logrus.Fields{
		"entity_id":     entityID,
		"friendly_name": entity.GetFriendlyName(),
		"type":          entity.GetType(),
		"source":        entity.GetSource(),
		"state":         entity.GetState(),
	}).Debug("Entity found in registry")

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
	// Use semaphore to limit concurrent sync operations
	select {
	case s.syncSemaphore <- struct{}{}:
		defer func() { <-s.syncSemaphore }()
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return nil, fmt.Errorf("too many concurrent sync operations")
	}

	s.logger.WithField("source", source).Info("Starting entity sync from source")

	// Add timeout to prevent hanging syncs
	ctx, cancel := context.WithTimeout(ctx, s.syncTimeout)
	defer cancel()

	adapter, err := s.registryManager.GetAdapterRegistry().GetAdapterBySource(source)
	if err != nil {
		s.logger.WithError(err).WithField("source", source).Error("No adapter found for source")
		return nil, fmt.Errorf("no adapter found for source %s: %w", source, err)
	}

	s.logger.WithFields(logrus.Fields{
		"source":       source,
		"adapter_id":   adapter.GetID(),
		"adapter_name": adapter.GetName(),
	}).Info("Found adapter for sync")

	startTime := time.Now()

	// Broadcast sync start (non-blocking, but with limit)
	if s.eventEmitter != nil {
		// Use a single goroutine for all broadcasts to prevent memory leaks
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.WithField("panic", r).Error("WebSocket broadcast panicked")
				}
			}()

			s.eventEmitter.BroadcastPMASyncStatus(string(source), "syncing", map[string]interface{}{
				"started_at": startTime,
			})
		}()
	}

	// Sync entities
	s.logger.WithField("source", source).Info("Calling adapter.SyncEntities()")
	entities, err := adapter.SyncEntities(ctx)
	if err != nil {
		s.logger.WithError(err).WithField("source", source).Error("Adapter SyncEntities failed")

		// Broadcast sync error (non-blocking)
		if s.eventEmitter != nil {
			go func() {
				defer func() {
					if r := recover(); r != nil {
						s.logger.WithField("panic", r).Error("WebSocket error broadcast panicked")
					}
				}()

				s.eventEmitter.BroadcastPMASyncStatus(string(source), "error", map[string]interface{}{
					"error":    err.Error(),
					"duration": time.Since(startTime),
				})
			}()
		}

		return &SyncResult{
			Source:      source,
			Success:     false,
			Error:       err.Error(),
			Duration:    time.Since(startTime),
			ProcessedAt: time.Now(),
		}, err
	}

	s.logger.WithFields(logrus.Fields{
		"source":         source,
		"entities_count": len(entities),
	}).Info("Adapter returned entities for sync")

	// Process entities in batches to prevent memory spikes
	const batchSize = 10 // Reduced from 50 to prevent memory spikes
	registeredCount := 0
	updatedCount := 0
	var errors []string

	// Create a semaphore to limit concurrent WebSocket broadcasts
	broadcastSemaphore := make(chan struct{}, 5) // Reduced from 10 to 5 concurrent broadcasts

	s.logger.WithField("entities_count", len(entities)).Info("Starting entity registration/update process")

	logMemStats(s.logger, "before_registration_loop")

	for i := 0; i < len(entities); i += batchSize {
		end := i + batchSize
		if end > len(entities) {
			end = len(entities)
		}

		batch := entities[i:end]
		s.logger.WithFields(logrus.Fields{
			"batch_start": i,
			"batch_end":   end,
			"batch_size":  len(batch),
		}).Debug("Registering entity batch")

		for j, entity := range batch {
			// Check context cancellation
			select {
			case <-ctx.Done():
				return &SyncResult{
					Source:      source,
					Success:     false,
					Error:       "sync cancelled",
					Duration:    time.Since(startTime),
					ProcessedAt: time.Now(),
				}, ctx.Err()
			default:
			}

			// Check if entity already exists
			existingEntity, err := s.registryManager.GetEntityRegistry().GetEntity(entity.GetID())

			if err != nil {
				// Entity doesn't exist, register it
				if err := s.registryManager.GetEntityRegistry().RegisterEntity(entity); err != nil {
					errMsg := fmt.Sprintf("Failed to register entity %s: %v", entity.GetID(), err)
					errors = append(errors, errMsg)
					s.logger.WithError(err).WithField("entity_id", entity.GetID()).Error("Failed to register entity")
					continue
				}
				registeredCount++

				// Cache the entity in Redis for fast access
				if s.redisCache != nil {
					if err := s.redisCache.SetEntity(ctx, entity.GetID(), entity); err != nil {
						s.logger.WithError(err).WithField("entity_id", entity.GetID()).Warn("Failed to cache entity in Redis")
					}
				}

				// Broadcast new entity added (with semaphore control)
				if s.eventEmitter != nil {
					select {
					case broadcastSemaphore <- struct{}{}:
						go func(entityID string, entityToAdd types.PMAEntity) {
							defer func() {
								<-broadcastSemaphore // Release semaphore
								if r := recover(); r != nil {
									s.logger.WithField("panic", r).Error("WebSocket entity added broadcast panicked")
								}
							}()

							s.eventEmitter.BroadcastPMAEntityAdded(entityToAdd)
						}(entity.GetID(), entity)
					default:
						s.logger.Debug("WebSocket broadcast queue full, skipping entity added broadcast")
					}
				}
			} else {
				// Entity exists, check for conflicts and update
				if s.shouldUpdateEntity(existingEntity, entity) {
					// Store old state for broadcasting
					oldState := existingEntity.GetState()

					if err := s.registryManager.GetEntityRegistry().UpdateEntity(entity); err != nil {
						errMsg := fmt.Sprintf("Failed to update entity %s: %v", entity.GetID(), err)
						errors = append(errors, errMsg)
						s.logger.WithError(err).WithField("entity_id", entity.GetID()).Error("Failed to update entity")
						continue
					}
					updatedCount++

					// Update the entity in Redis cache
					if s.redisCache != nil {
						if err := s.redisCache.SetEntity(ctx, entity.GetID(), entity); err != nil {
							s.logger.WithError(err).WithField("entity_id", entity.GetID()).Warn("Failed to update entity in Redis cache")
						}
					}

					// Broadcast entity state change (with semaphore control)
					if s.eventEmitter != nil && oldState != entity.GetState() {
						select {
						case broadcastSemaphore <- struct{}{}:
							go func(entityID string, oldStateVal, newStateVal types.PMAEntityState) {
								defer func() {
									<-broadcastSemaphore // Release semaphore
									if r := recover(); r != nil {
										s.logger.WithField("panic", r).Error("WebSocket state change broadcast panicked")
									}
								}()

								s.eventEmitter.BroadcastPMAEntityStateChange(
									entityID,
									oldStateVal,
									newStateVal,
									nil, // Pass nil for entityData to reduce memory usage
								)
							}(entity.GetID(), oldState, entity.GetState())
						default:
							s.logger.Debug("WebSocket broadcast queue full, skipping state change broadcast")
						}
					}
				}
			}

			// Log sample entity for the first in batch
			if j == 0 {
				s.logger.WithField("sample_entity_to_register", fmt.Sprintf("%#v", entity)).Info("Sample entity to register")
			}
		}

		// Log memory stats after each batch
		logMemStats(s.logger, fmt.Sprintf("after_registration_batch_%d", i/batchSize))

		// Force garbage collection after each batch to prevent memory accumulation
		runtime.GC()
		time.Sleep(10 * time.Millisecond) // Small delay to allow GC to complete
	}

	logMemStats(s.logger, "after_all_registration_batches")

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
		s.logger.WithFields(logrus.Fields{
			"source":      source,
			"error_count": len(errors),
			"errors":      errors,
		}).Warn("Sync completed with errors")
	}

	s.logger.WithFields(logrus.Fields{
		"source":              source,
		"success":             result.Success,
		"entities_found":      result.EntitiesFound,
		"entities_registered": result.EntitiesRegistered,
		"entities_updated":    result.EntitiesUpdated,
		"duration":            result.Duration,
		"error_count":         len(errors),
	}).Info("Entity sync completed")

	// Broadcast sync completion (non-blocking)
	if s.eventEmitter != nil {
		status := "completed"
		if len(errors) > 0 {
			status = "completed_with_errors"
		}

		go func(statusVal string, durationVal time.Duration) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.WithField("panic", r).Error("WebSocket completion broadcast panicked")
				}
			}()

			s.eventEmitter.BroadcastPMASyncStatus(string(source), statusVal, map[string]interface{}{
				"entities_found":      len(entities),
				"entities_registered": registeredCount,
				"entities_updated":    updatedCount,
				"duration":            durationVal,
				"error_count":         len(errors),
			})
		}(status, time.Since(startTime))
	}

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
	var mutex sync.Mutex

	// Process adapters sequentially to prevent goroutine explosion
	// Only use goroutines if we have multiple adapters and want to process them in parallel
	if len(adapters) > 1 {
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 2) // Limit to 2 concurrent syncs

		for _, adapter := range adapters {
			wg.Add(1)
			go func(adapter types.PMAAdapter) {
				defer wg.Done()

				// Acquire semaphore
				select {
				case semaphore <- struct{}{}:
					defer func() { <-semaphore }()
				case <-ctx.Done():
					return
				}

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
	} else {
		// Single adapter - process sequentially
		for _, adapter := range adapters {
			result, err := s.SyncFromSource(ctx, adapter.GetSourceType())
			if err != nil {
				result = &SyncResult{
					Source:      adapter.GetSourceType(),
					Success:     false,
					Error:       err.Error(),
					ProcessedAt: time.Now(),
				}
			}
			results = append(results, result)
		}
	}

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

	// Parse sync interval with safety checks
	syncInterval, err := time.ParseDuration(s.config.HomeAssistant.Sync.FullSyncInterval)
	if err != nil {
		s.logger.WithError(err).Warn("Invalid sync interval, using default 1 hour")
		syncInterval = 1 * time.Hour
	}

	// Prevent extremely short intervals that could cause memory leaks
	if syncInterval < 5*time.Minute {
		s.logger.WithField("requested_interval", syncInterval).Warn("Sync interval too short, using minimum 5 minutes")
		syncInterval = 5 * time.Minute
	}

	// Prevent extremely long intervals
	if syncInterval > 24*time.Hour {
		s.logger.WithField("requested_interval", syncInterval).Warn("Sync interval too long, using maximum 24 hours")
		syncInterval = 24 * time.Hour
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
	defer func() {
		if r := recover(); r != nil {
			s.logger.WithField("panic", r).Error("Periodic sync loop panicked")
			// Check if we should still be running before restarting
			select {
			case <-s.syncStopChan:
				s.logger.Info("Periodic sync loop stopped after panic")
				return
			default:
				// Only restart if we're still supposed to be running
				s.logger.Info("Restarting periodic sync loop after panic")
				time.Sleep(30 * time.Second)
				// Check again before restarting to prevent goroutine leaks
				select {
				case <-s.syncStopChan:
					s.logger.Info("Periodic sync loop stopped before restart")
					return
				default:
					// Check if sync is still running before restarting
					s.mutex.RLock()
					syncRunning := s.syncRunning
					s.mutex.RUnlock()

					if !syncRunning {
						s.logger.Info("Periodic sync stopped, not restarting")
						return
					}

					// Use a separate goroutine to avoid stack overflow, but with proper cleanup
					go func() {
						defer func() {
							if r := recover(); r != nil {
								s.logger.WithField("panic", r).Error("Restarted periodic sync loop panicked")
							}
						}()
						s.periodicSyncLoop()
					}()
				}
			}
		}
	}()

	for {
		select {
		case <-s.syncStopChan:
			s.logger.Info("Periodic sync loop stopped")
			return
		case <-s.syncTicker.C:
			// Use a separate goroutine for sync to prevent blocking
			go func() {
				defer func() {
					if r := recover(); r != nil {
						s.logger.WithField("panic", r).Error("Periodic sync panicked")
					}
				}()
				s.performPeriodicSync()
			}()
		}
	}
}

// performPeriodicSync executes a full sync from all sources
func (s *UnifiedEntityService) performPeriodicSync() {
	defer func() {
		if r := recover(); r != nil {
			s.logger.WithField("panic", r).Error("Periodic sync panicked")
		}
	}()

	s.logger.Info("Starting periodic sync from all sources")

	// Add timeout to prevent hanging syncs
	ctx, cancel := context.WithTimeout(context.Background(), s.syncTimeout)
	defer cancel()

	// Use semaphore to limit concurrent sync operations
	select {
	case s.syncSemaphore <- struct{}{}:
		defer func() { <-s.syncSemaphore }()
	default:
		s.logger.Warn("Too many concurrent sync operations, skipping this periodic sync")
		return
	}

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
		"total_sources":    len(results),
		"successful_syncs": successCount,
		"total_found":      totalFound,
		"total_registered": totalRegistered,
		"total_updated":    totalUpdated,
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
	// For now, return a default area. This would be enhanced with actual area service integration
	return &types.PMAArea{
		ID:   areaID,
		Name: "Unknown Area",
	}, nil
}

// UpdateEntityState updates an entity's state from an external source and broadcasts the change
func (s *UnifiedEntityService) UpdateEntityState(ctx context.Context, entityID string, newState string, source types.PMASourceType) (types.PMAEntity, error) {
	s.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"new_state": newState,
		"source":    source,
	}).Debug("Updating entity state from external source")

	// Get the entity from Redis cache (no mutex needed, Redis handles concurrency)
	var entity types.PMAEntity
	var exists bool
	var cacheSize int

	if s.redisCache != nil {
		var err error
		entity, err = s.redisCache.GetEntity(ctx, entityID)
		if err != nil {
			exists = false
		} else {
			exists = true
		}

		// Get cache size for debugging
		cacheSize, _ = s.redisCache.GetCacheSize(ctx)
	} else {
		exists = false
		cacheSize = 0
	}

	if !exists {
		s.logger.WithFields(logrus.Fields{
			"entity_id":  entityID,
			"cache_size": cacheSize,
		}).Debug("Entity not found in cache for state update")

		// Return success for unknown entities to avoid flooding logs
		return nil, nil
	}

	// Store old state for comparison (no mutex needed for read-only comparison)
	oldState := entity.GetState()

	// Only proceed if state actually changed
	if oldState == types.PMAEntityState(newState) {
		s.logger.WithFields(logrus.Fields{
			"entity_id": entityID,
			"state":     newState,
		}).Debug("Entity state unchanged, skipping update")
		return entity, nil
	}

	s.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"old_state": oldState,
		"new_state": newState,
	}).Debug("Entity state changed, updating")

	// Update the entity state directly instead of cloning
	// This is more memory efficient
	s.updateEntityStateDirectly(entity, newState)

	// Save the updated entity back to the cache (no mutex needed, Redis handles concurrency)
	if s.redisCache != nil {
		if err := s.redisCache.SetEntity(ctx, entityID, entity); err != nil {
			s.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to save entity to Redis cache")
			return entity, fmt.Errorf("failed to save entity to cache: %w", err)
		}
	}

	// CRITICAL SECTION: Only hold mutex for registry operations
	s.mutex.Lock()
	if err := s.registryManager.GetEntityRegistry().UpdateEntity(entity); err != nil {
		s.mutex.Unlock()
		s.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to update entity in registry")
		// Don't return error here as the cache was already updated
		// This ensures partial success rather than complete failure
	} else {
		s.mutex.Unlock()
		s.logger.WithField("entity_id", entityID).Debug("✅ Entity updated in registry")
	}

	// Enhanced real-time broadcasting for immediate UI updates
	if s.eventEmitter != nil {
		// Immediate broadcast in current goroutine for critical responsiveness
		s.eventEmitter.BroadcastPMAEntityStateChange(entityID, oldState, newState, map[string]interface{}{
			"entity":        entity,
			"change_source": source,
			"timestamp":     time.Now().UTC(),
		})

		s.logger.WithFields(logrus.Fields{
			"entity_id": entityID,
			"old_state": oldState,
			"new_state": newState,
			"source":    source,
		}).Info("📡 Real-time state change broadcast completed")
	}

	return entity, nil
}

// HandleExternalStateChange handles state changes from external sources (physical switches, automations, etc.)
func (s *UnifiedEntityService) HandleExternalStateChange(ctx context.Context, entityID string, newState string, source types.PMASourceType, metadata map[string]interface{}) error {
	s.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"new_state": newState,
		"source":    source,
		"metadata":  metadata,
	}).Info("🔄 Handling external state change")

	// Update the entity state immediately
	entity, err := s.UpdateEntityState(ctx, entityID, newState, source)
	if err != nil {
		s.logger.WithError(err).WithField("entity_id", entityID).Error("Failed to handle external state change")
		return err
	}

	// Additional broadcast with external source context for UI differentiation
	if s.eventEmitter != nil {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					s.logger.WithField("panic", r).Error("Panic during external state change broadcast")
				}
			}()

			broadcastData := map[string]interface{}{
				"entity":           entity,
				"change_source":    source,
				"external_trigger": true,
				"metadata":         metadata,
				"timestamp":        time.Now().UTC(),
			}

			s.eventEmitter.BroadcastPMAEntityStateChange(entityID, "", newState, broadcastData)

			s.logger.WithFields(logrus.Fields{
				"entity_id": entityID,
				"new_state": newState,
				"source":    source,
			}).Info("📡 External state change broadcast sent")
		}()
	}

	return nil
}

// cloneEntity creates a copy of an entity for safe modification
func (s *UnifiedEntityService) cloneEntity(entity types.PMAEntity) types.PMAEntity {
	switch e := entity.(type) {
	case *types.PMASwitchEntity:
		clone := *e
		return &clone
	case *types.PMALightEntity:
		clone := *e
		return &clone
	case *types.PMASensorEntity:
		clone := *e
		return &clone
	default:
		s.logger.WithField("entity_type", fmt.Sprintf("%T", entity)).Warn("Unknown entity type for cloning")
		return entity // Return original if we can't clone
	}
}

// updateGenericEntityState updates the state of a generic entity using reflection
func (s *UnifiedEntityService) updateGenericEntityState(entity types.PMAEntity, newState string) {
	// For generic entities, we'll try to set common fields
	s.logger.WithFields(logrus.Fields{
		"entity_id":   entity.GetID(),
		"entity_type": fmt.Sprintf("%T", entity),
		"new_state":   newState,
	}).Debug("Updating generic entity state")
}

// updateEntityStateDirectly updates the state of an entity directly
func (s *UnifiedEntityService) updateEntityStateDirectly(entity types.PMAEntity, newState string) {
	switch e := entity.(type) {
	case *types.PMASwitchEntity:
		e.State = types.PMAEntityState(newState)
		e.LastUpdated = time.Now()
	case *types.PMALightEntity:
		e.State = types.PMAEntityState(newState)
		e.LastUpdated = time.Now()
	case *types.PMASensorEntity:
		e.State = types.PMAEntityState(newState)
		e.LastUpdated = time.Now()
	default:
		s.logger.WithField("entity_id", entity.GetID()).Warn("Attempted to update state for unknown entity type")
	}
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

// logMemStats logs current memory usage
func logMemStats(logger *logrus.Logger, context string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	logger.WithFields(logrus.Fields{
		"context":        context,
		"alloc_mb":       m.Alloc / 1024 / 1024,
		"total_alloc_mb": m.TotalAlloc / 1024 / 1024,
		"sys_mb":         m.Sys / 1024 / 1024,
		"num_gc":         m.NumGC,
		"goroutines":     runtime.NumGoroutine(),
		"heap_objects":   m.HeapObjects,
	}).Info("[MEMSTATS] Memory usage snapshot")
}
