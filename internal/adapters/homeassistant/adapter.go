package homeassistant

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// HomeAssistantAdapter implements the PMAAdapter interface for HomeAssistant integration
type HomeAssistantAdapter struct {
	id           string
	client       *HAClientWrapper
	converter    *EntityConverter
	mapper       *StateMapper
	config       *config.Config
	logger       *logrus.Logger
	connected    bool
	lastSync     *time.Time
	metrics      *types.AdapterMetrics
	health       *types.AdapterHealth
	mutex        sync.RWMutex
	startTime    time.Time
	stopChan     chan bool                                 // Channel to stop event processing
	eventHandler func(entityID, oldState, newState string) // Handler for state changes
}

// NewHomeAssistantAdapter creates a new HomeAssistant adapter
func NewHomeAssistantAdapter(config *config.Config, logger *logrus.Logger) *HomeAssistantAdapter {
	adapter := &HomeAssistantAdapter{
		id:        "homeassistant_primary",
		config:    config,
		logger:    logger,
		startTime: time.Now(),
		stopChan:  make(chan bool, 1),
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
	a.logger.Info("🔵 ADAPTER Connect method starting...")

	a.logger.Info("🔗 ADAPTER connecting to client (no mutex held)...")

	// Connect to client without holding mutex to prevent deadlock
	if err := a.client.Connect(ctx); err != nil {
		a.mutex.Lock()
		a.connected = false
		a.updateHealth(false, fmt.Sprintf("Connection failed: %v", err))
		a.mutex.Unlock()
		a.logger.WithError(err).Error("❌ ADAPTER client connection failed")
		return fmt.Errorf("failed to connect to HomeAssistant: %w", err)
	}

	a.logger.Info("✅ ADAPTER client connection successful")

	// Now safely update state with mutex
	a.mutex.Lock()
	a.connected = true
	a.updateHealth(true, "Connected successfully")
	a.mutex.Unlock()

	a.logger.Info("🎉 ADAPTER Successfully connected to Home Assistant")

	// Start WebSocket event processing
	a.logger.Info("🚀 About to start WebSocket event processing goroutine")
	go a.processWebSocketEvents()
	a.logger.Info("✅ WebSocket event processing goroutine launched")

	return nil
}

// Disconnect closes the connection to HomeAssistant
func (a *HomeAssistantAdapter) Disconnect(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Stop event processing
	select {
	case a.stopChan <- true:
	default:
	}

	if a.client != nil {
		if err := a.client.Disconnect(); err != nil {
			a.logger.WithError(err).Warn("Error disconnecting client")
		}
	}

	a.connected = false
	a.updateHealth(false, "Disconnected")
	a.logger.Info("Disconnected from HomeAssistant")

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

// ExecuteAction executes a control action on a HomeAssistant entity
func (a *HomeAssistantAdapter) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	start := time.Now()

	// Use non-blocking metrics update to prevent deadlock with WebSocket event processing
	go func() {
		a.mutex.Lock()
		a.metrics.ActionsExecuted++
		a.mutex.Unlock()
	}()

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
	a.logger.WithField("pma_entity_id", action.EntityID).Info("🔄 About to convert PMA entity ID to HA format")
	haEntityID := a.convertPMAEntityIDToHA(action.EntityID)
	a.logger.WithFields(logrus.Fields{
		"pma_entity_id": action.EntityID,
		"ha_entity_id":  haEntityID,
	}).Info("✂️ PMA entity ID converted to HA format")

	// Map action to service call
	a.logger.WithFields(logrus.Fields{
		"action":        action.Action,
		"pma_entity_id": action.EntityID,
		"ha_entity_id":  haEntityID,
	}).Info("🗺️ About to map action to Home Assistant service")
	domain, service, data, err := a.mapper.MapActionToService(action)
	a.logger.WithFields(logrus.Fields{
		"action":        action.Action,
		"pma_entity_id": action.EntityID,
		"ha_entity_id":  haEntityID,
		"domain":        domain,
		"service":       service,
		"error":         err,
	}).Info("🎯 Action mapped to Home Assistant service")
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

	// Execute service call with detailed logging
	a.logger.WithFields(logrus.Fields{
		"domain":     domain,
		"service":    service,
		"ha_entity":  haEntityID,
		"pma_entity": action.EntityID,
	}).Info("🔥 About to call Home Assistant service")

	err = a.client.CallService(ctx, domain, service, haEntityID, data)

	a.logger.WithFields(logrus.Fields{
		"domain":     domain,
		"service":    service,
		"ha_entity":  haEntityID,
		"pma_entity": action.EntityID,
		"error":      err,
	}).Info("🏁 Home Assistant service call completed")

	if err != nil {
		a.logger.WithError(err).WithFields(logrus.Fields{
			"domain":     domain,
			"service":    service,
			"ha_entity":  haEntityID,
			"pma_entity": action.EntityID,
		}).Error("❌ Home Assistant service call failed")
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

	// Update metrics (non-blocking to prevent deadlock)
	duration := time.Since(start)
	go func() {
		a.mutex.Lock()
		a.metrics.SuccessfulActions++
		a.updateAverageResponseTime(duration)
		a.mutex.Unlock()
	}()

	// Fetch the updated entity state from Home Assistant
	a.logger.WithField("ha_entity", haEntityID).Info("🔍 Fetching updated entity state after action")

	// Create a fresh context for the state fetch with a short timeout
	fetchCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Immediate state determination based on action type
	var predictedState types.PMAEntityState
	switch action.Action {
	case "turn_on":
		predictedState = types.PMAEntityState("on")
	case "turn_off":
		predictedState = types.PMAEntityState("off")
	case "toggle":
		// For toggle, we need to fetch current state first, but we can predict immediately
		if currentEntities, err := a.client.GetAllEntitiesHTTPOnly(fetchCtx); err == nil {
			for _, entity := range currentEntities {
				if entity.EntityID == haEntityID {
					if entity.State == "on" {
						predictedState = types.PMAEntityState("off")
					} else {
						predictedState = types.PMAEntityState("on")
					}
					break
				}
			}
		}
		// If we couldn't fetch current state, assume it worked and predict based on most common case
		if predictedState == "" {
			predictedState = types.PMAEntityState("on")
		}
	default:
		// For other actions, try to determine state from action parameters
		predictedState = types.PMAEntityState("on") // Safe default
	}

	a.logger.WithFields(logrus.Fields{
		"ha_entity":       haEntityID,
		"action":          action.Action,
		"predicted_state": predictedState,
	}).Info("🎯 Predicted new state based on action")

	// For immediate response, attributes will be nil since we're not fetching them
	var updatedAttributes map[string]interface{}

	// Async verification of actual state (for accuracy but not blocking response)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.logger.WithField("panic", r).Error("Panic during async state verification")
			}
		}()

		// Brief wait for HA to process the change
		time.Sleep(200 * time.Millisecond)

		verifyCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if verifiedEntities, err := a.client.GetAllEntitiesHTTPOnly(verifyCtx); err == nil {
			for _, entity := range verifiedEntities {
				if entity.EntityID == haEntityID {
					actualState := types.PMAEntityState(entity.State)
					if actualState != predictedState {
						a.logger.WithFields(logrus.Fields{
							"ha_entity":       haEntityID,
							"predicted_state": predictedState,
							"actual_state":    actualState,
						}).Warn("⚠️ State prediction mismatch - triggering correction")

						// Trigger state correction via event handler if available
						if a.eventHandler != nil {
							a.eventHandler(action.EntityID, string(predictedState), string(actualState))
						}
					} else {
						a.logger.WithFields(logrus.Fields{
							"ha_entity":    haEntityID,
							"actual_state": actualState,
						}).Info("✅ State prediction confirmed by verification")
					}
					break
				}
			}
		} else {
			a.logger.WithError(err).Warn("Failed to verify entity state - prediction may be inaccurate")
		}
	}()

	// Return immediately with predicted state for optimal responsiveness
	return &types.PMAControlResult{
		Success:     true,
		EntityID:    action.EntityID,
		Action:      action.Action,
		NewState:    predictedState,
		Attributes:  updatedAttributes,
		ProcessedAt: time.Now(),
		Duration:    duration,
	}, nil
}

// RefreshEntityState refreshes a specific entity's state from Home Assistant
func (a *HomeAssistantAdapter) RefreshEntityState(ctx context.Context, entityID string) error {
	a.logger.WithField("entity_id", entityID).Info("🔄 Refreshing entity state from Home Assistant")

	// Convert PMA entity ID to HA entity ID
	haEntityID := a.convertPMAEntityIDToHA(entityID)

	// Fetch current state from Home Assistant
	entities, err := a.client.GetAllEntitiesHTTPOnly(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch entities from Home Assistant: %w", err)
	}

	// Find and update the specific entity
	for _, entity := range entities {
		if entity.EntityID == haEntityID {
			// Trigger state update through event handler if available
			if a.eventHandler != nil {
				a.eventHandler(entityID, "", entity.State)
				a.logger.WithFields(logrus.Fields{
					"entity_id": entityID,
					"ha_entity": haEntityID,
					"new_state": entity.State,
				}).Info("✅ Entity state refreshed from Home Assistant")
			}
			return nil
		}
	}

	return fmt.Errorf("entity %s not found in Home Assistant", haEntityID)
}

// SyncEntities synchronizes all entities from HomeAssistant
func (a *HomeAssistantAdapter) SyncEntities(ctx context.Context) ([]types.PMAEntity, error) {
	a.logger.Info("Starting entity synchronization from Home Assistant")

	// Use HTTP-only mode for syncing to avoid WebSocket connection issues
	// This bypasses any potential WebSocket conflicts and ensures reliable syncing
	a.logger.Debug("Calling GetAllEntitiesHTTPOnly...")
	logMemStats(a.logger, "before_GetAllEntitiesHTTPOnly")
	haEntities, err := a.client.GetAllEntitiesHTTPOnly(ctx)
	if err != nil {
		a.mutex.Lock()
		a.metrics.SyncErrors++
		a.mutex.Unlock()
		return nil, fmt.Errorf("failed to fetch entities from HomeAssistant via HTTP: %w", err)
	}
	a.logger.WithField("raw_entity_count", len(haEntities)).Info("Successfully fetched entities, starting batch conversion...")
	if len(haEntities) > 0 {
		a.logger.WithField("sample_ha_entity", fmt.Sprintf("%#v", haEntities[0])).Info("Sample HA entity after fetch")
	}
	logMemStats(a.logger, "after_GetAllEntitiesHTTPOnly")

	// Process entities in batches to reduce memory usage
	const batchSize = 10
	var pmaEntities []types.PMAEntity
	totalEntities := len(haEntities)

	for i := 0; i < totalEntities; i += batchSize {
		end := i + batchSize
		if end > totalEntities {
			end = totalEntities
		}

		batch := haEntities[i:end]
		a.logger.WithFields(logrus.Fields{
			"batch_start": i,
			"batch_end":   end,
			"batch_size":  len(batch),
			"total":       totalEntities,
		}).Debug("Processing entity batch...")

		// Process this batch
		for j, haEntity := range batch {
			pmaEntity, err := a.converter.ConvertToPMAEntity(haEntity)
			if err != nil {
				a.logger.WithError(err).WithField("entity_id", haEntity.EntityID).Warn("Failed to convert entity")
				continue
			}
			pmaEntities = append(pmaEntities, pmaEntity)

			// Log progress every 5 entities within a batch
			if j == 0 {
				a.logger.WithField("sample_pma_entity", fmt.Sprintf("%#v", pmaEntity)).Info("Sample PMA entity after conversion")
			}
		}

		logMemStats(a.logger, fmt.Sprintf("after_batch_%d", i/batchSize))

		// Force garbage collection after each batch to prevent memory buildup
		runtime.GC()

		// Small delay to allow GC to complete
		time.Sleep(10 * time.Millisecond)
	}

	a.logger.WithField("converted_entity_count", len(pmaEntities)).Info("Entity conversion completed, updating metrics...")
	if len(pmaEntities) > 0 {
		a.logger.WithField("sample_final_pma_entity", fmt.Sprintf("%#v", pmaEntities[0])).Info("Sample PMA entity after all conversion")
	}
	logMemStats(a.logger, "after_all_batches")

	// Update metrics (using non-blocking approach to prevent deadlock)
	now := time.Now()
	a.logger.Info("About to update metrics without mutex lock...")

	// These are simple assignments that don't need mutex protection
	// since they're just setting primitive values atomically
	a.lastSync = &now
	a.metrics.EntitiesManaged = len(pmaEntities)
	a.metrics.LastSync = &now

	a.logger.Info("Metrics updated successfully without mutex")

	a.logger.WithField("entity_count", len(pmaEntities)).Info("Entity synchronization completed")

	a.logger.WithFields(logrus.Fields{
		"entity_count": len(pmaEntities),
		"sample_entity_ids": func() []string {
			var ids []string
			for i, e := range pmaEntities {
				if i < 3 {
					ids = append(ids, e.GetID())
				}
			}
			return ids
		}(),
	}).Info("About to return entities from SyncEntities")

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

// SetEventHandler sets the callback function for WebSocket state change events
func (a *HomeAssistantAdapter) SetEventHandler(handler func(entityID, oldState, newState string)) {
	a.eventHandler = handler
	a.logger.Info("Event handler registered for WebSocket state changes")
}

// processWebSocketEvents processes WebSocket events from Home Assistant
func (a *HomeAssistantAdapter) processWebSocketEvents() {
	a.logger.Info("🎯 Starting WebSocket event processing goroutine")

	eventChan := a.client.GetStateChangeEvents()
	a.logger.WithField("event_chan", eventChan != nil).Info("📡 Got event channel from client")

	for {
		select {
		case event := <-eventChan:
			a.logger.WithFields(logrus.Fields{
				"entity_id":  event.Data.EntityID,
				"event_type": event.EventType,
			}).Debug("Received WebSocket state change event")

			a.processStateChangeEvent(event)

		case <-a.stopChan:
			a.logger.Info("Stopping WebSocket event processing")
			return
		}
	}
}

// processStateChangeEvent processes a single state change event
func (a *HomeAssistantAdapter) processStateChangeEvent(event HAStateChangeEvent) {
	if event.EventType != "state_changed" {
		return
	}

	entityID := event.Data.EntityID
	if entityID == "" {
		return
	}

	// Extract old and new states
	var oldState, newState string

	if event.Data.NewState != nil {
		if state, ok := event.Data.NewState["state"].(string); ok {
			newState = state
		}
	}

	if event.Data.OldState != nil {
		if state, ok := event.Data.OldState["state"].(string); ok {
			oldState = state
		}
	}

	a.logger.WithFields(logrus.Fields{
		"entity_id": entityID,
		"old_state": oldState,
		"new_state": newState,
	}).Info("Processing Home Assistant state change")

	// Call the registered event handler if available
	if a.eventHandler != nil {
		// Convert HA entity ID to PMA entity ID format
		pmaEntityID := a.convertHAEntityIDToPMA(entityID)
		a.eventHandler(pmaEntityID, oldState, newState)
	} else {
		a.logger.Warn("No event handler registered for state changes")
	}
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

func (a *HomeAssistantAdapter) convertHAEntityIDToPMA(haEntityID string) string {
	// Add "ha_" prefix if not present
	if len(haEntityID) > 0 && haEntityID[:2] != "ha_" {
		return "ha_" + haEntityID
	}
	return haEntityID
}

func getFirstAlias(aliases []string) string {
	if len(aliases) > 0 {
		return aliases[0]
	}
	return ""
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
	}).Info("[MEMSTATS] Memory usage snapshot")
}
