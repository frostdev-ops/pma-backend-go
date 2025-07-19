package shelly

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/frostdev-ops/pma-backend-go/internal/core/types"
	"github.com/sirupsen/logrus"
)

// ShellyAdapter implements the PMAAdapter interface for Shelly devices
type ShellyAdapter struct {
	client            *ShellyClient
	devices           map[string]*ShellyDevice
	logger            *logrus.Logger
	config            ShellyAdapterConfig
	mutex             sync.RWMutex
	connected         bool
	lastSyncTime      time.Time
	startTime         time.Time
	actionsExecuted   int
	successfulActions int
	failedActions     int
	syncErrors        int
}

// ShellyAdapterConfig holds configuration for the Shelly adapter
type ShellyAdapterConfig struct {
	Username      string        `json:"username"`
	Password      string        `json:"password"`
	PollInterval  time.Duration `json:"poll_interval"`
	AutoReconnect bool          `json:"auto_reconnect"`
	Devices       []string      `json:"devices"` // IP addresses or hostnames
}

// ShellyDevice represents a Shelly device
type ShellyDevice struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Type     string            `json:"type"`
	Host     string            `json:"host"`
	Info     *ShellyDeviceInfo `json:"info"`
	Status   *ShellyStatus     `json:"status"`
	Settings *ShellySettings   `json:"settings"`
	LastSeen time.Time         `json:"last_seen"`
}

// NewShellyAdapter creates a new Shelly adapter
func NewShellyAdapter(config ShellyAdapterConfig, logger *logrus.Logger) *ShellyAdapter {
	// Set defaults
	if config.PollInterval == 0 {
		config.PollInterval = 30 * time.Second
	}

	return &ShellyAdapter{
		client:    NewShellyClient(config.Username, config.Password, logger),
		devices:   make(map[string]*ShellyDevice),
		logger:    logger,
		config:    config,
		startTime: time.Now(),
	}
}

// ========================================
// PMAAdapter Interface Implementation
// ========================================

// GetID returns the unique identifier for this adapter instance
func (a *ShellyAdapter) GetID() string {
	return "shelly_adapter"
}

// GetSourceType returns the source type for Shelly
func (a *ShellyAdapter) GetSourceType() types.PMASourceType {
	return types.SourceShelly
}

// GetName returns the adapter name
func (a *ShellyAdapter) GetName() string {
	return "Shelly IoT Adapter"
}

// GetVersion returns the adapter version
func (a *ShellyAdapter) GetVersion() string {
	return "1.0.0"
}

// Connect establishes connection to Shelly devices
func (a *ShellyAdapter) Connect(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.logger.Info("Connecting to Shelly devices...")

	// Discover devices on the configured hosts
	for _, host := range a.config.Devices {
		device, err := a.discoverDevice(ctx, host)
		if err != nil {
			a.logger.WithError(err).Warnf("Failed to discover Shelly device at %s", host)
			continue
		}
		a.devices[device.ID] = device
		a.logger.WithField("device_id", device.ID).Infof("Discovered Shelly device: %s", device.Name)
	}

	a.connected = true
	a.logger.Infof("Successfully connected to %d Shelly devices", len(a.devices))
	return nil
}

// Disconnect closes connections to Shelly devices
func (a *ShellyAdapter) Disconnect(ctx context.Context) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.connected = false
	a.logger.Info("Disconnected from Shelly devices")
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

// ConvertEntity converts a Shelly device to PMA entity
func (a *ShellyAdapter) ConvertEntity(sourceEntity interface{}) (types.PMAEntity, error) {
	device, ok := sourceEntity.(*ShellyDevice)
	if !ok {
		return nil, fmt.Errorf("unsupported Shelly entity type: %T", sourceEntity)
	}

	return a.convertDeviceToPMAEntity(device)
}

// ConvertEntities converts multiple Shelly devices to PMA entities
func (a *ShellyAdapter) ConvertEntities(sourceEntities []interface{}) ([]types.PMAEntity, error) {
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

// ConvertRoom converts a Shelly room to PMA room (not supported)
func (a *ShellyAdapter) ConvertRoom(sourceRoom interface{}) (*types.PMARoom, error) {
	return nil, fmt.Errorf("room conversion not supported for Shelly devices")
}

// ConvertArea converts a Shelly area to PMA area (not supported)
func (a *ShellyAdapter) ConvertArea(sourceArea interface{}) (*types.PMAArea, error) {
	return nil, fmt.Errorf("area conversion not supported for Shelly devices")
}

// ExecuteAction executes control actions on Shelly devices
func (a *ShellyAdapter) ExecuteAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	return a.executeShellyAction(ctx, action)
}

// SyncEntities synchronizes entities from Shelly devices
func (a *ShellyAdapter) SyncEntities(ctx context.Context) ([]types.PMAEntity, error) {
	if !a.connected {
		return nil, fmt.Errorf("adapter not connected")
	}

	a.mutex.RLock()
	devices := make([]*ShellyDevice, 0, len(a.devices))
	for _, device := range a.devices {
		devices = append(devices, device)
	}
	a.mutex.RUnlock()

	// Update device status
	for _, device := range devices {
		if err := a.updateDeviceStatus(ctx, device); err != nil {
			a.logger.WithError(err).Warnf("Failed to update status for device %s", device.ID)
		}
	}

	// Convert to interface slice
	sourceEntities := make([]interface{}, len(devices))
	for i, device := range devices {
		sourceEntities[i] = device
	}

	// Convert to PMA entities
	pmaEntities, err := a.ConvertEntities(sourceEntities)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Shelly devices: %w", err)
	}

	a.lastSyncTime = time.Now()
	return pmaEntities, nil
}

// SyncRooms synchronizes rooms from Shelly (not supported)
func (a *ShellyAdapter) SyncRooms(ctx context.Context) ([]*types.PMARoom, error) {
	return []*types.PMARoom{}, nil
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

// GetSupportedEntityTypes returns entity types supported by Shelly adapter
func (a *ShellyAdapter) GetSupportedEntityTypes() []types.PMAEntityType {
	return []types.PMAEntityType{
		types.EntityTypeSwitch,
		types.EntityTypeLight,
		types.EntityTypeSensor,
		types.EntityTypeDevice,
	}
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

// SupportsRealtime returns whether Shelly supports real-time updates
func (a *ShellyAdapter) SupportsRealtime() bool {
	return false // Shelly uses HTTP polling
}

// GetHealth returns adapter health information
func (a *ShellyAdapter) GetHealth() *types.AdapterHealth {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	issues := []string{}
	if !a.connected {
		issues = append(issues, "Not connected to Shelly devices")
	}

	// Check device connectivity
	unreachableCount := 0
	for _, device := range a.devices {
		if time.Since(device.LastSeen) > 5*time.Minute {
			unreachableCount++
		}
	}

	if unreachableCount > 0 {
		issues = append(issues, fmt.Sprintf("%d devices unreachable", unreachableCount))
	}

	return &types.AdapterHealth{
		IsHealthy:       len(issues) == 0,
		LastHealthCheck: time.Now(),
		Issues:          issues,
		ResponseTime:    200 * time.Millisecond, // Typical HTTP response time
		ErrorRate:       a.calculateErrorRate(),
		Details: map[string]interface{}{
			"connected":         a.connected,
			"device_count":      len(a.devices),
			"unreachable_count": unreachableCount,
		},
	}
}

// GetMetrics returns adapter performance metrics
func (a *ShellyAdapter) GetMetrics() *types.AdapterMetrics {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	var lastSync *time.Time
	if !a.lastSyncTime.IsZero() {
		lastSync = &a.lastSyncTime
	}

	return &types.AdapterMetrics{
		EntitiesManaged:     len(a.devices),
		RoomsManaged:        0,
		ActionsExecuted:     int64(a.actionsExecuted),
		SuccessfulActions:   int64(a.successfulActions),
		FailedActions:       int64(a.failedActions),
		AverageResponseTime: 200 * time.Millisecond,
		LastSync:            lastSync,
		SyncErrors:          a.syncErrors,
		Uptime:              time.Since(a.startTime),
	}
}

// Helper methods
func (a *ShellyAdapter) calculateErrorRate() float64 {
	if a.actionsExecuted == 0 {
		return 0.0
	}
	return float64(a.failedActions) / float64(a.actionsExecuted)
}

// discoverDevice discovers a Shelly device at the given host
func (a *ShellyAdapter) discoverDevice(ctx context.Context, host string) (*ShellyDevice, error) {
	// Get device info
	info, err := a.client.GetDeviceInfo(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	// Get device status
	status, err := a.client.GetDeviceStatus(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("failed to get device status: %w", err)
	}

	// Get device settings
	settings, err := a.client.GetDeviceSettings(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("failed to get device settings: %w", err)
	}

	device := &ShellyDevice{
		ID:       fmt.Sprintf("shelly_%s", info.MAC),
		Name:     info.Name,
		Type:     info.Type,
		Host:     host,
		Info:     info,
		Status:   status,
		Settings: settings,
		LastSeen: time.Now(),
	}

	return device, nil
}

// updateDeviceStatus updates the status of a Shelly device
func (a *ShellyAdapter) updateDeviceStatus(ctx context.Context, device *ShellyDevice) error {
	status, err := a.client.GetDeviceStatus(ctx, device.Host)
	if err != nil {
		return fmt.Errorf("failed to update device status: %w", err)
	}

	a.mutex.Lock()
	device.Status = status
	device.LastSeen = time.Now()
	a.mutex.Unlock()

	return nil
}

// convertDeviceToPMAEntity converts a Shelly device to appropriate PMA entity
func (a *ShellyAdapter) convertDeviceToPMAEntity(device *ShellyDevice) (types.PMAEntity, error) {
	// Determine entity type based on device capabilities
	if len(device.Status.Relays) > 0 {
		return a.convertToSwitchEntity(device)
	}
	if len(device.Status.Lights) > 0 || len(device.Status.Dimmers) > 0 {
		return a.convertToLightEntity(device)
	}
	if device.Status.Temperature != nil || device.Status.Humidity != nil {
		return a.convertToSensorEntity(device)
	}

	// Default to generic device
	return a.convertToDeviceEntity(device)
}

// convertToSwitchEntity converts Shelly device to PMA switch entity
func (a *ShellyAdapter) convertToSwitchEntity(device *ShellyDevice) (types.PMAEntity, error) {
	relay := device.Status.Relays[0] // Use first relay

	entity := &types.PMABaseEntity{
		ID:           device.ID,
		Type:         types.EntityTypeSwitch,
		FriendlyName: device.Name,
		Icon:         "mdi:toggle-switch",
		State:        a.mapShellyState(relay.IsOn),
		Attributes:   a.convertSwitchAttributes(device),
		LastUpdated:  time.Now(),
		Capabilities: []types.PMACapability{},
		Metadata: &types.PMAMetadata{
			Source:         types.SourceShelly,
			SourceEntityID: device.ID,
			SourceData:     a.convertDeviceSourceData(device),
			LastSynced:     time.Now(),
			QualityScore:   0.90,
		},
		Available: time.Since(device.LastSeen) < 5*time.Minute,
	}

	return entity, nil
}

// convertToLightEntity converts Shelly device to PMA light entity
func (a *ShellyAdapter) convertToLightEntity(device *ShellyDevice) (types.PMAEntity, error) {
	var isOn bool
	var brightness int
	capabilities := []types.PMACapability{}

	if len(device.Status.Lights) > 0 {
		light := device.Status.Lights[0]
		isOn = light.IsOn
		brightness = light.Brightness
		capabilities = append(capabilities, types.CapabilityBrightness)

		// Check for color capabilities
		if light.Red >= 0 && light.Green >= 0 && light.Blue >= 0 {
			capabilities = append(capabilities, types.CapabilityColorable)
		}
	} else if len(device.Status.Dimmers) > 0 {
		dimmer := device.Status.Dimmers[0]
		isOn = dimmer.IsOn
		brightness = dimmer.Brightness
		capabilities = append(capabilities, types.CapabilityDimmable, types.CapabilityBrightness)
	}

	entity := &types.PMABaseEntity{
		ID:           device.ID,
		Type:         types.EntityTypeLight,
		FriendlyName: device.Name,
		Icon:         "mdi:lightbulb",
		State:        a.mapShellyState(isOn),
		Attributes:   a.convertLightAttributes(device, brightness),
		LastUpdated:  time.Now(),
		Capabilities: capabilities,
		Metadata: &types.PMAMetadata{
			Source:         types.SourceShelly,
			SourceEntityID: device.ID,
			SourceData:     a.convertDeviceSourceData(device),
			LastSynced:     time.Now(),
			QualityScore:   0.90,
		},
		Available: time.Since(device.LastSeen) < 5*time.Minute,
	}

	return entity, nil
}

// convertToSensorEntity converts Shelly device to PMA sensor entity
func (a *ShellyAdapter) convertToSensorEntity(device *ShellyDevice) (types.PMAEntity, error) {
	var sensorValue interface{}
	var unit string
	capabilities := []types.PMACapability{}

	// Priority: temperature > humidity > other sensors
	if device.Status.Temperature != nil {
		sensorValue = *device.Status.Temperature
		unit = "°C"
		capabilities = append(capabilities, types.CapabilityTemperature)
	} else if device.Status.Humidity != nil {
		sensorValue = *device.Status.Humidity
		unit = "%"
		capabilities = append(capabilities, types.CapabilityHumidity)
	}

	entity := &types.PMABaseEntity{
		ID:           device.ID,
		Type:         types.EntityTypeSensor,
		FriendlyName: device.Name,
		Icon:         "mdi:thermometer",
		State:        types.StateActive,
		Attributes:   a.convertSensorAttributes(device, sensorValue, unit),
		LastUpdated:  time.Now(),
		Capabilities: capabilities,
		Metadata: &types.PMAMetadata{
			Source:         types.SourceShelly,
			SourceEntityID: device.ID,
			SourceData:     a.convertDeviceSourceData(device),
			LastSynced:     time.Now(),
			QualityScore:   0.90,
		},
		Available: time.Since(device.LastSeen) < 5*time.Minute,
	}

	return entity, nil
}

// convertToDeviceEntity converts Shelly device to generic PMA device entity
func (a *ShellyAdapter) convertToDeviceEntity(device *ShellyDevice) (types.PMAEntity, error) {
	entity := &types.PMABaseEntity{
		ID:           device.ID,
		Type:         types.EntityTypeDevice,
		FriendlyName: device.Name,
		Icon:         "mdi:router-wireless",
		State:        types.StateActive,
		Attributes:   a.convertDeviceAttributes(device),
		LastUpdated:  time.Now(),
		Capabilities: []types.PMACapability{},
		Metadata: &types.PMAMetadata{
			Source:         types.SourceShelly,
			SourceEntityID: device.ID,
			SourceData:     a.convertDeviceSourceData(device),
			LastSynced:     time.Now(),
			QualityScore:   0.85,
		},
		Available: time.Since(device.LastSeen) < 5*time.Minute,
	}

	return entity, nil
}

// executeShellyAction executes control actions on Shelly devices
func (a *ShellyAdapter) executeShellyAction(ctx context.Context, action types.PMAControlAction) (*types.PMAControlResult, error) {
	device, exists := a.devices[action.EntityID]
	if !exists {
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

	// Execute action based on type
	var err error
	switch action.Action {
	case "turn_on":
		err = a.client.SetRelay(ctx, device.Host, 0, true, nil)
	case "turn_off":
		err = a.client.SetRelay(ctx, device.Host, 0, false, nil)
	case "toggle":
		// Get current state and toggle
		currentState := len(device.Status.Relays) > 0 && device.Status.Relays[0].IsOn
		err = a.client.SetRelay(ctx, device.Host, 0, !currentState, nil)
	case "set_brightness":
		if brightness, ok := action.Parameters["brightness"].(float64); ok {
			turnOn := true
			err = a.client.SetDimmer(ctx, device.Host, 0, int(brightness), &turnOn)
		} else {
			err = fmt.Errorf("invalid brightness parameter")
		}
	default:
		err = fmt.Errorf("unsupported action: %s", action.Action)
	}

	if err != nil {
		a.failedActions++
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

	// Update device status after action
	go func() {
		time.Sleep(100 * time.Millisecond) // Small delay
		a.updateDeviceStatus(ctx, device)
	}()

	return &types.PMAControlResult{
		Success:     true,
		EntityID:    action.EntityID,
		Action:      action.Action,
		ProcessedAt: time.Now(),
		Duration:    time.Since(startTime),
	}, nil
}

// Helper methods for attribute conversion
func (a *ShellyAdapter) mapShellyState(isOn bool) types.PMAEntityState {
	if isOn {
		return types.StateOn
	}
	return types.StateOff
}

func (a *ShellyAdapter) convertSwitchAttributes(device *ShellyDevice) map[string]interface{} {
	attrs := make(map[string]interface{})
	attrs["device_type"] = "switch"
	attrs["host"] = device.Host
	attrs["mac"] = device.Info.MAC

	if len(device.Status.Relays) > 0 {
		relay := device.Status.Relays[0]
		attrs["has_timer"] = relay.HasTimer
		attrs["overpower"] = relay.Overpower
		attrs["over_temperature"] = relay.OverTemperature
	}

	return attrs
}

func (a *ShellyAdapter) convertLightAttributes(device *ShellyDevice, brightness int) map[string]interface{} {
	attrs := make(map[string]interface{})
	attrs["device_type"] = "light"
	attrs["host"] = device.Host
	attrs["mac"] = device.Info.MAC
	attrs["brightness"] = brightness

	if len(device.Status.Lights) > 0 {
		light := device.Status.Lights[0]
		attrs["red"] = light.Red
		attrs["green"] = light.Green
		attrs["blue"] = light.Blue
		attrs["white"] = light.White
		attrs["temp"] = light.Temp
	}

	return attrs
}

func (a *ShellyAdapter) convertSensorAttributes(device *ShellyDevice, value interface{}, unit string) map[string]interface{} {
	attrs := make(map[string]interface{})
	attrs["device_type"] = "sensor"
	attrs["host"] = device.Host
	attrs["mac"] = device.Info.MAC
	attrs["value"] = value
	attrs["unit"] = unit

	if device.Status.Temperature != nil {
		attrs["temperature"] = *device.Status.Temperature
	}
	if device.Status.Humidity != nil {
		attrs["humidity"] = *device.Status.Humidity
	}
	if device.Status.Pressure != nil {
		attrs["pressure"] = *device.Status.Pressure
	}

	return attrs
}

func (a *ShellyAdapter) convertDeviceAttributes(device *ShellyDevice) map[string]interface{} {
	attrs := make(map[string]interface{})
	attrs["device_type"] = "device"
	attrs["host"] = device.Host
	attrs["mac"] = device.Info.MAC
	attrs["model"] = device.Info.Model
	attrs["fw_id"] = device.Info.FwID
	attrs["version"] = device.Info.Version

	return attrs
}

func (a *ShellyAdapter) convertDeviceSourceData(device *ShellyDevice) map[string]interface{} {
	return map[string]interface{}{
		"host":       device.Host,
		"mac":        device.Info.MAC,
		"type":       device.Info.Type,
		"model":      device.Info.Model,
		"fw_version": device.Info.Version,
		"last_seen":  device.LastSeen,
	}
}
