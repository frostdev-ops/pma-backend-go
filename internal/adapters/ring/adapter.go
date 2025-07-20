package ring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/config"
	"github.com/frostdev-ops/pma-backend-go/internal/core/devices"
	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// RingAdapter implements the PMAAdapter interface for Ring devices
type RingAdapter struct {
	client            *RingClient
	devices           map[string]devices.Device
	entities          map[string]types.PMAEntity
	eventCallbacks    []func(devices.DeviceEvent)
	logger            *logrus.Logger
	config            RingAdapterConfig
	mutex             sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
	pollTicker        *time.Ticker
	eventTicker       *time.Ticker
	connected         bool
	lastEventID       int
	lastSyncTime      time.Time
	startTime         time.Time
	actionsExecuted   int
	successfulActions int
	failedActions     int
	syncErrors        int
}

// RingAdapterConfig holds configuration for the Ring adapter
type RingAdapterConfig struct {
	Credentials   RingCredentials `json:"credentials"`
	PollInterval  time.Duration   `json:"poll_interval"`
	EventInterval time.Duration   `json:"event_interval"`
	EventLimit    int             `json:"event_limit"`
	AutoReconnect bool            `json:"auto_reconnect"`
}

// NewRingAdapter creates a new Ring adapter
func NewRingAdapter(config RingAdapterConfig, fullConfig *config.Config, logger *logrus.Logger) *RingAdapter {
	ctx, cancel := context.WithCancel(context.Background())

	// Set defaults
	if config.PollInterval == 0 {
		config.PollInterval = 5 * time.Minute
	}
	if config.EventInterval == 0 {
		config.EventInterval = 30 * time.Second
	}
	if config.EventLimit == 0 {
		config.EventLimit = 20
	}

	return &RingAdapter{
		client:         NewRingClient(logger, fullConfig),
		devices:        make(map[string]devices.Device),
		entities:       make(map[string]types.PMAEntity),
		eventCallbacks: make([]func(devices.DeviceEvent), 0),
		logger:         logger,
		config:         config,
		ctx:            ctx,
		cancel:         cancel,
		pollTicker:     time.NewTicker(config.PollInterval),
		eventTicker:    time.NewTicker(config.EventInterval),
		connected:      false,
		startTime:      time.Now(),
	}
}

// GetType returns the adapter type
func (a *RingAdapter) GetType() string {
	return "ring"
}

// Connect establishes connection to Ring API
func (a *RingAdapter) Connect(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.logger.Info("Connecting to Ring API...")

	if err := a.client.Authenticate(ctx, a.config.Credentials); err != nil {
		a.logger.WithError(err).Error("Failed to authenticate with Ring")
		return fmt.Errorf("ring authentication failed: %w", err)
	}

	a.connected = true
	a.logger.Info("Successfully connected to Ring API")

	// Start background polling
	go a.pollDevices()
	go a.pollEvents()

	return nil
}

// Disconnect closes the connection to Ring API
func (a *RingAdapter) Disconnect(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.logger.Info("Disconnecting from Ring API...")

	a.cancel()
	a.pollTicker.Stop()
	a.eventTicker.Stop()
	a.connected = false

	a.logger.Info("Disconnected from Ring API")
	return nil
}

// Discover discovers Ring devices
func (a *RingAdapter) Discover(ctx context.Context) ([]devices.Device, error) {
	if !a.connected {
		return nil, devices.ErrAdapterNotConnected
	}

	a.logger.Info("Discovering Ring devices...")

	ringDevices, err := a.client.GetDevices(ctx)
	if err != nil {
		a.logger.WithError(err).Error("Failed to discover Ring devices")
		return nil, fmt.Errorf("device discovery failed: %w", err)
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	discoveredDevices := make([]devices.Device, 0)

	for _, ringDevice := range ringDevices {
		var device devices.Device

		// Create appropriate device type based on Ring device kind
		switch ringDevice.Kind {
		case "doorbell", "doorbell_v3", "doorbell_v4", "doorbell_v5":
			device = NewRingDoorbell(a.client, &ringDevice)
		case "stickup_cam", "stickup_cam_v3", "stickup_cam_battery", "floodlight_cam":
			device = NewRingCamera(a.client, &ringDevice)
		case "chime", "chime_v2":
			device = NewRingChime(a.client, &ringDevice)
		default:
			a.logger.WithField("kind", ringDevice.Kind).Warn("Unknown Ring device type, treating as camera")
			device = NewRingCamera(a.client, &ringDevice)
		}

		a.devices[device.GetID()] = device
		discoveredDevices = append(discoveredDevices, device)

		a.logger.WithFields(logrus.Fields{
			"device_id":   device.GetID(),
			"device_name": device.GetName(),
			"device_type": device.GetType(),
			"ring_kind":   ringDevice.Kind,
		}).Info("Discovered Ring device")
	}

	a.logger.WithField("count", len(discoveredDevices)).Info("Ring device discovery completed")
	return discoveredDevices, nil
}

// Subscribe registers an event callback
func (a *RingAdapter) Subscribe(callback func(devices.DeviceEvent)) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.eventCallbacks = append(a.eventCallbacks, callback)
	a.logger.Debug("Added Ring event subscriber")
	return nil
}

// Unsubscribe removes all event callbacks
func (a *RingAdapter) Unsubscribe() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.eventCallbacks = make([]func(devices.DeviceEvent), 0)
	a.logger.Debug("Removed all Ring event subscribers")
	return nil
}

// GetDevice retrieves a specific device by ID
func (a *RingAdapter) GetDevice(deviceID string) (devices.Device, error) {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	device, exists := a.devices[deviceID]
	if !exists {
		return nil, fmt.Errorf("%w: %s", devices.ErrDeviceNotFound, deviceID)
	}

	return device, nil
}

// GetDevices returns all discovered devices
func (a *RingAdapter) GetDevices() []devices.Device {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	deviceList := make([]devices.Device, 0, len(a.devices))
	for _, device := range a.devices {
		deviceList = append(deviceList, device)
	}

	return deviceList
}

// IsConnected returns the connection status
func (a *RingAdapter) IsConnected() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.connected && a.client.IsAuthenticated()
}

// HealthCheck verifies the adapter health
func (a *RingAdapter) HealthCheck() error {
	if !a.IsConnected() {
		if a.config.AutoReconnect {
			a.logger.Info("Health check failed, attempting to reconnect...")
			return a.Connect(a.ctx)
		}
		return devices.ErrAdapterNotConnected
	}

	// Test API connectivity by getting device list
	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	_, err := a.client.GetDevices(ctx)
	if err != nil {
		a.logger.WithError(err).Error("Ring health check failed")
		return err
	}

	return nil
}

// pollDevices periodically updates device states
func (a *RingAdapter) pollDevices() {
	defer func() {
		if r := recover(); r != nil {
			a.logger.WithField("panic", r).Error("Ring device polling panicked")
		}
	}()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-a.pollTicker.C:
			a.updateDeviceStates()
		}
	}
}

// pollEvents periodically checks for new Ring events
func (a *RingAdapter) pollEvents() {
	defer func() {
		if r := recover(); r != nil {
			a.logger.WithField("panic", r).Error("Ring event polling panicked")
		}
	}()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-a.eventTicker.C:
			a.checkForEvents()
		}
	}
}

// updateDeviceStates refreshes device states from Ring API
func (a *RingAdapter) updateDeviceStates() {
	if !a.IsConnected() {
		return
	}

	ctx, cancel := context.WithTimeout(a.ctx, 60*time.Second)
	defer cancel()

	ringDevices, err := a.client.GetDevices(ctx)
	if err != nil {
		a.logger.WithError(err).Error("Failed to update Ring device states")
		return
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Update existing devices
	for _, ringDevice := range ringDevices {
		for _, device := range a.devices {
			// Match by Ring device ID
			var ringDeviceID int

			switch d := device.(type) {
			case *RingDoorbell:
				ringDeviceID = d.GetRingDeviceID()
				if ringDeviceID == ringDevice.ID {
					d.UpdateFromRingData(&ringDevice)
				}
			case *RingCamera:
				ringDeviceID = d.GetRingDeviceID()
				if ringDeviceID == ringDevice.ID {
					d.UpdateFromRingData(&ringDevice)
				}
			case *RingChime:
				ringDeviceID = d.GetRingDeviceID()
				if ringDeviceID == ringDevice.ID {
					d.UpdateFromRingData(&ringDevice)
				}
			}
		}
	}

	a.logger.Debug("Updated Ring device states")
}

// checkForEvents polls for new Ring events
func (a *RingAdapter) checkForEvents() {
	if !a.IsConnected() {
		return
	}

	ctx, cancel := context.WithTimeout(a.ctx, 30*time.Second)
	defer cancel()

	events, err := a.client.GetEvents(ctx, a.config.EventLimit)
	if err != nil {
		a.logger.WithError(err).Error("Failed to get Ring events")
		return
	}

	// Process new events (events with ID greater than lastEventID)
	newEvents := make([]RingEvent, 0)
	for _, event := range events {
		if event.ID > a.lastEventID {
			newEvents = append(newEvents, event)
			if event.ID > a.lastEventID {
				a.lastEventID = event.ID
			}
		}
	}

	// Process events from newest to oldest (reverse order since we want chronological)
	for i := len(newEvents) - 1; i >= 0; i-- {
		event := newEvents[i]
		a.processRingEvent(event)
	}

	if len(newEvents) > 0 {
		a.logger.WithField("count", len(newEvents)).Debug("Processed new Ring events")
	}
}

// processRingEvent converts a Ring event to a device event and emits it
func (a *RingAdapter) processRingEvent(ringEvent RingEvent) {
	deviceID := fmt.Sprintf("ring-doorbell-%d", ringEvent.DoorbotID)

	// Check if it's a camera instead
	a.mutex.RLock()
	if device, exists := a.devices[deviceID]; !exists {
		// Try camera ID format
		deviceID = fmt.Sprintf("ring-camera-%d", ringEvent.DoorbotID)
		if _, exists := a.devices[deviceID]; !exists {
			a.mutex.RUnlock()
			a.logger.WithField("doorbot_id", ringEvent.DoorbotID).Debug("Event for unknown device")
			return
		}
	} else {
		_ = device // Use the device variable to avoid unused variable error
	}
	a.mutex.RUnlock()

	// Determine event type
	var eventType devices.EventType
	eventData := map[string]interface{}{
		"ring_event_id": ringEvent.ID,
		"state":         ringEvent.State,
		"protocol":      ringEvent.Protocol,
		"created_at":    ringEvent.CreatedAt,
		"updated_at":    ringEvent.UpdatedAt,
	}

	switch ringEvent.Kind {
	case "ding":
		eventType = devices.EventTypeDoorbellPressed
	case "motion":
		eventType = devices.EventTypeMotionDetected
		eventData["motion"] = ringEvent.MotionDetection
	default:
		eventType = devices.EventTypeStateChanged
	}

	// Add additional event data
	if ringEvent.SnapshotURL != "" {
		eventData["snapshot_url"] = ringEvent.SnapshotURL
	}
	if ringEvent.RecordingURL != "" {
		eventData["recording_url"] = ringEvent.RecordingURL
	}
	if ringEvent.StreamingURL != "" {
		eventData["streaming_url"] = ringEvent.StreamingURL
	}
	if ringEvent.AnsweredAt != nil {
		eventData["answered_at"] = ringEvent.AnsweredAt
	}

	deviceEvent := devices.DeviceEvent{
		DeviceID:    deviceID,
		AdapterType: "ring",
		EventType:   eventType,
		Data:        eventData,
		Timestamp:   ringEvent.CreatedAt,
		Source:      "ring_api",
	}

	a.emitEvent(deviceEvent)
}

// emitEvent sends an event to all subscribers
func (a *RingAdapter) emitEvent(event devices.DeviceEvent) {
	a.mutex.RLock()
	callbacks := make([]func(devices.DeviceEvent), len(a.eventCallbacks))
	copy(callbacks, a.eventCallbacks)
	a.mutex.RUnlock()

	for _, callback := range callbacks {
		go func(cb func(devices.DeviceEvent)) {
			defer func() {
				if r := recover(); r != nil {
					a.logger.WithField("panic", r).Error("Ring event callback panicked")
				}
			}()
			cb(event)
		}(callback)
	}
}

// UpdateConfig updates the adapter configuration
func (a *RingAdapter) UpdateConfig(config RingAdapterConfig) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.config = config

	// Update timers if connected
	if a.connected {
		a.pollTicker.Reset(config.PollInterval)
		a.eventTicker.Reset(config.EventInterval)
	}

	a.logger.Info("Ring adapter configuration updated")
	return nil
}

// GetRefreshToken returns the current refresh token for persistence
func (a *RingAdapter) GetRefreshToken() string {
	return a.client.GetRefreshToken()
}

// ========================================
// PMAAdapter Interface Implementation
// ========================================

// GetID returns the unique identifier for this adapter instance
func (a *RingAdapter) GetID() string {
	return fmt.Sprintf("ring_%s", a.config.Credentials.Email)
}

// GetSourceType returns the source type for Ring
func (a *RingAdapter) GetSourceType() types.PMASourceType {
	return types.SourceRing
}

// GetName returns the adapter name
func (a *RingAdapter) GetName() string {
	return "Ring Security Adapter"
}

// GetVersion returns the adapter version
func (a *RingAdapter) GetVersion() string {
	return "1.0.0"
}

// IsConnected method already exists in the adapter

// GetStatus returns the adapter status
func (a *RingAdapter) GetStatus() string {
	if a.IsConnected() {
		return "connected"
	}
	return "disconnected"
}

// ConvertEntity converts a source entity to PMA entity
func (a *RingAdapter) ConvertEntity(sourceEntity interface{}) (types.PMAEntity, error) {
	switch device := sourceEntity.(type) {
	case *RingDoorbell:
		return a.convertDoorbellToPMACamera(device)
	case *RingCamera:
		return a.convertCameraToPMACamera(device)
	case *RingChime:
		return a.convertChimeToPMADevice(device)
	default:
		return nil, fmt.Errorf("unsupported Ring device type: %T", sourceEntity)
	}
}

// ConvertEntities converts multiple source entities to PMA entities
func (a *RingAdapter) ConvertEntities(sourceEntities []interface{}) ([]types.PMAEntity, error) {
	pmaEntities := make([]types.PMAEntity, 0, len(sourceEntities))

	for _, sourceEntity := range sourceEntities {
		entity, err := a.ConvertEntity(sourceEntity)
		if err != nil {
			a.logger.WithError(err).Warnf("Failed to convert entity: %v", sourceEntity)
			continue
		}
		pmaEntities = append(pmaEntities, entity)
	}

	return pmaEntities, nil
}

// ConvertRoom converts a Ring location to PMA room (Ring doesn't have room concept)
func (a *RingAdapter) ConvertRoom(sourceRoom interface{}) (*types.PMARoom, error) {
	// Ring doesn't have room/area concepts in the traditional sense
	// Locations could be mapped to rooms, but it's not a direct mapping
	return nil, fmt.Errorf("room conversion not supported for Ring devices")
}

// ConvertArea converts a Ring location to PMA area
func (a *RingAdapter) ConvertArea(sourceArea interface{}) (*types.PMAArea, error) {
	// Ring doesn't have area concepts in the traditional sense
	// Could potentially use device locations as areas
	return nil, fmt.Errorf("area conversion not supported for Ring devices")
}

// ExecuteAction executes control actions on Ring devices
func (a *RingAdapter) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	ringAction, params, err := a.mapPMAActionToRingAction(action)
	if err != nil {
		return &types.PMAControlResult{
			Success:     false,
			EntityID:    action.EntityID,
			Action:      action.Action,
			ProcessedAt: time.Now(),
			Error: &types.PMAError{
				Code:     "INVALID_ACTION",
				Message:  err.Error(),
				Source:   "ring",
				EntityID: action.EntityID,
			},
		}, nil
	}

	return a.executeRingAction(ctx, action.EntityID, ringAction, params)
}

// SyncEntities synchronizes entities from Ring API
func (a *RingAdapter) SyncEntities(ctx context.Context) ([]types.PMAEntity, error) {
	if !a.connected {
		return nil, fmt.Errorf("adapter not connected")
	}

	// Use existing Discover method to get Ring devices
	devices, err := a.Discover(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover Ring devices: %w", err)
	}

	// Convert to interface slice for ConvertEntities
	sourceEntities := make([]interface{}, len(devices))
	for i, device := range devices {
		sourceEntities[i] = device
	}

	// Convert to PMA entities
	pmaEntities, err := a.ConvertEntities(sourceEntities)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Ring devices: %w", err)
	}

	a.lastSyncTime = time.Now()
	return pmaEntities, nil
}

// SyncRooms synchronizes rooms from Ring (not supported)
func (a *RingAdapter) SyncRooms(ctx context.Context) ([]*types.PMARoom, error) {
	// Ring doesn't have room concepts
	return []*types.PMARoom{}, nil
}

// GetLastSyncTime returns the last synchronization time
func (a *RingAdapter) GetLastSyncTime() *time.Time {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	if a.lastSyncTime.IsZero() {
		return nil
	}
	return &a.lastSyncTime
}

// GetSupportedEntityTypes returns entity types supported by Ring adapter
func (a *RingAdapter) GetSupportedEntityTypes() []types.PMAEntityType {
	return []types.PMAEntityType{
		types.EntityTypeCamera,
		types.EntityTypeDevice,
	}
}

// GetSupportedCapabilities returns capabilities supported by Ring devices
func (a *RingAdapter) GetSupportedCapabilities() []types.PMACapability {
	return []types.PMACapability{
		types.CapabilityStreaming,
		types.CapabilityRecording,
		types.CapabilityMotion,
		types.CapabilityNotification,
		types.CapabilityBattery,
	}
}

// SupportsRealtime returns whether Ring supports real-time updates
func (a *RingAdapter) SupportsRealtime() bool {
	return true // Ring supports webhooks and polling
}

// GetHealth returns adapter health information
func (a *RingAdapter) GetHealth() *types.AdapterHealth {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	issues := []string{}
	if !a.connected {
		issues = append(issues, "Not connected to Ring API")
	}

	if !a.client.IsAuthenticated() {
		issues = append(issues, "Ring authentication invalid")
	}

	// Calculate response time by measuring a quick API call
	start := time.Now()
	_, err := a.client.GetDevices(context.Background())
	responseTime := time.Since(start)

	if err != nil {
		issues = append(issues, fmt.Sprintf("API error: %v", err))
	}

	return &types.AdapterHealth{
		IsHealthy:       len(issues) == 0,
		LastHealthCheck: time.Now(),
		Issues:          issues,
		ResponseTime:    responseTime,
		ErrorRate:       a.calculateErrorRate(),
		Details: map[string]interface{}{
			"connected":     a.connected,
			"authenticated": a.client.IsAuthenticated(),
			"device_count":  len(a.devices),
		},
	}
}

// GetMetrics returns adapter performance metrics
func (a *RingAdapter) GetMetrics() *types.AdapterMetrics {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	var lastSync *time.Time
	if !a.lastSyncTime.IsZero() {
		lastSync = &a.lastSyncTime
	}

	return &types.AdapterMetrics{
		EntitiesManaged:     len(a.devices),
		RoomsManaged:        0, // Ring doesn't have rooms
		ActionsExecuted:     int64(a.actionsExecuted),
		SuccessfulActions:   int64(a.successfulActions),
		FailedActions:       int64(a.failedActions),
		AverageResponseTime: a.calculateAverageResponseTime(),
		LastSync:            lastSync,
		SyncErrors:          a.syncErrors,
		Uptime:              time.Since(a.startTime),
	}
}

// Helper methods for metrics calculation
func (a *RingAdapter) calculateErrorRate() float64 {
	if a.actionsExecuted == 0 {
		return 0.0
	}
	return float64(a.failedActions) / float64(a.actionsExecuted)
}

func (a *RingAdapter) calculateAverageResponseTime() time.Duration {
	// This would need to be implemented with actual response time tracking
	// For now, return a default value
	return 500 * time.Millisecond
}
