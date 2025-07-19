package homeassistant

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// HomeAssistantAdapter implements the PMAAdapter interface for HomeAssistant integration
type HomeAssistantAdapter struct {
	id        string
	client    *HAClientWrapper
	converter *EntityConverter
	mapper    *StateMapper
	config    *config.Config
	logger    *logrus.Logger
	connected bool
	lastSync  *time.Time
	metrics   *types.AdapterMetrics
	health    *types.AdapterHealth
	mutex     sync.RWMutex
	startTime time.Time
}

// NewHomeAssistantAdapter creates a new HomeAssistant adapter
func NewHomeAssistantAdapter(config *config.Config, logger *logrus.Logger) *HomeAssistantAdapter {
	adapter := &HomeAssistantAdapter{
		id:        "homeassistant_primary",
		config:    config,
		logger:    logger,
		startTime: time.Now(),
		metrics: &types.AdapterMetrics{
			EntitiesManaged:     0,
			RoomsManaged:        0,
			ActionsExecuted:     0,
			SuccessfulActions:   0,
			FailedActions:       0,
			AverageResponseTime: 0,
			SyncErrors:          0,
			Uptime:              0,
		},
		health: &types.AdapterHealth{
			IsHealthy:       false,
			LastHealthCheck: time.Now(),
			Issues:          []string{},
			ResponseTime:    0,
			ErrorRate:       0.0,
			Details:         make(map[string]interface{}),
		},
	}

	// Initialize components
	adapter.client = NewHAClientWrapper(config, logger)
	adapter.converter = NewEntityConverter(logger)
	adapter.mapper = NewStateMapper(logger)

	return adapter
}

// GetID returns the adapter's unique identifier
func (a *HomeAssistantAdapter) GetID() string {
	return a.id
}

// GetSourceType returns the source type for HomeAssistant
func (a *HomeAssistantAdapter) GetSourceType() types.PMASourceType {
	return types.SourceHomeAssistant
}

// GetName returns the adapter's display name
func (a *HomeAssistantAdapter) GetName() string {
	return "Home Assistant"
}

// GetVersion returns the adapter's version
func (a *HomeAssistantAdapter) GetVersion() string {
	return "1.0.0"
}

// Connect establishes connection to HomeAssistant
func (a *HomeAssistantAdapter) Connect(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.logger.Info("Connecting to Home Assistant...")

	if err := a.client.Connect(ctx); err != nil {
		a.connected = false
		a.updateHealth(false, fmt.Sprintf("Connection failed: %v", err))
		return fmt.Errorf("failed to connect to HomeAssistant: %w", err)
	}

	a.connected = true
	a.updateHealth(true, "Connected successfully")
	a.logger.Info("Successfully connected to Home Assistant")

	return nil
}

// Disconnect closes the connection to HomeAssistant
func (a *HomeAssistantAdapter) Disconnect(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.logger.Info("Disconnecting from Home Assistant...")

	if err := a.client.Disconnect(ctx); err != nil {
		a.logger.WithError(err).Error("Error during disconnect")
		return fmt.Errorf("failed to disconnect from HomeAssistant: %w", err)
	}

	a.connected = false
	a.updateHealth(false, "Disconnected")
	a.logger.Info("Disconnected from Home Assistant")

	return nil
}

// IsConnected returns the connection status
func (a *HomeAssistantAdapter) IsConnected() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.connected
}

// GetStatus returns the current adapter status
func (a *HomeAssistantAdapter) GetStatus() string {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if !a.connected {
		return "disconnected"
	}

	if a.health.IsHealthy {
		return "healthy"
	}

	return "unhealthy"
}

// ConvertEntity converts a single HomeAssistant entity to PMA format
func (a *HomeAssistantAdapter) ConvertEntity(sourceEntity interface{}) (types.PMAEntity, error) {
	haEntity, ok := sourceEntity.(*HAEntity)
	if !ok {
		return nil, fmt.Errorf("invalid entity type: expected *HAEntity, got %T", sourceEntity)
	}

	return a.converter.ConvertToPMAEntity(haEntity)
}

// ConvertEntities converts multiple HomeAssistant entities to PMA format
func (a *HomeAssistantAdapter) ConvertEntities(sourceEntities []interface{}) ([]types.PMAEntity, error) {
	var pmaEntities []types.PMAEntity
	var errors []error

	for _, sourceEntity := range sourceEntities {
		entity, err := a.ConvertEntity(sourceEntity)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		pmaEntities = append(pmaEntities, entity)
	}

	if len(errors) > 0 {
		a.logger.WithField("error_count", len(errors)).Warn("Some entities failed to convert")
	}

	return pmaEntities, nil
}

// ConvertRoom converts a HomeAssistant area to PMA room format
func (a *HomeAssistantAdapter) ConvertRoom(sourceRoom interface{}) (*types.PMARoom, error) {
	haArea, ok := sourceRoom.(*HAArea)
	if !ok {
		return nil, fmt.Errorf("invalid room type: expected *HAArea, got %T", sourceRoom)
	}

	return &types.PMARoom{
		ID:          fmt.Sprintf("ha_room_%s", haArea.ID),
		Name:        haArea.Name,
		Icon:        haArea.Icon,
		Description: getFirstAlias(haArea.Aliases), // Use first alias as description if available
		EntityIDs:   []string{},                    // Will be populated during entity sync
		Metadata: &types.PMAMetadata{
			Source:         types.SourceHomeAssistant,
			SourceEntityID: haArea.ID,
			SourceData: map[string]interface{}{
				"aliases": haArea.Aliases,
			},
			LastSynced:   time.Now(),
			QualityScore: 1.0,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// ConvertArea converts a HomeAssistant area to PMA area format
func (a *HomeAssistantAdapter) ConvertArea(sourceArea interface{}) (*types.PMAArea, error) {
	haArea, ok := sourceArea.(*HAArea)
	if !ok {
		return nil, fmt.Errorf("invalid area type: expected *HAArea, got %T", sourceArea)
	}

	return &types.PMAArea{
		ID:          fmt.Sprintf("ha_area_%s", haArea.ID),
		Name:        haArea.Name,
		Icon:        haArea.Icon,
		Description: getFirstAlias(haArea.Aliases), // Use first alias as description if available
		RoomIDs:     []string{},                    // HA areas don't have sub-rooms
		EntityIDs:   []string{},        // Will be populated during entity sync
		Metadata: &types.PMAMetadata{
			Source:         types.SourceHomeAssistant,
			SourceEntityID: haArea.ID,
			SourceData: map[string]interface{}{
				"aliases": haArea.Aliases,
			},
			LastSynced:   time.Now(),
			QualityScore: 1.0,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// ExecuteAction executes a control action on a HomeAssistant entity
func (a *HomeAssistantAdapter) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	start := time.Now()

	a.mutex.Lock()
	a.metrics.ActionsExecuted++
	a.mutex.Unlock()

	// Validate action
	if action.EntityID == "" || action.Action == "" {
		a.incrementFailedActions()
		return &types.PMAControlResult{
			Success: false,
			Error: &types.PMAError{
				Code:      "INVALID_ACTION",
				Message:   "Entity ID and action are required",
				Source:    "homeassistant_adapter",
				EntityID:  action.EntityID,
				Timestamp: time.Now(),
				Retryable: false,
			},
		}, nil
	}

	// Convert PMA entity ID to HA entity ID
	haEntityID := a.convertPMAEntityIDToHA(action.EntityID)

	// Map action to service call
	domain, service, data, err := a.mapper.MapActionToService(action)
	if err != nil {
		a.incrementFailedActions()
		return &types.PMAControlResult{
			Success: false,
			Error: &types.PMAError{
				Code:      "MAPPING_ERROR",
				Message:   err.Error(),
				Source:    "homeassistant_adapter",
				EntityID:  action.EntityID,
				Timestamp: time.Now(),
				Retryable: false,
			},
		}, nil
	}

	// Execute service call
	err = a.client.CallService(ctx, domain, service, haEntityID, data)
	if err != nil {
		a.incrementFailedActions()
		return &types.PMAControlResult{
			Success: false,
			Error: &types.PMAError{
				Code:      "EXECUTION_ERROR",
				Message:   err.Error(),
				Source:    "homeassistant_adapter",
				EntityID:  action.EntityID,
				Timestamp: time.Now(),
				Retryable: true,
			},
		}, nil
	}

	// Update metrics
	duration := time.Since(start)
	a.mutex.Lock()
	a.metrics.SuccessfulActions++
	a.updateAverageResponseTime(duration)
	a.mutex.Unlock()

	return &types.PMAControlResult{
		Success:     true,
		EntityID:    action.EntityID,
		Action:      action.Action,
		ProcessedAt: time.Now(),
		Duration:    duration,
	}, nil
}

// SyncEntities synchronizes all entities from HomeAssistant
func (a *HomeAssistantAdapter) SyncEntities(ctx context.Context) ([]types.PMAEntity, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("adapter not connected")
	}

	a.logger.Info("Starting entity synchronization from Home Assistant")

	haEntities, err := a.client.GetAllEntities(ctx)
	if err != nil {
		a.mutex.Lock()
		a.metrics.SyncErrors++
		a.mutex.Unlock()
		return nil, fmt.Errorf("failed to fetch entities from HomeAssistant: %w", err)
	}

	var pmaEntities []types.PMAEntity
	for _, haEntity := range haEntities {
		pmaEntity, err := a.converter.ConvertToPMAEntity(haEntity)
		if err != nil {
			a.logger.WithError(err).WithField("entity_id", haEntity.EntityID).Warn("Failed to convert entity")
			continue
		}
		pmaEntities = append(pmaEntities, pmaEntity)
	}

	// Update metrics
	now := time.Now()
	a.mutex.Lock()
	a.lastSync = &now
	a.metrics.EntitiesManaged = len(pmaEntities)
	a.metrics.LastSync = &now
	a.mutex.Unlock()

	a.logger.WithField("entity_count", len(pmaEntities)).Info("Entity synchronization completed")
	return pmaEntities, nil
}

// SyncRooms synchronizes all rooms/areas from HomeAssistant
func (a *HomeAssistantAdapter) SyncRooms(ctx context.Context) ([]*types.PMARoom, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("adapter not connected")
	}

	a.logger.Info("Starting room synchronization from Home Assistant")

	haAreas, err := a.client.GetAllAreas(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch areas from HomeAssistant: %w", err)
	}

	var pmaRooms []*types.PMARoom
	for _, haArea := range haAreas {
		pmaRoom, err := a.ConvertRoom(haArea)
		if err != nil {
			a.logger.WithError(err).WithField("area_id", haArea.ID).Warn("Failed to convert area to room")
			continue
		}
		pmaRooms = append(pmaRooms, pmaRoom)
	}

	// Update metrics
	a.mutex.Lock()
	a.metrics.RoomsManaged = len(pmaRooms)
	a.mutex.Unlock()

	a.logger.WithField("room_count", len(pmaRooms)).Info("Room synchronization completed")
	return pmaRooms, nil
}

// GetLastSyncTime returns the last synchronization time
func (a *HomeAssistantAdapter) GetLastSyncTime() *time.Time {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.lastSync
}

// GetSupportedEntityTypes returns the entity types supported by this adapter
func (a *HomeAssistantAdapter) GetSupportedEntityTypes() []types.PMAEntityType {
	return []types.PMAEntityType{
		types.EntityTypeLight,
		types.EntityTypeSwitch,
		types.EntityTypeSensor,
		types.EntityTypeClimate,
		types.EntityTypeCover,
		types.EntityTypeCamera,
		types.EntityTypeLock,
		types.EntityTypeFan,
		types.EntityTypeMediaPlayer,
		types.EntityTypeBinarySensor,
	}
}

// GetSupportedCapabilities returns the capabilities supported by this adapter
func (a *HomeAssistantAdapter) GetSupportedCapabilities() []types.PMACapability {
	return []types.PMACapability{
		types.CapabilityDimmable,
		types.CapabilityColorable,
		types.CapabilityTemperature,
		types.CapabilityHumidity,
		types.CapabilityPosition,
		types.CapabilityVolume,
		types.CapabilityBrightness,
		types.CapabilityMotion,
		types.CapabilityRecording,
		types.CapabilityStreaming,
		types.CapabilityNotification,
		types.CapabilityBattery,
		types.CapabilityConnectivity,
	}
}

// SupportsRealtime returns true as HomeAssistant supports real-time updates via WebSocket
func (a *HomeAssistantAdapter) SupportsRealtime() bool {
	return true
}

// GetHealth returns the current health status of the adapter
func (a *HomeAssistantAdapter) GetHealth() *types.AdapterHealth {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	// Update uptime
	health := *a.health
	health.Details["uptime"] = time.Since(a.startTime).String()

	return &health
}

// GetMetrics returns the current metrics for the adapter
func (a *HomeAssistantAdapter) GetMetrics() *types.AdapterMetrics {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	// Update uptime
	metrics := *a.metrics
	metrics.Uptime = time.Since(a.startTime)

	return &metrics
}

// Helper methods

func (a *HomeAssistantAdapter) updateHealth(healthy bool, message string) {
	a.health.IsHealthy = healthy
	a.health.LastHealthCheck = time.Now()
	if !healthy && message != "" {
		a.health.Issues = []string{message}
	} else if healthy {
		a.health.Issues = []string{}
	}
}

func (a *HomeAssistantAdapter) incrementFailedActions() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.metrics.FailedActions++

	// Update error rate
	total := a.metrics.SuccessfulActions + a.metrics.FailedActions
	if total > 0 {
		a.health.ErrorRate = float64(a.metrics.FailedActions) / float64(total)
	}
}

func (a *HomeAssistantAdapter) updateAverageResponseTime(duration time.Duration) {
	// Simple moving average calculation
	if a.metrics.AverageResponseTime == 0 {
		a.metrics.AverageResponseTime = duration
	} else {
		a.metrics.AverageResponseTime = (a.metrics.AverageResponseTime + duration) / 2
	}
}

func (a *HomeAssistantAdapter) convertPMAEntityIDToHA(pmaEntityID string) string {
	// Remove "ha_" prefix if present
	if len(pmaEntityID) > 3 && pmaEntityID[:3] == "ha_" {
		return pmaEntityID[3:]
	}
	return pmaEntityID
}

func getFirstAlias(aliases []string) string {
	if len(aliases) > 0 {
		return aliases[0]
	}
	return ""
}
