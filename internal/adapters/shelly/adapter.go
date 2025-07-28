package shelly

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/frostdev-ops/pma-backend-go/pkg/debug"
	"github.com/sirupsen/logrus"
)

// ShellyAdapter implements the PMAAdapter interface for Shelly devices
type ShellyAdapter struct {
	client            *ShellyClient
	logger            *logrus.Logger
	debugLogger       *debug.ServiceLogger // For detailed debug logging
	config            ShellyAdapterConfig
	mutex             sync.RWMutex
	connected         bool
	lastSyncTime      time.Time
	startTime         time.Time
	actionsExecuted   int
	successfulActions int
	failedActions     int
	syncErrors        int
	discoveredDevices map[string]*EnhancedShellyDevice
	lastDiscoverySync time.Time
	ctx               context.Context
}

// ShellyAdapterConfig holds configuration for the Shelly adapter
type ShellyAdapterConfig struct {
	Enabled                bool          `json:"enabled"`
	DiscoveryInterval      time.Duration `json:"discovery_interval"`
	DiscoveryTimeout       time.Duration `json:"discovery_timeout"`
	NetworkScanEnabled     bool          `json:"network_scan_enabled"`
	NetworkScanRanges      []string      `json:"network_scan_ranges"`
	AutoWiFiSetup          bool          `json:"auto_wifi_setup"`
	DefaultUsername        string        `json:"default_username"`
	DefaultPassword        string        `json:"default_password"`
	PollInterval           time.Duration `json:"poll_interval"`
	MaxDevices             int           `json:"max_devices"`
	HealthCheckInterval    time.Duration `json:"health_check_interval"`
	RetryAttempts          int           `json:"retry_attempts"`
	RetryBackoff           time.Duration `json:"retry_backoff"`
	EnableGen1Support      bool          `json:"enable_gen1_support"`
	EnableGen2Support      bool          `json:"enable_gen2_support"`
	DiscoveryBroadcastAddr string        `json:"discovery_broadcast_addr"`

	// Auto-detection configuration
	AutoDetectSubnets         bool     `json:"auto_detect_subnets"`
	AutoDetectInterfaceFilter []string `json:"auto_detect_interface_filter"`
	ExcludeLoopback           bool     `json:"exclude_loopback"`
	ExcludeDockerInterfaces   bool     `json:"exclude_docker_interfaces"`
	MinSubnetSize             int      `json:"min_subnet_size"`
}

// NewShellyAdapter creates a new Shelly adapter with enhanced discovery
func NewShellyAdapter(config ShellyAdapterConfig, logger *logrus.Logger, debugLogger *debug.DebugLogger) *ShellyAdapter {
	// Set defaults
	if config.PollInterval == 0 {
		config.PollInterval = 30 * time.Second
	}
	if config.DiscoveryInterval == 0 {
		config.DiscoveryInterval = 5 * time.Minute
	}
	if config.DiscoveryTimeout == 0 {
		config.DiscoveryTimeout = 30 * time.Second
	}
	if config.MaxDevices == 0 {
		config.MaxDevices = 100
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}
	if config.RetryBackoff == 0 {
		config.RetryBackoff = 10 * time.Second
	}
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 60 * time.Second
	}

	// Create discovery configuration
	discoveryConfig := DiscoveryConfig{
		Enabled:            config.Enabled,
		Interval:           config.DiscoveryInterval,
		Timeout:            config.DiscoveryTimeout,
		NetworkScanEnabled: config.NetworkScanEnabled,
		NetworkScanRanges:  config.NetworkScanRanges,
		MaxDevices:         config.MaxDevices,
		RetryAttempts:      config.RetryAttempts,
		RetryBackoff:       config.RetryBackoff,
		EnableGen1Support:  config.EnableGen1Support,
		EnableGen2Support:  config.EnableGen2Support,
		AutoWiFiSetup:      config.AutoWiFiSetup,
		DefaultUsername:    config.DefaultUsername,
		DefaultPassword:    config.DefaultPassword,
	}

	return &ShellyAdapter{
		client:            NewEnhancedShellyClient(discoveryConfig, logger),
		logger:            logger,
		debugLogger:       debug.NewServiceLogger("shelly_adapter", debugLogger),
		config:            config,
		startTime:         time.Now(),
		discoveredDevices: make(map[string]*EnhancedShellyDevice),
		ctx:               context.Background(),
	}
}

// ========================================
// PMAAdapter Interface Implementation
// ========================================

// GetID returns the unique identifier for this adapter instance
func (a *ShellyAdapter) GetID() string {
	return "shelly_adapter_enhanced"
}

// GetSourceType returns the source type for Shelly
func (a *ShellyAdapter) GetSourceType() types.PMASourceType {
	return types.SourceShelly
}

// GetName returns the adapter name
func (a *ShellyAdapter) GetName() string {
	return "Enhanced Shelly IoT Adapter"
}

// GetVersion returns the adapter version
func (a *ShellyAdapter) GetVersion() string {
	return "2.0.0"
}

// GetClient returns the underlying Shelly client
func (a *ShellyAdapter) GetClient() *ShellyClient {
	return a.client
}

// Connect establishes connection and starts discovery
func (a *ShellyAdapter) Connect(ctx context.Context) error {
	defer a.debugLogger.LogCall(ctx, "Connect")()
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.config.Enabled {
		a.debugLogger.LogInfo(ctx, "Connect", "Shelly adapter is disabled", nil)
		a.logger.Info("Shelly adapter is disabled")
		return nil
	}

	a.logger.Info("Starting enhanced Shelly adapter with automatic discovery...")

	// Start the enhanced discovery process
	if err := a.client.StartDiscovery(ctx); err != nil {
		a.debugLogger.LogError(ctx, "Connect", err, map[string]interface{}{"step": "StartDiscovery"})
		return fmt.Errorf("failed to start Shelly discovery: %w", err)
	}

	a.connected = true
	a.ctx = ctx
	a.debugLogger.LogInfo(ctx, "Connect", "Adapter connected successfully", nil)
	a.logger.Info("Enhanced Shelly adapter connected successfully")

	// Start device synchronization in background
	go a.runDeviceSync()

	return nil
}

// Disconnect stops discovery and closes connections
func (a *ShellyAdapter) Disconnect(ctx context.Context) error {
	defer a.debugLogger.LogCall(ctx, "Disconnect")()
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.client.StopDiscovery()
	a.connected = false
	a.debugLogger.LogInfo(ctx, "Disconnect", "Adapter disconnected", nil)
	a.logger.Info("Enhanced Shelly adapter disconnected")
	return nil
}

// IsConnected returns connection status
func (a *ShellyAdapter) IsConnected() bool {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.connected
}

// GetStatus returns the adapter status
func (a *ShellyAdapter) GetStatus() string {
	if a.IsConnected() {
		return "connected"
	}
	return "disconnected"
}

// GetEntities returns all discovered Shelly devices as PMA entities
func (a *ShellyAdapter) GetEntities(ctx context.Context) ([]types.PMAEntity, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("adapter not connected")
	}

	// Get online devices from the client
	devices := a.client.GetOnlineDevices()

	entities := make([]types.PMAEntity, 0, len(devices))
	for _, device := range devices {
		entity, err := a.convertDeviceToPMAEntity(device)
		if err != nil {
			a.logger.WithError(err).Warnf("Failed to convert device %s to PMA entity", device.ID)
			continue
		}
		entities = append(entities, entity)
	}

	a.logger.Debugf("Retrieved %d Shelly entities from %d online devices", len(entities), len(devices))
	return entities, nil
}

// GetEntityByID returns a specific entity by ID
func (a *ShellyAdapter) GetEntityByID(ctx context.Context, entityID string) (types.PMAEntity, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("adapter not connected")
	}

	// Find device by ID
	devices := a.client.GetDiscoveredDevices()
	for _, device := range devices {
		if device.ID == entityID {
			return a.convertDeviceToPMAEntity(device)
		}
	}

	return nil, fmt.Errorf("entity not found: %s", entityID)
}

// SyncEntities synchronizes and returns all Shelly devices as PMA entities
func (a *ShellyAdapter) SyncEntities(ctx context.Context) ([]types.PMAEntity, error) {
	defer a.debugLogger.LogCall(ctx, "SyncEntities")()
	if !a.IsConnected() {
		err := fmt.Errorf("adapter not connected")
		a.debugLogger.LogError(ctx, "SyncEntities", err, nil)
		return nil, err
	}

	a.logger.Debug("Synchronizing Shelly entities...")

	// Update device status for all online devices
	devices := a.client.GetOnlineDevices()
	a.debugLogger.LogData(ctx, "SyncEntities", "Online devices to sync", devices)
	for _, device := range devices {
		if err := a.updateDeviceStatus(ctx, device); err != nil {
			a.debugLogger.LogError(ctx, "SyncEntities", err, map[string]interface{}{"device_id": device.ID, "step": "updateDeviceStatus"})
			a.logger.WithError(err).Warnf("Failed to update status for device %s", device.ID)
			a.syncErrors++
		}
	}

	// Convert devices to PMA entities
	entities := make([]types.PMAEntity, 0, len(devices))
	for _, device := range devices {
		entity, err := a.convertDeviceToPMAEntity(device)
		if err != nil {
			a.debugLogger.LogError(ctx, "SyncEntities", err, map[string]interface{}{"device_id": device.ID, "step": "convertDeviceToPMAEntity"})
			a.logger.WithError(err).Warnf("Failed to convert device %s to PMA entity", device.ID)
			continue
		}
		entities = append(entities, entity)
	}

	a.lastSyncTime = time.Now()
	a.debugLogger.LogInfo(ctx, "SyncEntities", "Synchronization completed", map[string]interface{}{"entity_count": len(entities), "device_count": len(devices)})
	a.logger.Debugf("Synchronized %d Shelly devices", len(devices))
	return entities, nil
}

// ExecuteAction executes control actions on Shelly devices
func (a *ShellyAdapter) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	defer a.debugLogger.LogCall(ctx, "ExecuteAction", action)()
	if !a.IsConnected() {
		err := fmt.Errorf("adapter not connected")
		a.debugLogger.LogError(ctx, "ExecuteAction", err, map[string]interface{}{"action": action})
		return &types.PMAControlResult{
			Success:     false,
			EntityID:    action.EntityID,
			Action:      action.Action,
			ProcessedAt: time.Now(),
			Error: &types.PMAError{
				Code:     "ADAPTER_DISCONNECTED",
				Message:  "Shelly adapter not connected",
				Source:   "shelly",
				EntityID: action.EntityID,
			},
		}, nil
	}

	// Find the device
	device := a.client.GetDeviceByIP(action.EntityID)
	if device == nil {
		// Try to find by device ID
		devices := a.client.GetDiscoveredDevices()
		for _, d := range devices {
			if d.ID == action.EntityID {
				device = d
				break
			}
		}
	}
	a.debugLogger.LogData(ctx, "ExecuteAction", "Device found for action", device)

	if device == nil {
		err := fmt.Errorf("device not found")
		a.debugLogger.LogError(ctx, "ExecuteAction", err, map[string]interface{}{"entity_id": action.EntityID})
		return &types.PMAControlResult{
			Success:     false,
			EntityID:    action.EntityID,
			Action:      action.Action,
			ProcessedAt: time.Now(),
			Error: &types.PMAError{
				Code:     "DEVICE_NOT_FOUND",
				Message:  "Device not found",
				Source:   "shelly",
				EntityID: action.EntityID,
			},
		}, nil
	}

	startTime := time.Now()
	a.actionsExecuted++

	// Execute action based on device generation and type
	var err error
	switch action.Action {
	case "turn_on", "turn_off", "toggle":
		err = a.executeRelayAction(ctx, device, action)
	case "set_brightness", "set_color", "set_light":
		err = a.executeLightAction(ctx, device, action)
	default:
		err = fmt.Errorf("unsupported action: %s", action.Action)
	}
	a.debugLogger.LogInfo(ctx, "ExecuteAction", "Action execution result", map[string]interface{}{"action": action.Action, "error": err})

	if err != nil {
		a.failedActions++
		a.debugLogger.LogError(ctx, "ExecuteAction", err, map[string]interface{}{"action": action})
		return &types.PMAControlResult{
			Success:     false,
			EntityID:    action.EntityID,
			Action:      action.Action,
			ProcessedAt: time.Now(),
			Duration:    time.Since(startTime),
			Error: &types.PMAError{
				Code:     "ACTION_FAILED",
				Message:  err.Error(),
				Source:   "shelly",
				EntityID: action.EntityID,
			},
		}, nil
	}

	a.successfulActions++

	// Immediate state prediction for optimal responsiveness
	var predictedState types.PMAEntityState
	switch action.Action {
	case "turn_on":
		predictedState = types.PMAEntityState("on")
	case "turn_off":
		predictedState = types.PMAEntityState("off")
	case "toggle":
		// Predict toggle state based on current device status
		currentState := a.getCurrentDeviceState(device)
		if currentState == "on" {
			predictedState = types.PMAEntityState("off")
		} else {
			predictedState = types.PMAEntityState("on")
		}
	default:
		// For brightness/color changes, assume device is on
		predictedState = types.PMAEntityState("on")
	}

	// Async device status update for verification (non-blocking)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				a.logger.WithField("panic", r).Error("Panic during Shelly device status update")
			}
		}()

		// Brief wait for device to process the change
		time.Sleep(150 * time.Millisecond)

		if err := a.updateDeviceStatus(ctx, device); err != nil {
			a.debugLogger.LogError(ctx, "ExecuteAction", err, map[string]interface{}{"device_id": device.ID, "step": "updateDeviceStatus_after_action"})
			a.logger.WithError(err).Warnf("Failed to update device status after action")
		} else {
			a.logger.WithFields(logrus.Fields{
				"device_id": device.ID,
				"entity_id": action.EntityID,
				"action":    action.Action,
			}).Info("✅ Shelly device status updated after action")
		}
	}()

	result := &types.PMAControlResult{
		Success:     true,
		EntityID:    action.EntityID,
		Action:      action.Action,
		NewState:    predictedState,
		ProcessedAt: time.Now(),
		Duration:    time.Since(startTime),
	}
	a.debugLogger.LogData(ctx, "ExecuteAction", "Action result", result)
	return result, nil
}

// GetAdapterInfo returns adapter information and statistics
func (a *ShellyAdapter) GetAdapterStats() map[string]interface{} {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	stats := a.client.GetDiscoveryStats()

	return map[string]interface{}{
		"id":                 a.GetID(),
		"name":               a.GetName(),
		"version":            a.GetVersion(),
		"source_type":        a.GetSourceType(),
		"connected":          a.connected,
		"last_sync":          a.lastSyncTime,
		"start_time":         a.startTime,
		"device_count":       stats["total_devices"].(int),
		"online_devices":     stats["online_devices"].(int),
		"actions_executed":   a.actionsExecuted,
		"successful_actions": a.successfulActions,
		"failed_actions":     a.failedActions,
		"sync_errors":        a.syncErrors,
		"gen1_devices":       stats["gen1_devices"].(int),
		"gen2_devices":       stats["gen2_devices"].(int),
		"discovery_running":  stats["discovery_running"].(bool),
		"last_discovery":     stats["last_discovery"].(time.Time),
	}
}

// ConvertEntity converts a Shelly device to PMA entity
func (a *ShellyAdapter) ConvertEntity(source interface{}) (types.PMAEntity, error) {
	device, ok := source.(*EnhancedShellyDevice)
	if !ok {
		return nil, fmt.Errorf("invalid source type for Shelly adapter")
	}

	return a.convertDeviceToPMAEntity(device)
}

// ConvertEntities converts multiple Shelly devices to PMA entities
func (a *ShellyAdapter) ConvertEntities(sources []interface{}) ([]types.PMAEntity, error) {
	entities := make([]types.PMAEntity, 0, len(sources))

	for _, source := range sources {
		entity, err := a.ConvertEntity(source)
		if err != nil {
			a.logger.WithError(err).Warnf("Failed to convert Shelly device to entity")
			continue
		}
		entities = append(entities, entity)
	}

	return entities, nil
}

// ConvertRoom converts room data (Shelly devices don't have native room structures to convert)
func (a *ShellyAdapter) ConvertRoom(source interface{}) (*types.PMARoom, error) {
	return nil, fmt.Errorf("room conversion not supported - Shelly devices don't have native room structures")
}

// ConvertArea converts area data (Shelly devices don't have native area structures to convert)
func (a *ShellyAdapter) ConvertArea(source interface{}) (*types.PMAArea, error) {
	return nil, fmt.Errorf("area conversion not supported - Shelly devices don't have native area structures")
}

// GetLastSyncTime returns the last synchronization time
func (a *ShellyAdapter) GetLastSyncTime() *time.Time {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	if a.lastSyncTime.IsZero() {
		return nil
	}
	return &a.lastSyncTime
}

// GetSupportedCapabilities returns capabilities supported by Shelly devices
func (a *ShellyAdapter) GetSupportedCapabilities() []types.PMACapability {
	return []types.PMACapability{
		types.CapabilityDimmable,
		types.CapabilityColorable,
		types.CapabilityTemperature,
		types.CapabilityHumidity,
		types.CapabilityBrightness,
	}
}

// GetSupportedEntityTypes returns entity types supported by Shelly adapter
func (a *ShellyAdapter) GetSupportedEntityTypes() []types.PMAEntityType {
	return []types.PMAEntityType{
		types.EntityTypeSwitch,
		types.EntityTypeLight,
		types.EntityTypeSensor,
		types.EntityTypeCover,
	}
}

// SupportsRealtime returns whether Shelly supports real-time updates
func (a *ShellyAdapter) SupportsRealtime() bool {
	return false // Shelly uses HTTP polling
}

// SyncRooms synchronizes rooms from Shelly (not supported)
func (a *ShellyAdapter) SyncRooms(ctx context.Context) ([]*types.PMARoom, error) {
	return []*types.PMARoom{}, nil
}

// GetMetrics returns adapter performance metrics
func (a *ShellyAdapter) GetMetrics() *types.AdapterMetrics {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	stats := a.client.GetDiscoveryStats()

	var lastSync *time.Time
	if !a.lastSyncTime.IsZero() {
		lastSync = &a.lastSyncTime
	}

	return &types.AdapterMetrics{
		EntitiesManaged:     stats["total_devices"].(int),
		RoomsManaged:        0, // Shelly doesn't manage rooms
		ActionsExecuted:     int64(a.actionsExecuted),
		SuccessfulActions:   int64(a.successfulActions),
		FailedActions:       int64(a.failedActions),
		AverageResponseTime: 200 * time.Millisecond, // Typical HTTP response
		LastSync:            lastSync,
		SyncErrors:          a.syncErrors,
		Uptime:              time.Since(a.startTime),
	}
}

// GetHealth returns the adapter health status
func (a *ShellyAdapter) GetHealth() *types.AdapterHealth {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	issues := []string{}
	if !a.connected {
		issues = append(issues, "Adapter not connected")
	}

	stats := a.client.GetDiscoveryStats()
	onlineDevices := stats["online_devices"].(int)
	totalDevices := stats["total_devices"].(int)

	if totalDevices > 0 && onlineDevices < totalDevices {
		issues = append(issues, fmt.Sprintf("%d of %d devices offline", totalDevices-onlineDevices, totalDevices))
	}

	errorRate := 0.0
	if a.actionsExecuted > 0 {
		errorRate = float64(a.failedActions) / float64(a.actionsExecuted)
	}

	if errorRate > 0.1 { // More than 10% error rate
		issues = append(issues, fmt.Sprintf("High error rate: %.1f%%", errorRate*100))
	}

	return &types.AdapterHealth{
		IsHealthy:       len(issues) == 0,
		LastHealthCheck: time.Now(),
		Issues:          issues,
		ResponseTime:    200 * time.Millisecond, // Typical HTTP response time
		ErrorRate:       errorRate,
		Details: map[string]interface{}{
			"connected":         a.connected,
			"total_devices":     totalDevices,
			"online_devices":    onlineDevices,
			"discovery_running": stats["discovery_running"].(bool),
		},
	}
}

// ========================================
// Internal Methods
// ========================================

// runDeviceSync runs continuous device synchronization
func (s *ShellyAdapter) runDeviceSync() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("Stopping device sync")
			return
		case <-ticker.C:
			s.logger.Debug("Running periodic device sync")
			if err := s.RefreshDevices(s.ctx); err != nil {
				s.logger.WithError(err).Error("Failed to refresh devices during periodic sync")
			}
		}
	}
}

// syncDiscoveredDevices synchronizes with the discovery engine
func (a *ShellyAdapter) syncDiscoveredDevices(ctx context.Context) error {
	devices := a.client.GetDiscoveredDevices()

	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Update our local device cache
	for _, device := range devices {
		a.discoveredDevices[device.ID] = device
	}

	a.lastDiscoverySync = time.Now()
	a.logger.Debugf("Synchronized %d discovered devices", len(devices))
	return nil
}

// updateDeviceStatus updates the status of a specific device
func (a *ShellyAdapter) updateDeviceStatus(ctx context.Context, device *EnhancedShellyDevice) error {
	// The enhanced client handles status updates internally
	// We just need to refresh our view if needed
	return nil
}

// convertDeviceToPMAEntity converts a Shelly device to appropriate PMA entity
func (a *ShellyAdapter) convertDeviceToPMAEntity(device *EnhancedShellyDevice) (types.PMAEntity, error) {
	// Analyze device capabilities to determine entity type
	switch device.Type {
	case "relay", "switch", "switch_pm":
		return a.convertToSwitchEntity(device)
	case "light", "dimmer", "rgbw", "bulb":
		return a.convertToLightEntity(device)
	case "sensor", "ht", "temperature", "humidity":
		return a.convertToSensorEntity(device)
	case "cover", "roller":
		return a.convertToCoverEntity(device)
	default:
		// Try to determine from device model or capabilities
		if device.Model != "" {
			if containsAny(device.Model, []string{"1PM", "2PM", "relay", "switch"}) {
				return a.convertToSwitchEntity(device)
			}
			if containsAny(device.Model, []string{"bulb", "RGBW", "dimmer", "light"}) {
				return a.convertToLightEntity(device)
			}
			if containsAny(device.Model, []string{"H&T", "sensor", "temp"}) {
				return a.convertToSensorEntity(device)
			}
			if containsAny(device.Model, []string{"2.5", "roller", "cover"}) {
				return a.convertToCoverEntity(device)
			}
		}

		// Default to switch if we can't determine type
		return a.convertToSwitchEntity(device)
	}
}

// convertToSwitchEntity converts Shelly device to PMA switch entity
func (a *ShellyAdapter) convertToSwitchEntity(device *EnhancedShellyDevice) (types.PMAEntity, error) {
	// Determine state based on device generation and status
	var isOn bool
	var powerConsumption float64

	if device.Status != nil {
		switch device.Generation {
		case Gen1:
			if status, ok := device.Status.(map[string]interface{}); ok {
				if relays, ok := status["relays"].([]interface{}); ok && len(relays) > 0 {
					if relay, ok := relays[0].(map[string]interface{}); ok {
						if ison, ok := relay["ison"].(bool); ok {
							isOn = ison
						}
					}
				}
				if meters, ok := status["meters"].([]interface{}); ok && len(meters) > 0 {
					if meter, ok := meters[0].(map[string]interface{}); ok {
						if power, ok := meter["power"].(float64); ok {
							powerConsumption = power
						}
					}
				}
			}
		case Gen2, Gen3, Gen4:
			if status, ok := device.Status.(map[string]interface{}); ok {
				if switchStatus, ok := status["switch:0"].(map[string]interface{}); ok {
					if output, ok := switchStatus["output"].(bool); ok {
						isOn = output
					}
					if apower, ok := switchStatus["apower"].(float64); ok {
						powerConsumption = apower
					}
				}
			}
		}
	}

	attributes := map[string]interface{}{
		"device_generation": int(device.Generation),
		"model":             device.Model,
		"firmware_version":  device.FirmwareVersion,
		"ip_address":        device.IP,
		"mac_address":       device.MAC,
		"wifi_mode":         device.WiFiMode,
		"wifi_ssid":         device.WiFiSSID,
		"power":             powerConsumption,
		"discovery_method":  device.DiscoveryMethod,
		"first_seen":        device.FirstSeen,
		"last_seen":         device.LastSeen,
		"supported_methods": device.SupportedMethods,
	}

	// Add auth information
	if device.AuthEnabled {
		attributes["auth_enabled"] = true
	}

	// Create metadata
	metadata := &types.PMAMetadata{
		Source:         types.SourceShelly,
		SourceEntityID: device.ID,
		SourceData:     attributes,
		LastSynced:     device.LastSeen,
		QualityScore:   0.90,
	}

	entity := &types.PMASwitchEntity{
		PMABaseEntity: &types.PMABaseEntity{
			ID:           device.ID,
			Type:         types.EntityTypeSwitch,
			FriendlyName: device.Name,
			Icon:         "mdi:toggle-switch",
			State:        getBoolEntityState(isOn),
			Attributes:   attributes,
			LastUpdated:  device.LastSeen,
			Available:    device.IsOnline,
			Metadata:     metadata,
		},
	}

	return entity, nil
}

// convertToLightEntity converts Shelly device to PMA light entity
func (a *ShellyAdapter) convertToLightEntity(device *EnhancedShellyDevice) (types.PMAEntity, error) {
	var isOn bool
	var brightness int
	var colorMode string
	var rgb []int
	var colorTemp int

	if device.Status != nil {
		switch device.Generation {
		case Gen1:
			if status, ok := device.Status.(map[string]interface{}); ok {
				if lights, ok := status["lights"].([]interface{}); ok && len(lights) > 0 {
					if light, ok := lights[0].(map[string]interface{}); ok {
						if ison, ok := light["ison"].(bool); ok {
							isOn = ison
						}
						if bright, ok := light["brightness"].(float64); ok {
							brightness = int(bright)
						}
						if mode, ok := light["mode"].(string); ok {
							colorMode = mode
						}
					}
				}
			}
		case Gen2, Gen3, Gen4:
			if status, ok := device.Status.(map[string]interface{}); ok {
				if lightStatus, ok := status["light:0"].(map[string]interface{}); ok {
					if output, ok := lightStatus["output"].(bool); ok {
						isOn = output
					}
					if bright, ok := lightStatus["brightness"].(float64); ok {
						brightness = int(bright)
					}
				}
			}
		}
	}

	attributes := map[string]interface{}{
		"device_generation": int(device.Generation),
		"model":             device.Model,
		"firmware_version":  device.FirmwareVersion,
		"ip_address":        device.IP,
		"mac_address":       device.MAC,
		"wifi_mode":         device.WiFiMode,
		"wifi_ssid":         device.WiFiSSID,
		"color_mode":        colorMode,
		"discovery_method":  device.DiscoveryMethod,
		"first_seen":        device.FirstSeen,
		"last_seen":         device.LastSeen,
		"supported_methods": device.SupportedMethods,
	}

	if len(rgb) > 0 {
		attributes["rgb_color"] = rgb
	}
	if colorTemp > 0 {
		attributes["color_temp"] = colorTemp
	}

	supportedFeatures := []string{
		"turn_on",
		"turn_off",
		"toggle",
	}

	// Add brightness support if the device supports it
	if brightness > 0 || containsAny(device.Model, []string{"dimmer", "RGBW", "bulb"}) {
		supportedFeatures = append(supportedFeatures, "set_brightness")
	}

	// Add color support for RGBW devices
	if containsAny(device.Model, []string{"RGBW", "bulb"}) {
		supportedFeatures = append(supportedFeatures, "set_color", "set_color_temp")
	}

	// Create metadata
	metadata := &types.PMAMetadata{
		Source:         types.SourceShelly,
		SourceEntityID: device.ID,
		SourceData:     attributes,
		LastSynced:     device.LastSeen,
		QualityScore:   0.90,
	}

	entity := &types.PMALightEntity{
		PMABaseEntity: &types.PMABaseEntity{
			ID:           device.ID,
			Type:         types.EntityTypeLight,
			FriendlyName: device.Name,
			Icon:         "mdi:lightbulb",
			State:        getBoolEntityState(isOn),
			Attributes:   attributes,
			LastUpdated:  device.LastSeen,
			Available:    device.IsOnline,
			Metadata:     metadata,
		},
	}

	return entity, nil
}

// convertToSensorEntity converts Shelly device to PMA sensor entity
func (a *ShellyAdapter) convertToSensorEntity(device *EnhancedShellyDevice) (types.PMAEntity, error) {
	var value float64
	var unit string
	var sensorType string

	// Determine primary sensor type and value
	if device.Status != nil {
		switch device.Generation {
		case Gen1:
			if status, ok := device.Status.(map[string]interface{}); ok {
				if temp, ok := status["tmp"].(map[string]interface{}); ok {
					if tC, ok := temp["tC"].(float64); ok {
						value = tC
						unit = "°C"
						sensorType = "temperature"
					}
				} else if humid, ok := status["hum"].(map[string]interface{}); ok {
					if relHumidity, ok := humid["value"].(float64); ok {
						value = relHumidity
						unit = "%"
						sensorType = "humidity"
					}
				}
			}
		case Gen2, Gen3, Gen4:
			if status, ok := device.Status.(map[string]interface{}); ok {
				if tempStatus, ok := status["temperature:0"].(map[string]interface{}); ok {
					if tC, ok := tempStatus["tC"].(float64); ok {
						value = tC
						unit = "°C"
						sensorType = "temperature"
					}
				} else if humStatus, ok := status["humidity:0"].(map[string]interface{}); ok {
					if rh, ok := humStatus["rh"].(float64); ok {
						value = rh
						unit = "%"
						sensorType = "humidity"
					}
				}
			}
		}
	}

	// Default to temperature if not determined
	if sensorType == "" {
		sensorType = "temperature"
		unit = "°C"
	}

	attributes := map[string]interface{}{
		"device_generation": int(device.Generation),
		"model":             device.Model,
		"firmware_version":  device.FirmwareVersion,
		"ip_address":        device.IP,
		"mac_address":       device.MAC,
		"wifi_mode":         device.WiFiMode,
		"wifi_ssid":         device.WiFiSSID,
		"sensor_type":       sensorType,
		"value":             value,
		"unit":              unit,
		"discovery_method":  device.DiscoveryMethod,
		"first_seen":        device.FirstSeen,
		"last_seen":         device.LastSeen,
		"supported_methods": device.SupportedMethods,
	}

	// Create metadata
	metadata := &types.PMAMetadata{
		Source:         types.SourceShelly,
		SourceEntityID: device.ID,
		SourceData:     attributes,
		LastSynced:     device.LastSeen,
		QualityScore:   0.90,
	}

	entity := &types.PMASensorEntity{
		PMABaseEntity: &types.PMABaseEntity{
			ID:           device.ID,
			Type:         types.EntityTypeSensor,
			FriendlyName: device.Name,
			Icon:         "mdi:thermometer",
			State:        types.StateActive,
			Attributes:   attributes,
			LastUpdated:  device.LastSeen,
			Available:    device.IsOnline,
			Metadata:     metadata,
		},
	}

	return entity, nil
}

// convertToCoverEntity converts Shelly device to PMA cover entity
func (a *ShellyAdapter) convertToCoverEntity(device *EnhancedShellyDevice) (types.PMAEntity, error) {
	var state string
	var position int

	if device.Status != nil {
		switch device.Generation {
		case Gen1:
			if status, ok := device.Status.(map[string]interface{}); ok {
				if rollers, ok := status["rollers"].([]interface{}); ok && len(rollers) > 0 {
					if roller, ok := rollers[0].(map[string]interface{}); ok {
						if rollerState, ok := roller["state"].(string); ok {
							state = rollerState
						}
						if pos, ok := roller["current_pos"].(float64); ok {
							position = int(pos)
						}
					}
				}
			}
		case Gen2, Gen3, Gen4:
			if status, ok := device.Status.(map[string]interface{}); ok {
				if coverStatus, ok := status["cover:0"].(map[string]interface{}); ok {
					if coverState, ok := coverStatus["state"].(string); ok {
						state = coverState
					}
					if pos, ok := coverStatus["current_pos"].(float64); ok {
						position = int(pos)
					}
				}
			}
		}
	}

	// Default state if not determined
	if state == "" {
		state = "stopped"
	}

	attributes := map[string]interface{}{
		"device_generation": int(device.Generation),
		"model":             device.Model,
		"firmware_version":  device.FirmwareVersion,
		"ip_address":        device.IP,
		"mac_address":       device.MAC,
		"wifi_mode":         device.WiFiMode,
		"wifi_ssid":         device.WiFiSSID,
		"position":          position,
		"discovery_method":  device.DiscoveryMethod,
		"first_seen":        device.FirstSeen,
		"last_seen":         device.LastSeen,
		"supported_methods": device.SupportedMethods,
	}

	// Create metadata
	metadata := &types.PMAMetadata{
		Source:         types.SourceShelly,
		SourceEntityID: device.ID,
		SourceData:     attributes,
		LastSynced:     device.LastSeen,
		QualityScore:   0.90,
	}

	entity := &types.PMABaseEntity{
		ID:           device.ID,
		Type:         types.EntityTypeCover,
		FriendlyName: device.Name,
		Icon:         "mdi:window-shutter",
		State:        types.PMAEntityState(state),
		Attributes:   attributes,
		LastUpdated:  device.LastSeen,
		Available:    device.IsOnline,
		Metadata:     metadata,
	}

	return entity, nil
}

// executeRelayAction executes relay/switch actions
func (a *ShellyAdapter) executeRelayAction(ctx context.Context, device *EnhancedShellyDevice, action types.PMAControlAction) error {
	var state bool

	switch action.Action {
	case "turn_on":
		state = true
	case "turn_off":
		state = false
	case "toggle":
		// Get current state and toggle
		currentState := false
		if device.Status != nil {
			switch device.Generation {
			case Gen1:
				if status, ok := device.Status.(map[string]interface{}); ok {
					if relays, ok := status["relays"].([]interface{}); ok && len(relays) > 0 {
						if relay, ok := relays[0].(map[string]interface{}); ok {
							if ison, ok := relay["ison"].(bool); ok {
								currentState = ison
							}
						}
					}
				}
			case Gen2, Gen3, Gen4:
				if status, ok := device.Status.(map[string]interface{}); ok {
					if switchStatus, ok := status["switch:0"].(map[string]interface{}); ok {
						if output, ok := switchStatus["output"].(bool); ok {
							currentState = output
						}
					}
				}
			}
		}
		state = !currentState
	}

	// Extract timer parameter if present
	var timer *int
	if timerVal, ok := action.Parameters["timer"].(float64); ok {
		timerInt := int(timerVal)
		timer = &timerInt
	}

	return a.client.SetRelay(ctx, device.IP, 0, state, timer)
}

// executeLightAction executes light actions
func (a *ShellyAdapter) executeLightAction(ctx context.Context, device *EnhancedShellyDevice, action types.PMAControlAction) error {
	params := make(map[string]interface{})

	switch action.Action {
	case "turn_on":
		params["turn"] = true
	case "turn_off":
		params["turn"] = false
	case "toggle":
		// Get current state and toggle
		currentState := false
		if device.Status != nil {
			switch device.Generation {
			case Gen1:
				if status, ok := device.Status.(map[string]interface{}); ok {
					if lights, ok := status["lights"].([]interface{}); ok && len(lights) > 0 {
						if light, ok := lights[0].(map[string]interface{}); ok {
							if ison, ok := light["ison"].(bool); ok {
								currentState = ison
							}
						}
					}
				}
			case Gen2, Gen3, Gen4:
				if status, ok := device.Status.(map[string]interface{}); ok {
					if lightStatus, ok := status["light:0"].(map[string]interface{}); ok {
						if output, ok := lightStatus["output"].(bool); ok {
							currentState = output
						}
					}
				}
			}
		}
		params["turn"] = !currentState
	case "set_brightness":
		if brightness, ok := action.Parameters["brightness"].(float64); ok {
			params["brightness"] = int(brightness)
		}
	case "set_color":
		if red, ok := action.Parameters["red"].(float64); ok {
			params["red"] = int(red)
		}
		if green, ok := action.Parameters["green"].(float64); ok {
			params["green"] = int(green)
		}
		if blue, ok := action.Parameters["blue"].(float64); ok {
			params["blue"] = int(blue)
		}
	case "set_light":
		// Copy all parameters for generic light control
		for key, value := range action.Parameters {
			params[key] = value
		}
	}

	return a.client.SetLight(ctx, device.IP, 0, params)
}

// Helper functions

// getBoolEntityState converts a boolean to PMAEntityState
func getBoolEntityState(isOn bool) types.PMAEntityState {
	if isOn {
		return types.StateOn
	}
	return types.StateOff
}

// containsAny checks if a string contains any of the given substrings
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

// Additional utility methods for adapter management

// GetDiscoveredDevices returns all discovered devices
func (a *ShellyAdapter) GetDiscoveredDevices() []*EnhancedShellyDevice {
	return a.client.GetDiscoveredDevices()
}

// GetOnlineDevices returns only online devices
func (a *ShellyAdapter) GetOnlineDevices() []*EnhancedShellyDevice {
	return a.client.GetOnlineDevices()
}

// GetDevicesByGeneration returns devices filtered by generation
func (a *ShellyAdapter) GetDevicesByGeneration(gen DeviceGeneration) []*EnhancedShellyDevice {
	return a.client.GetDevicesByGeneration(gen)
}

// GetDiscoveryStats returns discovery statistics
func (a *ShellyAdapter) GetDiscoveryStats() map[string]interface{} {
	return a.client.GetDiscoveryStats()
}

// RefreshDevices forces a device refresh
func (a *ShellyAdapter) RefreshDevices(ctx context.Context) error {
	return a.syncDiscoveredDevices(ctx)
}

// RemoveDevice removes a device from tracking
func (a *ShellyAdapter) RemoveDevice(deviceID string) bool {
	return a.client.RemoveDevice(deviceID)
}

// ClearDevices clears all discovered devices
func (a *ShellyAdapter) ClearDevices() {
	a.client.ClearDevices()
}

// GetDeviceByIP returns a device by IP address
func (a *ShellyAdapter) GetDeviceByIP(ip string) *EnhancedShellyDevice {
	return a.client.GetDeviceByIP(ip)
}

// getCurrentDeviceState returns the current state of the device
func (a *ShellyAdapter) getCurrentDeviceState(device *EnhancedShellyDevice) string {
	if device.Status == nil {
		return "unknown"
	}

	switch device.Generation {
	case Gen1:
		if status, ok := device.Status.(map[string]interface{}); ok {
			// Check relay status first
			if relays, ok := status["relays"].([]interface{}); ok && len(relays) > 0 {
				if relay, ok := relays[0].(map[string]interface{}); ok {
					if ison, ok := relay["ison"].(bool); ok {
						if ison {
							return "on"
						}
						return "off"
					}
				}
			}
			// Check light status
			if lights, ok := status["lights"].([]interface{}); ok && len(lights) > 0 {
				if light, ok := lights[0].(map[string]interface{}); ok {
					if ison, ok := light["ison"].(bool); ok {
						if ison {
							return "on"
						}
						return "off"
					}
				}
			}
		}
	case Gen2, Gen3, Gen4:
		if status, ok := device.Status.(map[string]interface{}); ok {
			// Check switch status first
			if switchStatus, ok := status["switch:0"].(map[string]interface{}); ok {
				if output, ok := switchStatus["output"].(bool); ok {
					if output {
						return "on"
					}
					return "off"
				}
			}
			// Check light status
			if lightStatus, ok := status["light:0"].(map[string]interface{}); ok {
				if output, ok := lightStatus["output"].(bool); ok {
					if output {
						return "on"
					}
					return "off"
				}
			}
		}
	}

	return "unknown"
}

// RefreshEntityState refreshes a specific device's state
func (a *ShellyAdapter) RefreshEntityState(ctx context.Context, entityID string) error {
	device := a.client.GetDeviceByIP(entityID)
	if device == nil {
		// Try to find by device ID
		devices := a.client.GetDiscoveredDevices()
		for _, d := range devices {
			if d.ID == entityID {
				device = d
				break
			}
		}
	}

	if device == nil {
		return fmt.Errorf("device not found: %s", entityID)
	}

	return a.updateDeviceStatus(ctx, device)
}
